// Package validate provides struct validation using struct tags with support
// for custom validators and internationalized error messages.
//
// Basic usage:
//
//	type User struct {
//	    Name  string `validate:"required,min=2,max=50"`
//	    Email string `validate:"required,email"`
//	    Age   int    `validate:"min=18,max=120"`
//	}
//
//	v := validate.New()
//	if err := v.Struct(user); err != nil {
//	    for _, e := range err.(validate.Errors) {
//	        fmt.Printf("%s: %s\n", e.Field, e.Message)
//	    }
//	}
package validate

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Validator validates struct fields using tags.
type Validator struct {
	tagName    string
	rules      map[string]RuleFunc
	messages   *MessageProvider
	mu         sync.RWMutex
	stopOnFirst bool
}

// RuleFunc is a validation rule function.
// It receives the field value, the parameter (if any), and the full struct.
// Returns an error message key if validation fails, empty string if valid.
type RuleFunc func(value any, param string, structValue reflect.Value) string

// Option configures the validator.
type Option func(*Validator)

// New creates a new validator with default rules.
func New(opts ...Option) *Validator {
	v := &Validator{
		tagName:  "validate",
		rules:    make(map[string]RuleFunc),
		messages: DefaultMessages(),
	}

	// Register built-in rules
	v.registerBuiltinRules()

	// Apply options
	for _, opt := range opts {
		opt(v)
	}

	return v
}

// WithTagName sets a custom tag name (default: "validate").
func WithTagName(name string) Option {
	return func(v *Validator) {
		v.tagName = name
	}
}

// WithMessages sets a custom message provider.
func WithMessages(m *MessageProvider) Option {
	return func(v *Validator) {
		v.messages = m
	}
}

// WithStopOnFirstError stops validation after the first error.
func WithStopOnFirstError() Option {
	return func(v *Validator) {
		v.stopOnFirst = true
	}
}

// RegisterRule registers a custom validation rule.
func (v *Validator) RegisterRule(name string, fn RuleFunc) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.rules[name] = fn
}

// RegisterRuleFunc registers a simple validation function.
func (v *Validator) RegisterRuleFunc(name string, fn func(value any) bool, messageKey string) {
	v.RegisterRule(name, func(value any, param string, sv reflect.Value) string {
		if fn(value) {
			return ""
		}
		return messageKey
	})
}

// SetMessages sets the message provider.
func (v *Validator) SetMessages(m *MessageProvider) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.messages = m
}

// Struct validates a struct using its validate tags.
func (v *Validator) Struct(s any) error {
	return v.StructCtx(nil, s)
}

// StructCtx validates a struct with additional context data.
func (v *Validator) StructCtx(ctx map[string]any, s any) error {
	val := reflect.ValueOf(s)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("validate: expected struct, got %s", val.Kind())
	}

	return v.validateStruct(val, "")
}

// Var validates a single variable.
func (v *Validator) Var(value any, tag string) error {
	errs := v.validateValue(reflect.ValueOf(value), "", tag, reflect.Value{})
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// validateStruct validates all fields of a struct.
func (v *Validator) validateStruct(val reflect.Value, prefix string) Errors {
	var errs Errors
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get field name
		fieldName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				fieldName = parts[0]
			}
		}

		if prefix != "" {
			fieldName = prefix + "." + fieldName
		}

		// Get validation tag
		tag := field.Tag.Get(v.tagName)

		// Validate field
		fieldErrs := v.validateValue(fieldVal, fieldName, tag, val)
		errs = append(errs, fieldErrs...)

		if v.stopOnFirst && len(errs) > 0 {
			return errs
		}

		// Handle nested structs
		if fieldVal.Kind() == reflect.Struct && fieldVal.Type() != reflect.TypeOf(time.Time{}) {
			nestedErrs := v.validateStruct(fieldVal, fieldName)
			errs = append(errs, nestedErrs...)
		}

		// Handle pointers to structs
		if fieldVal.Kind() == reflect.Ptr && !fieldVal.IsNil() {
			elem := fieldVal.Elem()
			if elem.Kind() == reflect.Struct && elem.Type() != reflect.TypeOf(time.Time{}) {
				nestedErrs := v.validateStruct(elem, fieldName)
				errs = append(errs, nestedErrs...)
			}
		}

		// Handle slices of structs
		if fieldVal.Kind() == reflect.Slice {
			for j := 0; j < fieldVal.Len(); j++ {
				elem := fieldVal.Index(j)
				if elem.Kind() == reflect.Struct && elem.Type() != reflect.TypeOf(time.Time{}) {
					nestedErrs := v.validateStruct(elem, fmt.Sprintf("%s[%d]", fieldName, j))
					errs = append(errs, nestedErrs...)
				}
				if elem.Kind() == reflect.Ptr && !elem.IsNil() {
					ptrElem := elem.Elem()
					if ptrElem.Kind() == reflect.Struct && ptrElem.Type() != reflect.TypeOf(time.Time{}) {
						nestedErrs := v.validateStruct(ptrElem, fmt.Sprintf("%s[%d]", fieldName, j))
						errs = append(errs, nestedErrs...)
					}
				}
			}
		}
	}

	return errs
}

