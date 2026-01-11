// feature/feature.go
package feature

// Terminology: User Identifiers
//   - UserID / userID / user_id: The MongoDB ObjectID (_id) that uniquely identifies a user record
//   - LoginID / loginID / login_id: The human-readable string users type to log in

import (
	"context"
	"hash/fnv"
	"sync"
	"time"
)

// Flag represents a feature flag configuration.
type Flag struct {
	// Key is the unique identifier for the flag.
	Key string

	// Name is a human-readable name for the flag.
	Name string

	// Description describes what the flag controls.
	Description string

	// Enabled is the default state when no rules match.
	Enabled bool

	// Rules are evaluated in order; first match wins.
	Rules []Rule

	// Percentage enables gradual rollout (0-100).
	// When set, the flag is enabled for this percentage of users.
	// Requires a user identifier for consistent hashing.
	Percentage int

	// Groups are named groups that have this flag enabled.
	// e.g., "beta-testers", "employees", "premium"
	Groups []string

	// Variants are different values the flag can return.
	// Used for A/B testing and multivariate flags.
	Variants []Variant

	// Metadata holds arbitrary key-value data.
	Metadata map[string]any

	// CreatedAt is when the flag was created.
	CreatedAt time.Time

	// UpdatedAt is when the flag was last modified.
	UpdatedAt time.Time
}

// Rule defines a condition for enabling a flag.
type Rule struct {
	// Attribute is the context attribute to check.
	// e.g., "user_id", "country", "plan", "version"
	Attribute string

	// Operator is the comparison operator.
	// Supported: "eq", "neq", "in", "nin", "gt", "gte", "lt", "lte", "contains", "regex"
	Operator string

	// Value is the value(s) to compare against.
	// For "in"/"nin", this should be a slice.
	Value any

	// Enabled is the result when this rule matches.
	Enabled bool

	// Variant is the variant to return when this rule matches.
	Variant string
}

// Variant represents a possible value for a multivariate flag.
type Variant struct {
	// Key is the unique identifier for this variant.
	Key string

	// Value is the variant's value (can be any type).
	Value any

	// Weight is the relative weight for random distribution.
	// Higher weight = more likely to be selected.
	Weight int
}

// Manager manages feature flags.
type Manager struct {
	mu       sync.RWMutex
	store    Store
	flags    map[string]*Flag
	defaults map[string]bool

	// Hooks
	onEvaluate func(flag string, ctx *EvalContext, result bool)
}

// NewManager creates a new feature flag manager.
func NewManager(store Store) *Manager {
	m := &Manager{
		store:    store,
		flags:    make(map[string]*Flag),
		defaults: make(map[string]bool),
	}

	// Load initial flags from store
	if store != nil {
		if flags, err := store.LoadAll(); err == nil {
			for _, f := range flags {
				m.flags[f.Key] = f
			}
		}
	}

	return m
}

// NewMemoryManager creates a manager with in-memory storage.
func NewMemoryManager() *Manager {
	return NewManager(NewMemoryStore())
}

