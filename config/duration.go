// config/duration.go
package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// parseDurationFlexible accepts strings like "90s"/"2m", numeric seconds, or time.Duration.
// Returns def on empty/unknown types; returns def + error on invalid strings.
func parseDurationFlexible(raw interface{}, def time.Duration) (time.Duration, error) {
	switch t := raw.(type) {
	case time.Duration:
		if t <= 0 {
			return def, fmt.Errorf("duration must be >0")
		}
		return t, nil
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return def, nil
		}
		if d, err := time.ParseDuration(s); err == nil {
			if d <= 0 {
				return def, fmt.Errorf("duration must be >0")
			}
			return d, nil
		}
		// Allow plain seconds in string form, e.g. "120"
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			if n <= 0 {
				return def, fmt.Errorf("seconds must be >0")
			}
			return time.Duration(n) * time.Second, nil
		}
		return def, fmt.Errorf("cannot parse duration %q", s)
	case int:
		if t <= 0 {
			return def, fmt.Errorf("seconds must be >0")
		}
		return time.Duration(t) * time.Second, nil
	case int32:
		if t <= 0 {
			return def, fmt.Errorf("seconds must be >0")
		}
		return time.Duration(int64(t)) * time.Second, nil
	case int64:
		if t <= 0 {
			return def, fmt.Errorf("seconds must be >0")
		}
		return time.Duration(t) * time.Second, nil
	case float64:
		if t <= 0 {
			return def, fmt.Errorf("seconds must be >0")
		}
		return time.Duration(t * float64(time.Second)), nil
	default:
		// Unknown type (nil, bool, etc.) – use default, no error
		return def, nil
	}
}
