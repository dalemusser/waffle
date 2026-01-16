package urlutil

import "testing"

func TestSafeReturn(t *testing.T) {
	tests := []struct {
		name     string
		ret      string
		badID    string
		fallback string
		want     string
	}{
		// Basic functionality
		{"empty returns fallback", "", "", "/default", "/default"},
		{"valid path", "/dashboard", "", "/default", "/dashboard"},
		{"valid path with segments", "/admin/users", "", "/default", "/admin/users"},

		// Security: header injection
		{"rejects CR", "/foo\rbar", "", "/default", "/default"},
		{"rejects LF", "/foo\nbar", "", "/default", "/default"},
		{"rejects backslash", "/foo\\bar", "", "/default", "/default"},

		// Security: open redirect
		{"rejects scheme", "http://evil.com", "", "/default", "/default"},
		{"rejects scheme-relative", "//evil.com/path", "", "/default", "/default"},
		{"rejects non-absolute", "relative/path", "", "/default", "/default"},

		// badID filtering
		{"rejects path with badID", "/users/123/edit", "123", "/default", "/default"},
		{"allows path without badID", "/users/456/edit", "123", "/default", "/users/456/edit"},

		// Default exclusions (new behavior)
		{"rejects /logout", "/logout", "", "/default", "/default"},
		{"rejects /login", "/login", "", "/default", "/default"},
		{"rejects /logout with query", "/logout?foo=bar", "", "/default", "/default"},
		{"rejects /login with query", "/login?return=/dashboard", "", "/default", "/default"},
		{"allows /logout-help (not exact match)", "/logout-help", "", "/default", "/logout-help"},
		{"allows /login-page (not exact match)", "/login-page", "", "/default", "/login-page"},
		{"allows /user/logout (not at root)", "/user/logout", "", "/default", "/user/logout"},

		// Path normalization
		{"normalizes path", "/foo/../bar", "", "/default", "/bar"},
		{"handles trailing slash", "/dashboard/", "", "/default", "/dashboard"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SafeReturn(tt.ret, tt.badID, tt.fallback)
			if got != tt.want {
				t.Errorf("SafeReturn(%q, %q, %q) = %q, want %q",
					tt.ret, tt.badID, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestSafeReturnExcluding(t *testing.T) {
	tests := []struct {
		name     string
		ret      string
		badID    string
		fallback string
		excluded []string
		want     string
	}{
		// Custom exclusions
		{"custom exclusion exact", "/admin", "", "/default", []string{"/admin"}, "/default"},
		{"custom exclusion with query", "/admin?tab=users", "", "/default", []string{"/admin"}, "/default"},
		{"custom exclusion not matched", "/admin-panel", "", "/default", []string{"/admin"}, "/admin-panel"},
		{"multiple exclusions", "/logout", "", "/default", []string{"/login", "/logout", "/register"}, "/default"},

		// No exclusions
		{"nil exclusions allows /logout", "/logout", "", "/default", nil, "/logout"},
		{"empty exclusions allows /logout", "/logout", "", "/default", []string{}, "/logout"},

		// Still validates other rules
		{"still rejects CR even with no exclusions", "/foo\rbar", "", "/default", nil, "/default"},
		{"still checks badID", "/users/123", "123", "/default", nil, "/default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SafeReturnExcluding(tt.ret, tt.badID, tt.fallback, tt.excluded)
			if got != tt.want {
				t.Errorf("SafeReturnExcluding(%q, %q, %q, %v) = %q, want %q",
					tt.ret, tt.badID, tt.fallback, tt.excluded, got, tt.want)
			}
		})
	}
}

func TestSafeReturnRaw(t *testing.T) {
	tests := []struct {
		name     string
		ret      string
		badID    string
		fallback string
		want     string
	}{
		// Should allow default excluded paths
		{"allows /logout", "/logout", "", "/default", "/logout"},
		{"allows /login", "/login", "", "/default", "/login"},
		{"allows /logout with query", "/logout?foo=bar", "", "/default", "/logout?foo=bar"},

		// Should still enforce other security rules
		{"still rejects CR", "/foo\rbar", "", "/default", "/default"},
		{"still rejects external URLs", "http://evil.com", "", "/default", "/default"},
		{"still checks badID", "/users/123", "123", "/default", "/default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SafeReturnRaw(tt.ret, tt.badID, tt.fallback)
			if got != tt.want {
				t.Errorf("SafeReturnRaw(%q, %q, %q) = %q, want %q",
					tt.ret, tt.badID, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestDefaultExcludedPaths(t *testing.T) {
	// Verify the default exclusions are set correctly
	expected := []string{"/logout", "/login"}
	if len(DefaultExcludedPaths) != len(expected) {
		t.Errorf("DefaultExcludedPaths has %d items, want %d",
			len(DefaultExcludedPaths), len(expected))
	}
	for i, path := range expected {
		if i >= len(DefaultExcludedPaths) || DefaultExcludedPaths[i] != path {
			t.Errorf("DefaultExcludedPaths[%d] = %q, want %q",
				i, DefaultExcludedPaths[i], path)
		}
	}
}
