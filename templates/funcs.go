// templates/funcs.go
package templates

import (
	"fmt"
	"html/template"
	"net/url"
	"strings"
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
	}
}
