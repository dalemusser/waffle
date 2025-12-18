// testing/helpers.go
package testing

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
)

// TestLogger returns a no-op logger for tests.
func TestLogger() *zap.Logger {
	return zap.NewNop()
}

// DevLogger returns a development logger for debugging tests.
func DevLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

// Context returns a context with a reasonable timeout for tests.
func Context(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// ContextWithTimeout returns a context with a custom timeout.
func ContextWithTimeout(t *testing.T, timeout time.Duration) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)
	return ctx
}

// TempDir creates a temporary directory that is cleaned up after the test.
func TempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "waffle-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	return dir
}

// TempFile creates a temporary file with the given content.
func TempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := TempDir(t)
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	return path
}

// ReadFixture reads a fixture file from testdata directory.
func ReadFixture(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", path))
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", path, err)
	}
	return data
}

// ReadJSONFixture reads and unmarshals a JSON fixture file.
func ReadJSONFixture(t *testing.T, path string, v any) {
	t.Helper()
	data := ReadFixture(t, path)
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("failed to unmarshal fixture %s: %v", path, err)
	}
}

// MustJSON marshals v to JSON, failing the test on error.
func MustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	return data
}

// MustJSONString marshals v to a JSON string, failing the test on error.
func MustJSONString(t *testing.T, v any) string {
	return string(MustJSON(t, v))
}

// SetEnv sets an environment variable for the duration of the test.
func SetEnv(t *testing.T, key, value string) {
	t.Helper()
	old, exists := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("failed to set env %s: %v", key, err)
	}
	t.Cleanup(func() {
		if exists {
			os.Setenv(key, old)
		} else {
			os.Unsetenv(key)
		}
	})
}

// UnsetEnv unsets an environment variable for the duration of the test.
func UnsetEnv(t *testing.T, key string) {
	t.Helper()
	old, exists := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("failed to unset env %s: %v", key, err)
	}
	t.Cleanup(func() {
		if exists {
			os.Setenv(key, old)
		}
	})
}

// Eventually retries a check function until it passes or times out.
func Eventually(t *testing.T, check func() bool, timeout, interval time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(interval)
	}
	t.Fatal("condition not met within timeout")
}

// Collect is an interface for collecting test failures.
type Collect interface {
	Error(args ...any)
	Errorf(format string, args ...any)
	Fail()
	FailNow()
	Failed() bool
}

// collector is a minimal implementation for collecting failures.
type collector struct {
	failed bool
}

func (c *collector) Error(args ...any)                 { c.failed = true }
func (c *collector) Errorf(format string, args ...any) { c.failed = true }
func (c *collector) Fail()                             { c.failed = true }
func (c *collector) FailNow()                          { c.failed = true }
func (c *collector) Failed() bool                      { return c.failed }

// EventuallyWithCollect retries a check function that uses a Collect interface until it passes.
func EventuallyWithCollect(t *testing.T, check func(c Collect), timeout, interval time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		c := &collector{}
		check(c)
		if !c.Failed() {
			return
		}
		time.Sleep(interval)
	}
	t.Fatal("condition not met within timeout")
}

// Parallel marks the test as parallel.
// Convenience wrapper for t.Parallel().
func Parallel(t *testing.T) {
	t.Helper()
	t.Parallel()
}

// Skip skips the test if the condition is true.
func Skip(t *testing.T, condition bool, reason string) {
	t.Helper()
	if condition {
		t.Skip(reason)
	}
}

// SkipShort skips the test if running with -short flag.
func SkipShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
}

// SkipCI skips the test if running in CI environment.
func SkipCI(t *testing.T) {
	t.Helper()
	if os.Getenv("CI") != "" {
		t.Skip("skipping in CI")
	}
}

// RequireEnv skips the test if an environment variable is not set.
func RequireEnv(t *testing.T, key string) string {
	t.Helper()
	value := os.Getenv(key)
	if value == "" {
		t.Skipf("skipping: %s not set", key)
	}
	return value
}
