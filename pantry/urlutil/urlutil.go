// pantry/urlutil/urlutil.go
package urlutil

import (
	"net/url"
	"path"
	"strings"
)

// DefaultExcludedPaths contains paths that should typically not be used as
// return URLs after login. These prevent redirect loops and immediate logout
// scenarios where a user logs in and is immediately redirected to logout.
var DefaultExcludedPaths = []string{"/logout", "/login"}

// SafeReturn validates and sanitizes a same-origin redirect path.
// It accepts only absolute paths (no scheme/host), rejects CR/LF and backslashes,
// optionally rejects targets containing a specified resource identifier (badID),
// and returns fallback if validation fails.

// SafeReturn is for validating and sanitizing redirect targets
// to prevent open redirect vulnerabilities and header injection attacks. All
// redirect operations should use this to ensure only safe, intended
// redirect paths are followed.
//
// Security Considerations:
//
// Open Redirect Vulnerability: Attackers can manipulate redirect parameters to
// send users to malicious third-party sites. This package validates that all
// redirect targets are relative paths (no scheme or host components).
//
// Header Injection: Carriage return (CR) and line feed (LF) characters can be
// used to inject arbitrary HTTP headers or split responses. All functions reject
// URLs containing these characters.
//
// Resource Deletion Bypass: When deleting a resource, the redirect target should
// not reference the deleted ID. For example, redirecting to a detail page after
// deletion could cause confusion or expose a 404. This package supports filtering
// out such cases.
//
// Usage:
//
// Typical usage in a delete handler:
//
//	ret := r.FormValue("return")
//	safeRedirect := urlutil.SafeReturn(ret, deletedID, "/admin/items")
//	http.Redirect(w, r, safeRedirect, http.StatusSeeOther)
//
// This ensures the redirect is:
//   - A relative path (not an external URL)
//   - Free of header injection characters
//   - Does not reference the deleted resource ID
//   - Not in the DefaultExcludedPaths list (e.g., /logout, /login)
//   - Falls back to a safe default if any validation fails
//
// To allow all paths without exclusions, use SafeReturnRaw.
// To specify custom exclusions, use SafeReturnExcluding.
func SafeReturn(ret, badID, fallback string) string {
	return SafeReturnExcluding(ret, badID, fallback, DefaultExcludedPaths)
}

// SafeReturnExcluding is like SafeReturn but with custom excluded paths.
// Pass nil or empty slice for no path exclusions.
//
// The exclusion check matches:
//   - Exact path: "/logout" matches "/logout"
//   - Path with query string: "/logout" matches "/logout?foo=bar"
//
// It does NOT match partial path names (e.g., "/logout" does not match "/logout-help").
func SafeReturnExcluding(ret, badID, fallback string, excluded []string) string {
	ret = strings.TrimSpace(ret)
	if ret == "" {
		return fallback
	}
	// Reject characters that can break headers.
	if strings.ContainsAny(ret, "\r\n") {
		return fallback
	}
	// No backslashes to avoid ambiguous interpretations on some stacks.
	if strings.ContainsRune(ret, '\\') {
		return fallback
	}
	// Must be an absolute path (same-origin) and not scheme-relative.
	if !strings.HasPrefix(ret, "/") || strings.HasPrefix(ret, "//") {
		return fallback
	}

	// Normalize the path; keep it anchored.
	clean := path.Clean(ret)
	// path.Clean keeps a leading "/" if present. Double-check anyway.
	if !strings.HasPrefix(clean, "/") {
		return fallback
	}

	// Parse defensively and ensure it's not absolute.
	if u, err := url.Parse(clean); err != nil || u.IsAbs() || u.Host != "" || u.Scheme != "" {
		return fallback
	}

	if badID != "" && pathHasSegment(clean, badID) {
		return fallback
	}

	// Check excluded paths
	for _, ex := range excluded {
		if clean == ex || strings.HasPrefix(clean, ex+"?") {
			return fallback
		}
	}

	return clean
}

// SafeReturnRaw is like SafeReturn but with no path exclusions.
// Use this only when you explicitly need to allow paths like /logout as return URLs.
func SafeReturnRaw(ret, badID, fallback string) string {
	return SafeReturnExcluding(ret, badID, fallback, nil)
}

// IsValidAbsHTTPURL reports whether s is an absolute http(s) URL with a host,
// no credentials in the authority, and no CR/LF.
//
// A valid URL must:
//   - Be non-empty after trimming whitespace
//   - Not contain newline or carriage return characters
//   - Parse successfully as a URL
//   - Have a non-empty host component
//   - Use either "http" or "https" scheme
//
// This function is useful for validating user-provided URLs in forms, such as
// resource launch URLs, before storing them in the database. It prevents common
// injection vectors and ensures only publicly accessible HTTP(S) resources
// are accepted (not file://, data://, or other schemes).
//
// Examples:
//
//	IsValidAbsHTTPURL("https://example.com")           // true
//	IsValidAbsHTTPURL("http://example.com/path")       // true
//	IsValidAbsHTTPURL("https://example.com:8080/path") // true
//	IsValidAbsHTTPURL("example.com")                   // false (no scheme)
//	IsValidAbsHTTPURL("ftp://example.com")             // false (invalid scheme)
//	IsValidAbsHTTPURL("https://example.com\r\n")       // false (contains whitespace chars)
//	IsValidAbsHTTPURL("")                              // false (empty)
//	IsValidAbsHTTPURL("   ")                           // false (whitespace only)
func IsValidAbsHTTPURL(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, "\r\n") {
		return false
	}
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	// Must be absolute http(s) with a host.
	if u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	// Disallow user:pass@host credentials in authority.
	if u.User != nil {
		return false
	}
	return true
}

func pathHasSegment(p, seg string) bool {
	if seg == "" {
		return false
	}
	// Normalize
	p = path.Clean(p)
	// Split keeps empty at root? path.Clean("/") => "/"
	parts := strings.Split(strings.TrimPrefix(p, "/"), "/")
	for _, s := range parts {
		if s == seg {
			return true
		}
	}
	return false
}