// validateValue validates a single value against rules.
func (v *Validator) validateValue(val reflect.Value, fieldName, tag string, structVal reflect.Value) Errors {
	if tag == "" || tag == "-" {
		return nil
	}

	var errs Errors
	rules := parseTag(tag)

	// Check if field is optional (omitempty)
	isOptional := false
	for _, r := range rules {
		if r.name == "omitempty" {
			isOptional = true
			break
		}
	}

	// If optional and empty, skip other validations
	if isOptional && isEmpty(val) {
		return nil
	}

	v.mu.RLock()
	defer v.mu.RUnlock()

	for _, rule := range rules {
		if rule.name == "omitempty" {
			continue
		}

		ruleFn, ok := v.rules[rule.name]
		if !ok {
			continue
		}

		var value any
		if val.IsValid() && val.CanInterface() {
			value = val.Interface()
		}

		msgKey := ruleFn(value, rule.param, structVal)
		if msgKey != "" {
			msg := v.messages.Get(msgKey, fieldName, rule.param)
			errs = append(errs, &Error{
				Field:   fieldName,
				Rule:    rule.name,
				Param:   rule.param,
				Value:   value,
				Message: msg,
			})

			if v.stopOnFirst {
				return errs
			}
		}
	}

	return errs
}

// rule represents a parsed validation rule.
type rule struct {
	name  string
	param string
}

// parseTag parses a validation tag into rules.
func parseTag(tag string) []rule {
	var rules []rule
	parts := strings.Split(tag, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		r := rule{}
		if idx := strings.Index(part, "="); idx != -1 {
			r.name = part[:idx]
			r.param = part[idx+1:]
		} else {
			r.name = part
		}
		rules = append(rules, r)
	}

	return rules
}

// isEmpty checks if a value is empty.
func isEmpty(val reflect.Value) bool {
	if !val.IsValid() {
		return true
	}

	switch val.Kind() {
	case reflect.String:
		return val.String() == ""
	case reflect.Bool:
		return !val.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return val.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return val.Float() == 0
	case reflect.Slice, reflect.Map, reflect.Array:
		return val.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return val.IsNil()
	}

	return false
}

