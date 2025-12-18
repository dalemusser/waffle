// pantry/validate/email.go
package validate

import "strings"

// SimpleEmailValid is a light, readable server-side guardrail.
// It is not an RFC validator; it catches empty, missing '@', or
// no dot in the domain.
//
// Limitation: This function intentionally rejects local-only addresses
// (e.g., "user@localhost", "admin@server") because they lack a dot in
// the domain portion. This is by design for production use cases where
// internet-routable email addresses are expected. For intranet or testing
// scenarios that need local domains, use a custom validator.
//
// Note: For stricter validation (e.g., ACME registration where account
// recovery requires a valid email), see config.isValidEmail which adds
// RFC 5321 length limits and additional character restrictions.
func SimpleEmailValid(s string) bool {
	s = strings.TrimSpace(s)
	at := strings.IndexByte(s, '@')
	if at <= 0 || at == len(s)-1 {
		return false
	}
	domain := s[at+1:]
	return strings.Contains(domain, ".")
}
