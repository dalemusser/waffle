// templates/adapter.go
package templates

import (
	"net/http"

	"go.uber.org/zap"
)

var (
	engine *Engine
	logger *zap.Logger
)

// UseEngine installs the engine and logger used by the helper Render functions.
func UseEngine(e *Engine, l *zap.Logger) {
	engine = e
	logger = l
}

// Render executes a full page (entry template that calls layout).
func Render(w http.ResponseWriter, r *http.Request, name string, data any) {
	if engine == nil {
		if logger != nil {
			logger.Error("render called before engine installed", zap.String("name", name))
		}
		http.Error(w, "template exec error", http.StatusInternalServerError)
		return
	}
	if err := engine.Render(w, r, name, data); err != nil {
		if logger != nil {
			logger.Error("template render failed", zap.String("name", name), zap.Error(err))
		}
		http.Error(w, "template exec error", http.StatusInternalServerError)
	}
}

// RenderSnippet executes a partial by name (e.g., "groups_table").
func RenderSnippet(w http.ResponseWriter, name string, data any) {
	if engine == nil {
		if logger != nil {
			logger.Error("render snippet called before engine installed", zap.String("name", name))
		}
		http.Error(w, "template exec error", http.StatusInternalServerError)
		return
	}
	if err := engine.RenderSnippet(w, name, data); err != nil {
		if logger != nil {
			logger.Error("snippet render failed", zap.String("name", name), zap.Error(err))
		}
		http.Error(w, "template exec error", http.StatusInternalServerError)
	}
}

// RenderAutoMap picks a snippet based on HX-Target; if HX-Target is "content",
// it renders the page's content-only block. Otherwise it renders the full page.
func RenderAutoMap(w http.ResponseWriter, r *http.Request, page string, targets map[string]string, data any) {
	if engine == nil {
		if logger != nil {
			logger.Error("render auto called before engine installed", zap.String("page", page))
		}
		http.Error(w, "template exec error", http.StatusInternalServerError)
		return
	}

	// HTMX handling: if this is an HTMX request, look at HX-Target.
	if r.Header.Get("HX-Request") != "" {
		hxTarget := r.Header.Get("HX-Target")

		// First, explicit target->snippet mapping
		if snip, ok := targets[hxTarget]; ok && snip != "" {
			if err := engine.RenderSnippet(w, snip, data); err != nil {
				if logger != nil {
					logger.Error("snippet render failed", zap.String("snippet", snip), zap.Error(err))
				}
				http.Error(w, "template exec error", http.StatusInternalServerError)
			}
			return
		}

		// Fallback: if the target is the main page body, render just the content block
		if hxTarget == "content" {
			if err := engine.RenderContent(w, page, data); err != nil {
				if logger != nil {
					logger.Error("content render failed", zap.String("page", page), zap.Error(err))
				}
				http.Error(w, "template exec error", http.StatusInternalServerError)
			}
			return
		}
	}

	// Not HTMX (or no special mapping) → full page with layout
	if err := engine.Render(w, r, page, data); err != nil {
		if logger != nil {
			logger.Error("template render failed", zap.String("page", page), zap.Error(err))
		}
		http.Error(w, "template exec error", http.StatusInternalServerError)
	}
}

// Convenience for the common single-table swap case.
func RenderAuto(w http.ResponseWriter, r *http.Request, page, tableSnippet, targetID string, data any) {
	RenderAutoMap(w, r, page, map[string]string{targetID: tableSnippet}, data)
}