// Register registers a new feature flag.
func (m *Manager) Register(flag *Flag) error {
	if flag.Key == "" {
		return ErrInvalidKey
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	if flag.CreatedAt.IsZero() {
		flag.CreatedAt = now
	}
	flag.UpdatedAt = now

	m.flags[flag.Key] = flag

	if m.store != nil {
		return m.store.Save(flag)
	}

	return nil
}

// RegisterSimple registers a simple on/off flag.
func (m *Manager) RegisterSimple(key string, enabled bool) error {
	return m.Register(&Flag{
		Key:     key,
		Enabled: enabled,
	})
}

// SetDefault sets the default value for an unregistered flag.
func (m *Manager) SetDefault(key string, enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaults[key] = enabled
}

// SetDefaults sets multiple default values.
func (m *Manager) SetDefaults(defaults map[string]bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, v := range defaults {
		m.defaults[k] = v
	}
}

// OnEvaluate sets a callback for flag evaluations.
// Useful for logging, analytics, or debugging.
func (m *Manager) OnEvaluate(fn func(flag string, ctx *EvalContext, result bool)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onEvaluate = fn
}

// Get retrieves a flag by key.
func (m *Manager) Get(key string) (*Flag, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	flag, exists := m.flags[key]
	return flag, exists
}

// All returns all registered flags.
func (m *Manager) All() []*Flag {
	m.mu.RLock()
	defer m.mu.RUnlock()

	flags := make([]*Flag, 0, len(m.flags))
	for _, f := range m.flags {
		flags = append(flags, f)
	}
	return flags
}

// IsEnabled checks if a flag is enabled (simple check, no context).
func (m *Manager) IsEnabled(key string) bool {
	return m.IsEnabledFor(key, nil)
}

// IsEnabledFor checks if a flag is enabled for the given context.
func (m *Manager) IsEnabledFor(key string, ctx *EvalContext) bool {
	m.mu.RLock()
	flag, exists := m.flags[key]
	defaultVal := m.defaults[key]
	onEval := m.onEvaluate
	m.mu.RUnlock()

	if !exists {
		return defaultVal
	}

	result := m.evaluate(flag, ctx)

	if onEval != nil {
		onEval(key, ctx, result)
	}

	return result
}

// evaluate evaluates a flag against the given context.
func (m *Manager) evaluate(flag *Flag, ctx *EvalContext) bool {
	if ctx == nil {
		ctx = &EvalContext{}
	}

	// Check rules first (most specific)
	for _, rule := range flag.Rules {
		if m.matchRule(rule, ctx) {
			return rule.Enabled
		}
	}

	// Check groups
	if len(flag.Groups) > 0 && len(ctx.Groups) > 0 {
		for _, fg := range flag.Groups {
			for _, cg := range ctx.Groups {
				if fg == cg {
					return true
				}
			}
		}
	}

	// Check percentage rollout
	if flag.Percentage > 0 && ctx.UserID != "" {
		if m.inPercentage(flag.Key, ctx.UserID, flag.Percentage) {
			return true
		}
		// If percentage is set but user not in rollout, use default
		if flag.Percentage < 100 {
			return flag.Enabled
		}
	}

	return flag.Enabled
}

// matchRule checks if a rule matches the context.
func (m *Manager) matchRule(rule Rule, ctx *EvalContext) bool {
	// Get attribute value from context
	var attrValue any
	switch rule.Attribute {
	case "user_id":
		attrValue = ctx.UserID
	case "groups":
		attrValue = ctx.Groups
	default:
		if ctx.Attributes != nil {
			attrValue = ctx.Attributes[rule.Attribute]
		}
	}

	if attrValue == nil {
		return false
	}

	return matchOperator(rule.Operator, attrValue, rule.Value)
}

// matchOperator compares values using the specified operator.
func matchOperator(op string, actual, expected any) bool {
	switch op {
	case "eq", "==", "":
		return compareEqual(actual, expected)
	case "neq", "!=":
		return !compareEqual(actual, expected)
	case "in":
		return compareIn(actual, expected)
	case "nin", "not_in":
		return !compareIn(actual, expected)
	case "contains":
		return compareContains(actual, expected)
	case "gt", ">":
		return compareNumeric(actual, expected) > 0
	case "gte", ">=":
		return compareNumeric(actual, expected) >= 0
	case "lt", "<":
		return compareNumeric(actual, expected) < 0
	case "lte", "<=":
		return compareNumeric(actual, expected) <= 0
	default:
		return false
	}
}

func compareEqual(a, b any) bool {
	// Handle string comparison
	if as, ok := a.(string); ok {
		if bs, ok := b.(string); ok {
			return as == bs
		}
	}

	// Handle numeric comparison
	af := toFloat64(a)
	bf := toFloat64(b)
	if af != 0 || bf != 0 {
		return af == bf
	}

	// Fallback to interface comparison
	return a == b
}

func compareIn(actual, list any) bool {
	// Handle slice of strings
	if items, ok := list.([]string); ok {
		if s, ok := actual.(string); ok {
			for _, item := range items {
				if item == s {
					return true
				}
			}
		}
	}

	// Handle slice of any
	if items, ok := list.([]any); ok {
		for _, item := range items {
			if compareEqual(actual, item) {
				return true
			}
		}
	}

	return false
}

func compareContains(actual, substr any) bool {
	// String contains
	if as, ok := actual.(string); ok {
		if bs, ok := substr.(string); ok {
			return len(as) > 0 && len(bs) > 0 && containsString(as, bs)
		}
	}

	// Slice contains
	if items, ok := actual.([]string); ok {
		if s, ok := substr.(string); ok {
			for _, item := range items {
				if item == s {
					return true
				}
			}
		}
	}

	return false
}

func containsString(s, substr string) bool {
	return len(substr) <= len(s) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func compareNumeric(a, b any) int {
	af := toFloat64(a)
	bf := toFloat64(b)
	if af < bf {
		return -1
	}
	if af > bf {
		return 1
	}
	return 0
}

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case float32:
		return float64(n)
	case float64:
		return n
	default:
		return 0
	}
}

