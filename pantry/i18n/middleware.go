// i18n/middleware.go
package i18n

import (
	"net/http"
	"strings"
)

// LocaleDetector detects the locale from an HTTP request.
type LocaleDetector func(r *http.Request) string

// MiddlewareConfig configures the i18n middleware.
type MiddlewareConfig struct {
	// Bundle is the translation bundle to use.
	Bundle *Bundle

	// Detectors are locale detection strategies, tried in order.
	// First non-empty result is used.
	// Default: QueryDetector, CookieDetector, HeaderDetector
	Detectors []LocaleDetector

	// CookieName is the name of the locale cookie.
	// Default: "lang"
	CookieName string

	// QueryParam is the query parameter for locale.
	// Default: "lang"
	QueryParam string

	// SetCookie determines whether to set a cookie when locale is detected.
	// Default: false
	SetCookie bool

	// CookieMaxAge is the max age for the locale cookie in seconds.
	// Default: 365 days
	CookieMaxAge int

	// CookiePath is the path for the locale cookie.
	// Default: "/"
	CookiePath string
}

// DefaultMiddlewareConfig returns sensible defaults.
func DefaultMiddlewareConfig(bundle *Bundle) MiddlewareConfig {
	return MiddlewareConfig{
		Bundle:       bundle,
		CookieName:   "lang",
		QueryParam:   "lang",
		CookieMaxAge: 365 * 24 * 60 * 60, // 1 year
		CookiePath:   "/",
	}
}

