// templates/engine.go
package templates

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// Engine compiles and holds templates from all registered Sets.
// It supports a "shared" set (common layout) and per-page clones.
type Engine struct {
	mu      sync.RWMutex
	funcs   template.FuncMap
	base    *template.Template            // compiled from "shared"
	byName  map[string]*template.Template // templateName -> compiled set containing it
	devMode bool
	Logger  *zap.Logger
}

// New creates a new Engine. dev=true can be used later to support hot-reload.
func New(dev bool) *Engine {
	return &Engine{
		funcs:   Funcs(),
		byName:  map[string]*template.Template{},
		devMode: dev,
	}
}

// Boot compiles all registered template Sets into the Engine.
// It must be called before Render/RenderSnippet, typically at startup.
func (e *Engine) Boot(logger *zap.Logger) error {
	e.Logger = logger

	// Merge custom template functions registered by the app.
	customFuncsMu.RLock()
	for k, v := range customFuncs {
		e.funcs[k] = v
	}
	customFuncsMu.RUnlock()

	sets := All()
	if len(sets) == 0 {
		if e.Logger != nil {
			e.Logger.Warn("no template sets registered")
		}
		return nil
	}

	// 1) Parse shared first
	var shared *Set
	var others []Set
	for i := range sets {
		s := sets[i]
		if s.Name == "shared" {
			shared = &s
		} else {
			others = append(others, s)
		}
	}
	if shared == nil {
		return fmt.Errorf("shared templates not registered")
	}

	core, err := e.parseFS(nil, shared.FS, shared.Patterns...)
	if err != nil {
		return fmt.Errorf("parse shared: %w", err)
	}
	e.base = core

	// 2) For each feature set, compile one clone *per page file*.
	for _, s := range others {
		if err := e.compileSetPerPage(s); err != nil {
			return fmt.Errorf("compile set %q: %w", s.Name, err)
		}
	}
	return nil
}

/*
compileSetPerPage clones the shared base for each page file in the set,
parses all files into that clone, but rewrites non-target files’
`define "content"` to a unique, ignored name. Then it indexes only the
template names that are actually defined by the target file (so
foo_list resolves to the *list* page’s clone, not the edit page’s).
*/
func (e *Engine) compileSetPerPage(s Set) error {
	allFiles, err := globAll(s.FS, s.Patterns)
	if err != nil {
		return err
	}
	if len(allFiles) == 0 {
		if e.Logger != nil {
			e.Logger.Warn("no templates matched", zap.String("set", s.Name))
		}
		return nil
	}
	// Stable order
	sort.Strings(allFiles)

	for _, pagePath := range allFiles {
		pageSrcBytes, rerr := fs.ReadFile(s.FS, pagePath)
		if rerr != nil {
			return fmt.Errorf("read %s: %w", pagePath, rerr)
		}
		pageSrc := string(pageSrcBytes)

		// Names owned by this file (entrypoints + partials it defines)
		owned := extractDefineNames(pageSrc)
		delete(owned, "content") // never index the "content" template

		baseClone, err := e.base.Clone()
		if err != nil {
			return fmt.Errorf("clone base: %w", err)
		}

		// Parse each file; only the target keeps its "content" name.
		for _, p := range allFiles {
			src, rerr := fs.ReadFile(s.FS, p)
			if rerr != nil {
				return fmt.Errorf("read %s: %w", p, rerr)
			}
			text := string(src)
			if p != pagePath {
				// Rewrite other files’ `define \"content\"` -> unique ignored name.
				text = rewriteContentDefine(text, ignoredContentName(p))
			}
			if _, perr := baseClone.Funcs(e.funcs).Parse(text); perr != nil {
				return fmt.Errorf("parse %s (for %s): %w", p, pagePath, perr)
			}
		}

		// Index only the names *owned* by this file to this clone.
		// This prevents other clones from stealing the entrypoint.
		e.mu.Lock()
		for name := range owned {
			e.byName[name] = baseClone
		}
		e.mu.Unlock()

		if e.Logger != nil {
			e.Logger.Info("template page compiled",
				zap.String("set", s.Name),
				zap.String("page", filepath.Base(pagePath)))
		}
	}
	return nil
}

