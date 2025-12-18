// i18n/i18n.go
package i18n

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
)

// Bundle holds translations for multiple locales.
type Bundle struct {
	mu              sync.RWMutex
	locales         map[string]*Locale
	defaultLocale   string
	fallbackLocale  string
	missingKeyFunc  MissingKeyFunc
	templateFuncs   template.FuncMap
}

// Locale holds translations for a single locale.
type Locale struct {
	Tag          string
	Messages     map[string]string
	PluralRules  PluralFunc
}

// MissingKeyFunc is called when a translation key is not found.
type MissingKeyFunc func(locale, key string) string

// NewBundle creates a new translation bundle.
func NewBundle(defaultLocale string) *Bundle {
	return &Bundle{
		locales:        make(map[string]*Locale),
		defaultLocale:  defaultLocale,
		fallbackLocale: defaultLocale,
		missingKeyFunc: defaultMissingKeyFunc,
		templateFuncs:  make(template.FuncMap),
	}
}

// defaultMissingKeyFunc returns the key itself when translation is missing.
func defaultMissingKeyFunc(locale, key string) string {
	return key
}

// SetFallbackLocale sets the fallback locale for missing translations.
func (b *Bundle) SetFallbackLocale(locale string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.fallbackLocale = locale
}

// SetMissingKeyFunc sets the function called when a key is missing.
func (b *Bundle) SetMissingKeyFunc(fn MissingKeyFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.missingKeyFunc = fn
}

// AddLocale adds or updates a locale with the given messages.
func (b *Bundle) AddLocale(tag string, messages map[string]string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	locale, exists := b.locales[tag]
	if !exists {
		locale = &Locale{
			Tag:         tag,
			Messages:    make(map[string]string),
			PluralRules: GetPluralFunc(tag),
		}
		b.locales[tag] = locale
	}

	for k, v := range messages {
		locale.Messages[k] = v
	}
}

// AddMessages is an alias for AddLocale.
func (b *Bundle) AddMessages(locale string, messages map[string]string) {
	b.AddLocale(locale, messages)
}

// LoadJSON loads translations from a JSON file.
// The file should contain a flat key-value object: {"key": "translation"}
func (b *Bundle) LoadJSON(locale, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("i18n: failed to read file %s: %w", path, err)
	}

	var messages map[string]string
	if err := json.Unmarshal(data, &messages); err != nil {
		return fmt.Errorf("i18n: failed to parse JSON %s: %w", path, err)
	}

	b.AddLocale(locale, messages)
	return nil
}

// LoadJSONDir loads all JSON files from a directory.
// Files should be named {locale}.json (e.g., en.json, fr.json).
func (b *Bundle) LoadJSONDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("i18n: failed to read directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}

		locale := strings.TrimSuffix(name, ".json")
		path := filepath.Join(dir, name)

		if err := b.LoadJSON(locale, path); err != nil {
			return err
		}
	}

	return nil
}

// LoadNestedJSON loads translations from a nested JSON file.
// Nested keys are flattened with dots: {"errors": {"required": "..."}} -> "errors.required"
func (b *Bundle) LoadNestedJSON(locale, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("i18n: failed to read file %s: %w", path, err)
	}

	var nested map[string]any
	if err := json.Unmarshal(data, &nested); err != nil {
		return fmt.Errorf("i18n: failed to parse JSON %s: %w", path, err)
	}

	messages := flattenMap(nested, "")
	b.AddLocale(locale, messages)
	return nil
}

// flattenMap flattens a nested map into dot-separated keys.
func flattenMap(m map[string]any, prefix string) map[string]string {
	result := make(map[string]string)

	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}

		switch val := v.(type) {
		case string:
			result[key] = val
		case map[string]any:
			for fk, fv := range flattenMap(val, key) {
				result[fk] = fv
			}
		}
	}

	return result
}

// Locales returns all registered locale tags.
func (b *Bundle) Locales() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	tags := make([]string, 0, len(b.locales))
	for tag := range b.locales {
		tags = append(tags, tag)
	}
	return tags
}

// HasLocale returns true if the locale exists.
func (b *Bundle) HasLocale(locale string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, exists := b.locales[locale]
	return exists
}

// DefaultLocale returns the default locale.
func (b *Bundle) DefaultLocale() string {
	return b.defaultLocale
}

// Localizer returns a Localizer for the given locale.
func (b *Bundle) Localizer(locale string) *Localizer {
	return &Localizer{
		bundle: b,
		locale: locale,
	}
}

// T translates a key using the default locale.
func (b *Bundle) T(key string, args ...any) string {
	return b.Localizer(b.defaultLocale).T(key, args...)
}

// TWithLocale translates a key for a specific locale.
func (b *Bundle) TWithLocale(locale, key string, args ...any) string {
	return b.Localizer(locale).T(key, args...)
}

