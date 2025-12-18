// pantry/httpnav/httpnav.go
package httpnav

import (
	"net/http"
	"net/url"
	"strings"
)

// ResolveBackURL returns a safe local back URL from ?return=, form return,
// or Referer. Fallback must be a local path (e.g., "/groups").
func ResolveBackURL(r *http.Request, fallback string) string {
	// Normalize fallback to something safe
	if !isLocal(fallback) {
		fallback = "/"
	}

	// 1) Explicit query param takes top priority
	if ret := strings.TrimSpace(r.URL.Query().Get("return")); isLocal(ret) {
		return ret
	}

	// 2) POST/PUT form fields (works for urlencoded and multipart)
	// FormValue() will ParseForm() as needed.
	if ret := strings.TrimSpace(r.FormValue("return")); isLocal(ret) {
		return ret
	}

	// 3) Safe Referer (same host or relative), preserving query string
	if ref := r.Header.Get("Referer"); ref != "" {
		if u, err := url.Parse(ref); err == nil {
			sameHostAndLocal :=
				(u.Scheme == "" && strings.HasPrefix(u.Path, "/")) ||
					(u.Host == r.Host && strings.HasPrefix(u.Path, "/"))
			if sameHostAndLocal {
				if p := u.Path + pickQuery(u.RawQuery); isLocal(p) {
					return p
				}
			}
		}
	}

	// 4) Fallback
	return fallback
}

func CurrentPath(r *http.Request) string {
	p := r.URL.Path
	if q := r.URL.RawQuery; q != "" {
		p += "?" + q
	}
	return p
}

func isLocal(p string) bool {
	return p != "" && strings.HasPrefix(p, "/")
}

func pickQuery(raw string) string {
	if raw == "" {
		return ""
	}
	return "?" + raw
}