// inPercentage checks if a user is in the percentage rollout.
// Uses consistent hashing so the same user always gets the same result.
func (m *Manager) inPercentage(flagKey, userID string, percentage int) bool {
	if percentage <= 0 {
		return false
	}
	if percentage >= 100 {
		return true
	}

	// Create consistent hash from flag key + user ID
	h := fnv.New32a()
	h.Write([]byte(flagKey))
	h.Write([]byte(":"))
	h.Write([]byte(userID))
	hash := h.Sum32()

	// Check if hash falls within percentage
	return int(hash%100) < percentage
}

// GetVariant returns the variant for a flag.
func (m *Manager) GetVariant(key string, ctx *EvalContext) (string, any) {
	m.mu.RLock()
	flag, exists := m.flags[key]
	m.mu.RUnlock()

	if !exists || len(flag.Variants) == 0 {
		return "", nil
	}

	if ctx == nil {
		ctx = &EvalContext{}
	}

	// Check rules for variant assignment
	for _, rule := range flag.Rules {
		if rule.Variant != "" && m.matchRule(rule, ctx) {
			for _, v := range flag.Variants {
				if v.Key == rule.Variant {
					return v.Key, v.Value
				}
			}
		}
	}

	// Use weighted random selection based on user ID
	if ctx.UserID != "" {
		return m.selectVariant(flag, ctx.UserID)
	}

	// Return first variant as default
	if len(flag.Variants) > 0 {
		return flag.Variants[0].Key, flag.Variants[0].Value
	}

	return "", nil
}

// selectVariant selects a variant based on weighted distribution.
func (m *Manager) selectVariant(flag *Flag, userID string) (string, any) {
	// Calculate total weight
	totalWeight := 0
	for _, v := range flag.Variants {
		weight := v.Weight
		if weight <= 0 {
			weight = 1
		}
		totalWeight += weight
	}

	if totalWeight == 0 {
		return "", nil
	}

	// Get consistent hash
	h := fnv.New32a()
	h.Write([]byte(flag.Key))
	h.Write([]byte(":variant:"))
	h.Write([]byte(userID))
	hash := int(h.Sum32() % uint32(totalWeight))

	// Select variant based on hash
	cumulative := 0
	for _, v := range flag.Variants {
		weight := v.Weight
		if weight <= 0 {
			weight = 1
		}
		cumulative += weight
		if hash < cumulative {
			return v.Key, v.Value
		}
	}

	return flag.Variants[0].Key, flag.Variants[0].Value
}

// Enable enables a flag.
func (m *Manager) Enable(key string) error {
	return m.SetEnabled(key, true)
}

// Disable disables a flag.
func (m *Manager) Disable(key string) error {
	return m.SetEnabled(key, false)
}

// SetEnabled sets the enabled state of a flag.
func (m *Manager) SetEnabled(key string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	flag, exists := m.flags[key]
	if !exists {
		return ErrFlagNotFound
	}

	flag.Enabled = enabled
	flag.UpdatedAt = time.Now()

	if m.store != nil {
		return m.store.Save(flag)
	}

	return nil
}

