// toolkit/validate/email.go
package validate

import "strings"

// SimpleEmailValid is a light, readable server-side guardrail.
// It is not an RFC validator; it catches empty, missing '@', or
// no dot in the domain.
func SimpleEmailValid(s string) bool {
	s = strings.TrimSpace(s)
	at := strings.IndexByte(s, '@')
	if at <= 0 || at == len(s)-1 {
		return false
	}
	domain := s[at+1:]
	return strings.Contains(domain, ".")
}