// --- helpers for parsing & name extraction ---

var (
	reContentDefine = regexp.MustCompile(`{{\s*define\s+"content"\s*}}`)
	reDefineName    = regexp.MustCompile(`{{\s*define\s+"([^"]+)"`)
)

func rewriteContentDefine(src string, newName string) string {
	// Rename only the template header; {{ end }} remains generic and still closes.
	return reContentDefine.ReplaceAllString(src, fmt.Sprintf(`{{ define "%s" }}`, newName))
}

func ignoredContentName(path string) string {
	// e.g., templates/admin_organization_list.gohtml -> _content_ignored_admin_organization_list
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	return "_content_ignored_" + base
}

func extractDefineNames(src string) map[string]struct{} {
	out := make(map[string]struct{})
	m := reDefineName.FindAllStringSubmatch(src, -1)
	for _, g := range m {
		if len(g) >= 2 {
			out[g[1]] = struct{}{}
		}
	}
	return out
}

// parseFS reads & parses all files matching patterns into t (or new root if t==nil).
// This is used only for the shared set.
func (e *Engine) parseFS(t *template.Template, filesystem fs.FS, patterns ...string) (*template.Template, error) {
	root := t
	if root == nil {
		root = template.New("root").Funcs(e.funcs)
	} else {
		root = root.Funcs(e.funcs)
	}

	for _, pat := range patterns {
		matches, err := fs.Glob(filesystem, pat)
		if err != nil {
			return nil, err
		}
		sort.Strings(matches)
		for _, path := range matches {
			b, err := fs.ReadFile(filesystem, path)
			if err != nil {
				return nil, err
			}
			if _, err = root.Parse(string(b)); err != nil {
				return nil, fmt.Errorf("parse %s: %w", path, err)
			}
		}
	}
	return root, nil
}

func globAll(filesystem fs.FS, patterns []string) ([]string, error) {
	seen := make(map[string]struct{})
	var out []string
	for _, pat := range patterns {
		matches, err := fs.Glob(filesystem, pat)
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			if _, ok := seen[m]; ok {
				continue
			}
			seen[m] = struct{}{}
			out = append(out, m)
		}
	}
	return out, nil
}

// Render executes a top-level template by name using this engine.
// Usually you’ll call the package-level Render helpers in adapter.go.
func (e *Engine) Render(w Writer, r Request, name string, data any) error {
	e.mu.RLock()
	t, ok := e.byName[name]
	e.mu.RUnlock()
	if !ok {
		return fmt.Errorf("template %q not found", name)
	}

	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, name, data); err != nil {
		return err
	}
	_, _ = w.Write(buf.Bytes())
	return nil
}

// RenderSnippet executes a partial template by name (e.g., a table or fragment).
func (e *Engine) RenderSnippet(w Writer, name string, data any) error {
	e.mu.RLock()
	t, ok := e.byName[name]
	e.mu.RUnlock()
	if !ok {
		return fmt.Errorf("snippet %q not found", name)
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, name, data); err != nil {
		return err
	}
	_, _ = w.Write(buf.Bytes())
	return nil
}

// RenderContent executes the "content" block associated with the given entry template.
func (e *Engine) RenderContent(w Writer, entry string, data any) error {
	e.mu.RLock()
	t, ok := e.byName[entry]
	e.mu.RUnlock()
	if !ok {
		return fmt.Errorf("template %q not found", entry)
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "content", data); err != nil {
		return err
	}
	_, _ = w.Write(buf.Bytes())
	return nil
}

// Writer and Request interfaces let us avoid importing net/http here.
type Writer interface{ Write([]byte) (int, error) }
type Request interface{}
