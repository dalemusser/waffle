package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dalemusser/waffle/config"
)

func TestSecurityHeaders_Defaults(t *testing.T) {
	handler := SecureDefaults()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	tests := []struct {
		header string
		want   string
	}{
		{"X-Frame-Options", "SAMEORIGIN"},
		{"X-Content-Type-Options", "nosniff"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
		{"X-XSS-Protection", "1; mode=block"},
	}

	for _, tt := range tests {
		got := rec.Header().Get(tt.header)
		if got != tt.want {
			t.Errorf("%s = %q, want %q", tt.header, got, tt.want)
		}
	}

	// HSTS should NOT be set for non-TLS requests
	if hsts := rec.Header().Get("Strict-Transport-Security"); hsts != "" {
		t.Errorf("HSTS should not be set for HTTP requests, got %q", hsts)
	}
}

func TestSecurityHeaders_HSTS_OnlyForTLS(t *testing.T) {
	opts := DefaultSecurityHeadersOptions()

	handler := SecurityHeaders(opts)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// HTTP request - no HSTS
	t.Run("HTTP", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if hsts := rec.Header().Get("Strict-Transport-Security"); hsts != "" {
			t.Errorf("HSTS should not be set for HTTP, got %q", hsts)
		}
	})

	// HTTPS request - HSTS should be set
	t.Run("HTTPS", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
		req.TLS = &tls.ConnectionState{} // Simulate TLS connection
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		hsts := rec.Header().Get("Strict-Transport-Security")
		if hsts == "" {
			t.Error("HSTS should be set for HTTPS requests")
		}
		if hsts != "max-age=31536000; includeSubDomains" {
			t.Errorf("HSTS = %q, want %q", hsts, "max-age=31536000; includeSubDomains")
		}
	})
}

func TestSecurityHeaders_HSTSPreload(t *testing.T) {
	opts := DefaultSecurityHeadersOptions()
	opts.HSTSPreload = true

	handler := SecurityHeaders(opts)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	want := "max-age=31536000; includeSubDomains; preload"
	if hsts != want {
		t.Errorf("HSTS = %q, want %q", hsts, want)
	}
}

func TestSecurityHeaders_CustomOptions(t *testing.T) {
	opts := SecurityHeadersOptions{
		XFrameOptions:         "DENY",
		XContentTypeOptions:   "nosniff",
		ReferrerPolicy:        "no-referrer",
		XSSProtection:         "0",
		HSTSMaxAge:            0, // Disabled
		ContentSecurityPolicy: "default-src 'self'",
		PermissionsPolicy:     "geolocation=()",
	}

	handler := SecurityHeaders(opts)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	tests := []struct {
		header string
		want   string
	}{
		{"X-Frame-Options", "DENY"},
		{"Referrer-Policy", "no-referrer"},
		{"X-XSS-Protection", "0"},
		{"Content-Security-Policy", "default-src 'self'"},
		{"Permissions-Policy", "geolocation=()"},
	}

	for _, tt := range tests {
		got := rec.Header().Get(tt.header)
		if got != tt.want {
			t.Errorf("%s = %q, want %q", tt.header, got, tt.want)
		}
	}
}

func TestSecurityHeaders_DisabledHeaders(t *testing.T) {
	opts := SecurityHeadersOptions{
		XFrameOptions:       "",
		XContentTypeOptions: "",
		ReferrerPolicy:      "",
		XSSProtection:       "",
		HSTSMaxAge:          0,
	}

	handler := SecurityHeaders(opts)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	headers := []string{
		"X-Frame-Options",
		"X-Content-Type-Options",
		"Referrer-Policy",
		"X-XSS-Protection",
		"Strict-Transport-Security",
	}

	for _, header := range headers {
		if got := rec.Header().Get(header); got != "" {
			t.Errorf("%s should not be set when disabled, got %q", header, got)
		}
	}
}

func TestSecurityHeadersFromConfig_Enabled(t *testing.T) {
	cfg := &config.CoreConfig{}
	cfg.Security.EnableSecurityHeaders = true
	cfg.Security.XFrameOptions = "DENY"
	cfg.Security.XContentTypeOptions = "nosniff"
	cfg.Security.ReferrerPolicy = "no-referrer"
	cfg.Security.XSSProtection = "1; mode=block"
	cfg.Security.HSTSMaxAge = 63072000

	handler := SecurityHeadersFromConfig(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q, want %q", got, "DENY")
	}
}

func TestSecurityHeadersFromConfig_Disabled(t *testing.T) {
	cfg := &config.CoreConfig{}
	cfg.Security.EnableSecurityHeaders = false

	handler := SecurityHeadersFromConfig(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// No security headers should be set when disabled
	headers := []string{
		"X-Frame-Options",
		"X-Content-Type-Options",
		"Referrer-Policy",
		"X-XSS-Protection",
	}

	for _, header := range headers {
		if got := rec.Header().Get(header); got != "" {
			t.Errorf("%s should not be set when disabled, got %q", header, got)
		}
	}
}

func TestSecurityHeadersFromConfig_NilConfig(t *testing.T) {
	handler := SecurityHeadersFromConfig(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should not panic and return 200
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestDefaultSecurityHeadersOptions(t *testing.T) {
	opts := DefaultSecurityHeadersOptions()

	if opts.XFrameOptions != "SAMEORIGIN" {
		t.Errorf("XFrameOptions = %q, want %q", opts.XFrameOptions, "SAMEORIGIN")
	}
	if opts.XContentTypeOptions != "nosniff" {
		t.Errorf("XContentTypeOptions = %q, want %q", opts.XContentTypeOptions, "nosniff")
	}
	if opts.ReferrerPolicy != "strict-origin-when-cross-origin" {
		t.Errorf("ReferrerPolicy = %q, want %q", opts.ReferrerPolicy, "strict-origin-when-cross-origin")
	}
	if opts.XSSProtection != "1; mode=block" {
		t.Errorf("XSSProtection = %q, want %q", opts.XSSProtection, "1; mode=block")
	}
	if opts.HSTSMaxAge != 31536000 {
		t.Errorf("HSTSMaxAge = %d, want %d", opts.HSTSMaxAge, 31536000)
	}
	if !opts.HSTSIncludeSubDomains {
		t.Error("HSTSIncludeSubDomains should be true by default")
	}
	if opts.HSTSPreload {
		t.Error("HSTSPreload should be false by default")
	}
}

func TestSecurityHeaders_PassesThrough(t *testing.T) {
	// Ensure the middleware doesn't interfere with request/response
	handler := SecureDefaults()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Header", "test")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if got := rec.Header().Get("X-Custom-Header"); got != "test" {
		t.Errorf("X-Custom-Header = %q, want %q", got, "test")
	}
	if rec.Body.String() != "hello" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "hello")
	}
}
