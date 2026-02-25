// templates/funcs.go
package templates

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/url"
	"strings"
	"sync"
)

// Funcs returns helpers available to all templates.
func Funcs() template.FuncMap {
	return template.FuncMap{
		// {{ "a b" | urlquery }} → "a+b"
		"urlquery": url.QueryEscape,
		// Mark a string as safe HTML (use sparingly!)
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },

		// Small quality-of-life helpers
		"lower":  strings.ToLower,
		"upper":  strings.ToUpper,
		"join":   strings.Join,
		"printf": func(f string, a ...any) string { return fmt.Sprintf(f, a...) },

		// JSON encoding for embedding data in templates
		// {{ .Data | toJSON }} → JSON string for use in JavaScript
		"toJSON": func(v any) template.JS {
			b, err := json.Marshal(v)
			if err != nil {
				return template.JS("null")
			}
			return template.JS(b)
		},
	}
}

var (
	customFuncsMu sync.RWMutex
	customFuncs   = template.FuncMap{}
)

// RegisterFunc adds a custom template function available to all templates.
// Call before Boot() — typically from init() in a resources package.
func RegisterFunc(name string, fn any) {
	customFuncsMu.Lock()
	defer customFuncsMu.Unlock()
	customFuncs[name] = fn
}