// SetPercentage sets the rollout percentage for a flag.
func (m *Manager) SetPercentage(key string, percentage int) error {
	if percentage < 0 || percentage > 100 {
		return ErrInvalidPercentage
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	flag, exists := m.flags[key]
	if !exists {
		return ErrFlagNotFound
	}

	flag.Percentage = percentage
	flag.UpdatedAt = time.Now()

	if m.store != nil {
		return m.store.Save(flag)
	}

	return nil
}

// Delete removes a flag.
func (m *Manager) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.flags[key]; !exists {
		return ErrFlagNotFound
	}

	delete(m.flags, key)

	if m.store != nil {
		return m.store.Delete(key)
	}

	return nil
}

// Reload reloads flags from the store.
func (m *Manager) Reload() error {
	if m.store == nil {
		return nil
	}

	flags, err := m.store.LoadAll()
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.flags = make(map[string]*Flag)
	for _, f := range flags {
		m.flags[f.Key] = f
	}

	return nil
}

// EvalContext provides context for flag evaluation.
type EvalContext struct {
	// UserID is the unique identifier for the user.
	// Used for percentage rollouts and variant selection.
	UserID string

	// Groups are the groups the user belongs to.
	// e.g., ["beta-testers", "premium"]
	Groups []string

	// Attributes are additional context attributes.
	// e.g., {"country": "US", "plan": "pro", "version": "2.0"}
	Attributes map[string]any
}

// NewEvalContext creates a new evaluation context.
func NewEvalContext() *EvalContext {
	return &EvalContext{
		Attributes: make(map[string]any),
	}
}

// WithUserID sets the user ID.
func (c *EvalContext) WithUserID(id string) *EvalContext {
	c.UserID = id
	return c
}

// WithGroups sets the groups.
func (c *EvalContext) WithGroups(groups ...string) *EvalContext {
	c.Groups = groups
	return c
}

// WithAttribute sets an attribute.
func (c *EvalContext) WithAttribute(key string, value any) *EvalContext {
	if c.Attributes == nil {
		c.Attributes = make(map[string]any)
	}
	c.Attributes[key] = value
	return c
}

// WithAttributes sets multiple attributes.
func (c *EvalContext) WithAttributes(attrs map[string]any) *EvalContext {
	if c.Attributes == nil {
		c.Attributes = make(map[string]any)
	}
	for k, v := range attrs {
		c.Attributes[k] = v
	}
	return c
}

// Context key for storing evaluation context.
type contextKey struct{}

// WithContext adds an EvalContext to a context.Context.
func WithContext(ctx context.Context, evalCtx *EvalContext) context.Context {
	return context.WithValue(ctx, contextKey{}, evalCtx)
}

// FromContext retrieves the EvalContext from a context.Context.
func FromContext(ctx context.Context) *EvalContext {
	if evalCtx, ok := ctx.Value(contextKey{}).(*EvalContext); ok {
		return evalCtx
	}
	return nil
}

// Global manager for convenience.
var (
	globalManager *Manager
	globalMu      sync.RWMutex
)

// SetGlobal sets the global feature manager.
func SetGlobal(m *Manager) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalManager = m
}

// Global returns the global feature manager.
func Global() *Manager {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalManager
}

// IsEnabled checks if a flag is enabled using the global manager.
func IsEnabled(key string) bool {
	m := Global()
	if m == nil {
		return false
	}
	return m.IsEnabled(key)
}

// IsEnabledFor checks if a flag is enabled for context using the global manager.
func IsEnabledFor(key string, ctx *EvalContext) bool {
	m := Global()
	if m == nil {
		return false
	}
	return m.IsEnabledFor(key, ctx)
}

// IsEnabledCtx checks if a flag is enabled using context from context.Context.
func IsEnabledCtx(ctx context.Context, key string) bool {
	m := Global()
	if m == nil {
		return false
	}
	return m.IsEnabledFor(key, FromContext(ctx))
}