// getMessage retrieves a message, falling back if necessary.
func (b *Bundle) getMessage(locale, key string) (string, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Try exact locale
	if loc, exists := b.locales[locale]; exists {
		if msg, ok := loc.Messages[key]; ok {
			return msg, true
		}
	}

	// Try base locale (e.g., "en" for "en-US")
	if idx := strings.IndexAny(locale, "-_"); idx > 0 {
		baseLocale := locale[:idx]
		if loc, exists := b.locales[baseLocale]; exists {
			if msg, ok := loc.Messages[key]; ok {
				return msg, true
			}
		}
	}

	// Try fallback locale
	if locale != b.fallbackLocale {
		if loc, exists := b.locales[b.fallbackLocale]; exists {
			if msg, ok := loc.Messages[key]; ok {
				return msg, true
			}
		}
	}

	return "", false
}

// Localizer translates messages for a specific locale.
type Localizer struct {
	bundle *Bundle
	locale string
}

// Locale returns the localizer's locale.
func (l *Localizer) Locale() string {
	return l.locale
}

// T translates a message key.
// If args are provided, they are used for template substitution.
//
// Simple substitution with positional args:
//
//	"Hello, %s!" with args "World" -> "Hello, World!"
//
// Named substitution with map:
//
//	"Hello, {{.Name}}!" with args map[string]any{"Name": "World"} -> "Hello, World!"
func (l *Localizer) T(key string, args ...any) string {
	msg, found := l.bundle.getMessage(l.locale, key)
	if !found {
		return l.bundle.missingKeyFunc(l.locale, key)
	}

	if len(args) == 0 {
		return msg
	}

	// Check if message uses Go template syntax
	if strings.Contains(msg, "{{") {
		return l.executeTemplate(msg, args)
	}

	// Use fmt.Sprintf for simple substitution
	return fmt.Sprintf(msg, args...)
}

// executeTemplate executes a Go template with the given data.
func (l *Localizer) executeTemplate(msg string, args []any) string {
	// Get data for template
	var data any
	if len(args) == 1 {
		data = args[0]
	} else {
		data = args
	}

	tmpl, err := template.New("").Funcs(l.bundle.templateFuncs).Parse(msg)
	if err != nil {
		return msg
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return msg
	}

	return buf.String()
}

// TPlural translates a message with pluralization.
// count determines which plural form to use.
// args are used for template substitution (count is automatically included as .Count).
func (l *Localizer) TPlural(key string, count int, args ...any) string {
	// Get plural form
	l.bundle.mu.RLock()
	loc, exists := l.bundle.locales[l.locale]
	l.bundle.mu.RUnlock()

	var pluralFunc PluralFunc = PluralEnglish
	if exists && loc.PluralRules != nil {
		pluralFunc = loc.PluralRules
	}

	form := pluralFunc(count)
	pluralKey := key + "." + string(form)

	// Try plural-specific key first
	msg, found := l.bundle.getMessage(l.locale, pluralKey)
	if !found {
		// Fall back to base key
		msg, found = l.bundle.getMessage(l.locale, key)
		if !found {
			return l.bundle.missingKeyFunc(l.locale, key)
		}
	}

	// Build data map with count
	data := make(map[string]any)
	data["Count"] = count

	// Merge additional args if provided
	if len(args) == 1 {
		if m, ok := args[0].(map[string]any); ok {
			for k, v := range m {
				data[k] = v
			}
		}
	}

	// Execute template
	if strings.Contains(msg, "{{") {
		return l.executeTemplate(msg, []any{data})
	}

	return fmt.Sprintf(msg, count)
}

// N is an alias for TPlural.
func (l *Localizer) N(key string, count int, args ...any) string {
	return l.TPlural(key, count, args...)
}

// Has returns true if the key exists for this locale.
func (l *Localizer) Has(key string) bool {
	_, found := l.bundle.getMessage(l.locale, key)
	return found
}

// Context key for storing localizer.
type contextKey struct{}

// WithLocalizer adds a Localizer to the context.
func WithLocalizer(ctx context.Context, l *Localizer) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}

// FromContext retrieves the Localizer from context.
// Returns nil if not found.
func FromContext(ctx context.Context) *Localizer {
	l, _ := ctx.Value(contextKey{}).(*Localizer)
	return l
}

// T translates using the Localizer from context.
// Falls back to key if no localizer in context.
func T(ctx context.Context, key string, args ...any) string {
	l := FromContext(ctx)
	if l == nil {
		return key
	}
	return l.T(key, args...)
}

// TPlural translates with pluralization using context.
func TPlural(ctx context.Context, key string, count int, args ...any) string {
	l := FromContext(ctx)
	if l == nil {
		return key
	}
	return l.TPlural(key, count, args...)
}

// Global bundle for convenience.
var globalBundle *Bundle
var globalMu sync.RWMutex

// SetGlobalBundle sets the global bundle.
func SetGlobalBundle(b *Bundle) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalBundle = b
}

// GetGlobalBundle returns the global bundle.
func GetGlobalBundle() *Bundle {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalBundle
}

// Global translates using the global bundle and default locale.
func Global(key string, args ...any) string {
	b := GetGlobalBundle()
	if b == nil {
		return key
	}
	return b.T(key, args...)
}

// GlobalWithLocale translates using the global bundle for a specific locale.
func GlobalWithLocale(locale, key string, args ...any) string {
	b := GetGlobalBundle()
	if b == nil {
		return key
	}
	return b.TWithLocale(locale, key, args...)
}