// Middleware creates HTTP middleware that detects locale and adds a Localizer to context.
func Middleware(cfg MiddlewareConfig) func(http.Handler) http.Handler {
	if cfg.CookieName == "" {
		cfg.CookieName = "lang"
	}
	if cfg.QueryParam == "" {
		cfg.QueryParam = "lang"
	}
	if cfg.CookieMaxAge == 0 {
		cfg.CookieMaxAge = 365 * 24 * 60 * 60
	}
	if cfg.CookiePath == "" {
		cfg.CookiePath = "/"
	}

	// Default detectors
	if len(cfg.Detectors) == 0 {
		cfg.Detectors = []LocaleDetector{
			QueryDetector(cfg.QueryParam),
			CookieDetector(cfg.CookieName),
			HeaderDetector(),
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			locale := detectLocale(r, cfg)

			// Validate locale exists, fall back to default
			if !cfg.Bundle.HasLocale(locale) {
				// Try base locale
				if idx := strings.IndexAny(locale, "-_"); idx > 0 {
					base := locale[:idx]
					if cfg.Bundle.HasLocale(base) {
						locale = base
					} else {
						locale = cfg.Bundle.DefaultLocale()
					}
				} else {
					locale = cfg.Bundle.DefaultLocale()
				}
			}

			// Set cookie if requested
			if cfg.SetCookie {
				http.SetCookie(w, &http.Cookie{
					Name:     cfg.CookieName,
					Value:    locale,
					Path:     cfg.CookiePath,
					MaxAge:   cfg.CookieMaxAge,
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
			}

			// Add localizer to context
			localizer := cfg.Bundle.Localizer(locale)
			ctx := WithLocalizer(r.Context(), localizer)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// detectLocale tries each detector until one returns a result.
func detectLocale(r *http.Request, cfg MiddlewareConfig) string {
	for _, detector := range cfg.Detectors {
		if locale := detector(r); locale != "" {
			return locale
		}
	}
	return cfg.Bundle.DefaultLocale()
}

// QueryDetector detects locale from a query parameter.
func QueryDetector(param string) LocaleDetector {
	return func(r *http.Request) string {
		return r.URL.Query().Get(param)
	}
}

// CookieDetector detects locale from a cookie.
func CookieDetector(name string) LocaleDetector {
	return func(r *http.Request) string {
		cookie, err := r.Cookie(name)
		if err != nil {
			return ""
		}
		return cookie.Value
	}
}

// HeaderDetector detects locale from Accept-Language header.
func HeaderDetector() LocaleDetector {
	return func(r *http.Request) string {
		return ParseAcceptLanguage(r.Header.Get("Accept-Language"))
	}
}

// PathDetector detects locale from URL path prefix.
// Example: /en/page, /fr/page
func PathDetector() LocaleDetector {
	return func(r *http.Request) string {
		path := r.URL.Path
		if len(path) < 3 {
			return ""
		}

		// Check for /xx/ or /xx-XX/ pattern
		if path[0] == '/' {
			path = path[1:]
		}

		idx := strings.Index(path, "/")
		if idx == -1 {
			idx = len(path)
		}

		candidate := path[:idx]

		// Validate it looks like a locale (2-5 chars)
		if len(candidate) >= 2 && len(candidate) <= 5 {
			return candidate
		}

		return ""
	}
}

// SessionDetector creates a detector that reads from a session.
// sessionGetter should return the session value for the locale key.
func SessionDetector(sessionGetter func(r *http.Request) string) LocaleDetector {
	return sessionGetter
}

// ParseAcceptLanguage parses the Accept-Language header and returns the preferred locale.
func ParseAcceptLanguage(header string) string {
	if header == "" {
		return ""
	}

	// Parse and find highest quality
	var bestLocale string
	var bestQuality float64 = -1

	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		locale := part
		quality := 1.0

		// Check for quality value
		if idx := strings.Index(part, ";"); idx != -1 {
			locale = strings.TrimSpace(part[:idx])
			qPart := strings.TrimSpace(part[idx+1:])

			if strings.HasPrefix(qPart, "q=") {
				qStr := qPart[2:]
				if q, ok := parseQuality(qStr); ok {
					quality = q
				}
			}
		}

		if quality > bestQuality {
			bestQuality = quality
			bestLocale = locale
		}
	}

	return bestLocale
}

// parseQuality parses a quality value string.
func parseQuality(s string) (float64, bool) {
	if s == "" {
		return 0, false
	}

	var result float64
	var decimal float64 = 0.1
	var inDecimal bool

	for _, c := range s {
		if c == '.' {
			inDecimal = true
			continue
		}

		if c < '0' || c > '9' {
			return 0, false
		}

		digit := float64(c - '0')

		if inDecimal {
			result += digit * decimal
			decimal *= 0.1
		} else {
			result = result*10 + digit
		}
	}

	return result, true
}

// ParseAcceptLanguageAll parses Accept-Language and returns all locales sorted by quality.
func ParseAcceptLanguageAll(header string) []string {
	if header == "" {
		return nil
	}

	type localeQuality struct {
		locale  string
		quality float64
	}

	var locales []localeQuality

	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		locale := part
		quality := 1.0

		if idx := strings.Index(part, ";"); idx != -1 {
			locale = strings.TrimSpace(part[:idx])
			qPart := strings.TrimSpace(part[idx+1:])

			if strings.HasPrefix(qPart, "q=") {
				if q, ok := parseQuality(qPart[2:]); ok {
					quality = q
				}
			}
		}

		locales = append(locales, localeQuality{locale, quality})
	}

	// Sort by quality descending (simple bubble sort for small lists)
	for i := 0; i < len(locales); i++ {
		for j := i + 1; j < len(locales); j++ {
			if locales[j].quality > locales[i].quality {
				locales[i], locales[j] = locales[j], locales[i]
			}
		}
	}

	result := make([]string, len(locales))
	for i, lq := range locales {
		result[i] = lq.locale
	}

	return result
}

// BestMatch finds the best matching locale from Accept-Language.
func BestMatch(header string, available []string) string {
	if header == "" || len(available) == 0 {
		return ""
	}

	// Create set of available locales and their base forms
	availableSet := make(map[string]bool)
	for _, loc := range available {
		availableSet[strings.ToLower(loc)] = true
		if idx := strings.IndexAny(loc, "-_"); idx > 0 {
			availableSet[strings.ToLower(loc[:idx])] = true
		}
	}

	// Try each requested locale in order of preference
	for _, requested := range ParseAcceptLanguageAll(header) {
		requested = strings.ToLower(requested)

		// Exact match
		if availableSet[requested] {
			for _, loc := range available {
				if strings.ToLower(loc) == requested {
					return loc
				}
			}
		}

		// Base language match
		if idx := strings.IndexAny(requested, "-_"); idx > 0 {
			base := requested[:idx]
			if availableSet[base] {
				for _, loc := range available {
					if strings.ToLower(loc) == base || strings.HasPrefix(strings.ToLower(loc), base+"-") || strings.HasPrefix(strings.ToLower(loc), base+"_") {
						return loc
					}
				}
			}
		}
	}

	return ""
}
