// config/duration.go
package config

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// maxDurationSeconds is the maximum duration in seconds that can be safely
// converted to time.Duration without overflow. time.Duration is int64 nanoseconds,
// so max is math.MaxInt64 / 1e9 ≈ 292 years.
const maxDurationSeconds = float64(math.MaxInt64) / float64(time.Second)

// maxDurationSecondsInt64 is the integer version of maxDurationSeconds for use
// with integer type cases. This avoids float conversion in overflow checks.
const maxDurationSecondsInt64 = math.MaxInt64 / int64(time.Second)

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
			if n > maxDurationSecondsInt64 {
				return def, fmt.Errorf("duration %d seconds exceeds maximum (~292 years)", n)
			}
			return time.Duration(n) * time.Second, nil
		}
		return def, fmt.Errorf("cannot parse duration %q", s)
	case int:
		if t <= 0 {
			return def, fmt.Errorf("seconds must be >0")
		}
		if int64(t) > maxDurationSecondsInt64 {
			return def, fmt.Errorf("duration %d seconds exceeds maximum (~292 years)", t)
		}
		return time.Duration(t) * time.Second, nil
	case int32:
		if t <= 0 {
			return def, fmt.Errorf("seconds must be >0")
		}
		// int32 max is ~2.1 billion, well under maxDurationSecondsInt64 (~9.2 billion),
		// so overflow is impossible. No check needed.
		return time.Duration(int64(t)) * time.Second, nil
	case int64:
		if t <= 0 {
			return def, fmt.Errorf("seconds must be >0")
		}
		if t > maxDurationSecondsInt64 {
			return def, fmt.Errorf("duration %d seconds exceeds maximum (~292 years)", t)
		}
		return time.Duration(t) * time.Second, nil
	case float64:
		if t <= 0 {
			return def, fmt.Errorf("seconds must be >0")
		}
		// Detect overflow: time.Duration is int64 nanoseconds, max ≈ 292 years
		if t > maxDurationSeconds {
			return def, fmt.Errorf("duration %.0f seconds exceeds maximum (~292 years)", t)
		}
		// Separate integer and fractional parts to avoid float64 precision loss
		// when multiplying large values by 1e9 (nanoseconds per second).
		secs := int64(t)
		frac := t - float64(secs)
		return time.Duration(secs)*time.Second + time.Duration(frac*float64(time.Second)), nil
	default:
		// Unknown type (nil, bool, etc.) – use default, no error
		return def, nil
	}
}

// parseDurationWithDefault is a convenience wrapper that parses a duration from viper
// and logs a warning if parsing fails, returning the default value.
func parseDurationWithDefault(logger *zap.Logger, v *viper.Viper, key string, def time.Duration) time.Duration {
	dur, err := parseDurationFlexible(v.Get(key), def)
	if err != nil && logger != nil {
		logger.Warn("invalid duration config; using default",
			zap.String("key", key),
			zap.Any("value", v.Get(key)),
			zap.Duration("default", def),
			zap.Error(err))
	}
	return dur
}