// registerBuiltinRules registers all built-in validation rules.
func (v *Validator) registerBuiltinRules() {
	// Required
	v.rules["required"] = ruleRequired

	// Type validations
	v.rules["email"] = ruleEmail
	v.rules["url"] = ruleURL
	v.rules["uri"] = ruleURI
	v.rules["uuid"] = ruleUUID
	v.rules["uuid3"] = ruleUUID3
	v.rules["uuid4"] = ruleUUID4
	v.rules["uuid5"] = ruleUUID5
	v.rules["ulid"] = ruleULID
	v.rules["alpha"] = ruleAlpha
	v.rules["alphanum"] = ruleAlphaNum
	v.rules["alphanumspace"] = ruleAlphaNumSpace
	v.rules["numeric"] = ruleNumeric
	v.rules["hexadecimal"] = ruleHexadecimal
	v.rules["hexcolor"] = ruleHexColor
	v.rules["rgb"] = ruleRGB
	v.rules["rgba"] = ruleRGBA
	v.rules["hsl"] = ruleHSL
	v.rules["hsla"] = ruleHSLA
	v.rules["json"] = ruleJSON
	v.rules["jwt"] = ruleJWT
	v.rules["base64"] = ruleBase64
	v.rules["base64url"] = ruleBase64URL
	v.rules["isbn"] = ruleISBN
	v.rules["isbn10"] = ruleISBN10
	v.rules["isbn13"] = ruleISBN13
	v.rules["issn"] = ruleISSN
	v.rules["ascii"] = ruleASCII
	v.rules["printascii"] = rulePrintASCII
	v.rules["multibyte"] = ruleMultibyte
	v.rules["lowercase"] = ruleLowercase
	v.rules["uppercase"] = ruleUppercase

	// Length/size validations
	v.rules["min"] = ruleMin
	v.rules["max"] = ruleMax
	v.rules["len"] = ruleLen
	v.rules["between"] = ruleBetween

	// Comparison validations
	v.rules["eq"] = ruleEq
	v.rules["ne"] = ruleNe
	v.rules["gt"] = ruleGt
	v.rules["gte"] = ruleGte
	v.rules["lt"] = ruleLt
	v.rules["lte"] = ruleLte

	// Field comparison validations
	v.rules["eqfield"] = ruleEqField
	v.rules["nefield"] = ruleNeField
	v.rules["gtfield"] = ruleGtField
	v.rules["gtefield"] = ruleGteField
	v.rules["ltfield"] = ruleLtField
	v.rules["ltefield"] = ruleLteField

	// Network validations
	v.rules["ip"] = ruleIP
	v.rules["ipv4"] = ruleIPv4
	v.rules["ipv6"] = ruleIPv6
	v.rules["cidr"] = ruleCIDR
	v.rules["cidrv4"] = ruleCIDRv4
	v.rules["cidrv6"] = ruleCIDRv6
	v.rules["mac"] = ruleMAC
	v.rules["hostname"] = ruleHostname
	v.rules["fqdn"] = ruleFQDN

	// String validations
	v.rules["contains"] = ruleContains
	v.rules["containsany"] = ruleContainsAny
	v.rules["containsrune"] = ruleContainsRune
	v.rules["excludes"] = ruleExcludes
	v.rules["excludesall"] = ruleExcludesAll
	v.rules["excludesrune"] = ruleExcludesRune
	v.rules["startswith"] = ruleStartsWith
	v.rules["endswith"] = ruleEndsWith
	v.rules["startsnotwith"] = ruleStartsNotWith
	v.rules["endsnotwith"] = ruleEndsNotWith

	// Format validations
	v.rules["regex"] = ruleRegex
	v.rules["datetime"] = ruleDatetime
	v.rules["date"] = ruleDate
	v.rules["time"] = ruleTime
	v.rules["timezone"] = ruleTimezone
	v.rules["duration"] = ruleDuration

	// Special validations
	v.rules["oneof"] = ruleOneOf
	v.rules["enum"] = ruleOneOf // alias
	v.rules["unique"] = ruleUnique
	v.rules["dive"] = ruleDive

	// Credit card
	v.rules["creditcard"] = ruleCreditCard

	// Phone
	v.rules["e164"] = ruleE164

	// Country/language codes
	v.rules["countrycode"] = ruleCountryCode
	v.rules["languagecode"] = ruleLanguageCode
	v.rules["bcp47"] = ruleBCP47

	// File validations
	v.rules["filepath"] = ruleFilePath
	v.rules["dirpath"] = ruleDirPath

	// Semantic versioning
	v.rules["semver"] = ruleSemver

	// Boolean string
	v.rules["boolean"] = ruleBoolean

	// CVV
	v.rules["cvv"] = ruleCVV

	// Latitude/Longitude
	v.rules["latitude"] = ruleLatitude
	v.rules["longitude"] = ruleLongitude

	// PostalCode
	v.rules["postcode"] = rulePostalCode

	// Slug
	v.rules["slug"] = ruleSlug

	// Strong password
	v.rules["strongpassword"] = ruleStrongPassword
}

// Error represents a validation error.
type Error struct {
	Field   string
	Rule    string
	Param   string
	Value   any
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

// Errors is a collection of validation errors.
type Errors []*Error

func (e Errors) Error() string {
	if len(e) == 0 {
		return ""
	}

	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Message)
	}
	return strings.Join(msgs, "; ")
}

// HasErrors returns true if there are any errors.
func (e Errors) HasErrors() bool {
	return len(e) > 0
}

// FieldErrors returns all errors for a specific field.
func (e Errors) FieldErrors(field string) Errors {
	var result Errors
	for _, err := range e {
		if err.Field == field {
			result = append(result, err)
		}
	}
	return result
}

// ToMap converts errors to a map of field -> messages.
func (e Errors) ToMap() map[string][]string {
	result := make(map[string][]string)
	for _, err := range e {
		result[err.Field] = append(result[err.Field], err.Message)
	}
	return result
}

// First returns the first error or nil.
func (e Errors) First() *Error {
	if len(e) > 0 {
		return e[0]
	}
	return nil
}

