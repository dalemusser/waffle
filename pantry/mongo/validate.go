// pantry/mongo/validate.go
package mongo

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidateURI does a lightweight shape check of a Mongo connection string.
// It does not require the Mongo driver and is safe to use in config validation.
// It accepts mongodb:// and mongodb+srv:// schemes, requires a non-empty host,
// and rejects CR/LF characters.
func ValidateURI(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("empty")
	}
	if strings.ContainsAny(raw, "\r\n") {
		return fmt.Errorf("contains CR/LF")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	switch u.Scheme {
	case "mongodb", "mongodb+srv":
	default:
		return fmt.Errorf(`scheme must be "mongodb" or "mongodb+srv" (got %q)`, u.Scheme)
	}

	if u.Host == "" {
		return fmt.Errorf("missing host")
	}

	return nil
}