// Default validator instance
var defaultValidator = New()

// Struct validates a struct using the default validator.
func Struct(s any) error {
	return defaultValidator.Struct(s)
}

// Var validates a variable using the default validator.
func Var(value any, tag string) error {
	return defaultValidator.Var(value, tag)
}

// RegisterRule registers a rule on the default validator.
func RegisterRule(name string, fn RuleFunc) {
	defaultValidator.RegisterRule(name, fn)
}

// Helper functions for rules

// toString converts a value to string.
func toString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// toInt converts a value to int64.
func toInt(v any) (int64, bool) {
	if v == nil {
		return 0, false
	}
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(val.Uint()), true
	case reflect.Float32, reflect.Float64:
		return int64(val.Float()), true
	case reflect.String:
		if i, err := strconv.ParseInt(val.String(), 10, 64); err == nil {
			return i, true
		}
	}
	return 0, false
}

// toFloat converts a value to float64.
func toFloat(v any) (float64, bool) {
	if v == nil {
		return 0, false
	}
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Float32, reflect.Float64:
		return val.Float(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(val.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(val.Uint()), true
	case reflect.String:
		if f, err := strconv.ParseFloat(val.String(), 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// getLen returns the length of a value.
func getLen(v any) int {
	if v == nil {
		return 0
	}
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.String:
		return utf8.RuneCountInString(val.String())
	case reflect.Slice, reflect.Map, reflect.Array:
		return val.Len()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(val.Uint())
	case reflect.Float32, reflect.Float64:
		return int(val.Float())
	}
	return 0
}

// getFieldValue gets a field value from a struct by name.
func getFieldValue(structVal reflect.Value, fieldName string) (any, bool) {
	if !structVal.IsValid() || structVal.Kind() != reflect.Struct {
		return nil, false
	}

	field := structVal.FieldByName(fieldName)
	if !field.IsValid() {
		return nil, false
	}

	return field.Interface(), true
}

// Compiled regex patterns
var (
	emailRegex        = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	urlRegex          = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	uuidRegex         = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	uuid3Regex        = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-3[0-9a-fA-F]{3}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	uuid4Regex        = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	uuid5Regex        = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-5[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	ulidRegex         = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)
	alphaRegex        = regexp.MustCompile(`^[a-zA-Z]+$`)
	alphaNumRegex     = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	alphaNumSpaceRegex = regexp.MustCompile(`^[a-zA-Z0-9 ]+$`)
	numericRegex      = regexp.MustCompile(`^[-+]?[0-9]+$`)
	hexRegex          = regexp.MustCompile(`^[0-9a-fA-F]+$`)
	hexColorRegex     = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)
	rgbRegex          = regexp.MustCompile(`^rgb\(\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,\s*(\d{1,3})\s*\)$`)
	rgbaRegex         = regexp.MustCompile(`^rgba\(\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,\s*(0|1|0?\.\d+)\s*\)$`)
	hslRegex          = regexp.MustCompile(`^hsl\(\s*(\d{1,3})\s*,\s*(\d{1,3})%\s*,\s*(\d{1,3})%\s*\)$`)
	hslaRegex         = regexp.MustCompile(`^hsla\(\s*(\d{1,3})\s*,\s*(\d{1,3})%\s*,\s*(\d{1,3})%\s*,\s*(0|1|0?\.\d+)\s*\)$`)
	base64Regex       = regexp.MustCompile(`^[A-Za-z0-9+/]*={0,2}$`)
	base64URLRegex    = regexp.MustCompile(`^[A-Za-z0-9_-]*={0,2}$`)
	isbn10Regex       = regexp.MustCompile(`^(?:\d{9}X|\d{10})$`)
	isbn13Regex       = regexp.MustCompile(`^\d{13}$`)
	issnRegex         = regexp.MustCompile(`^\d{4}-\d{3}[\dX]$`)
	ipv4Regex         = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	ipv6Regex         = regexp.MustCompile(`^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$|^([0-9a-fA-F]{1,4}:){1,7}:$|^([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}$|^([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}$|^([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}$|^([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}$|^([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}$|^[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})$|^:((:[0-9a-fA-F]{1,4}){1,7}|:)$`)
	macRegex          = regexp.MustCompile(`^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`)
	hostnameRegex     = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$`)
	fqdnRegex         = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
	e164Regex         = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	semverRegex       = regexp.MustCompile(`^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)
	slugRegex         = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	jwtRegex          = regexp.MustCompile(`^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`)
)
