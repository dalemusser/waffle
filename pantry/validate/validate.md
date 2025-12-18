# Validate Package

The `validate` package provides struct validation using struct tags with support for custom validators and internationalized error messages.

## Features

- **Struct Tag Validation**: Validate structs using `validate:"..."` tags
- **70+ Built-in Rules**: Required, email, URL, UUID, min/max, regex, and more
- **Custom Validators**: Register your own validation functions
- **i18n Support**: Error messages in 7 languages (EN, ES, FR, DE, PT, ZH, JA)
- **Field Comparison**: Compare fields with `eqfield`, `gtfield`, etc.
- **Nested Structs**: Automatic validation of nested structures
- **Slice Validation**: Validate slices with unique values

## Installation

The validate package is part of the waffle pantry:

```go
import "github.com/dalemusser/waffle/pantry/validate"
```

## Quick Start

```go
type User struct {
    Name     string `json:"name" validate:"required,min=2,max=50"`
    Email    string `json:"email" validate:"required,email"`
    Age      int    `json:"age" validate:"min=18,max=120"`
    Password string `json:"password" validate:"required,strongpassword"`
    Role     string `json:"role" validate:"oneof=admin user guest"`
}

func main() {
    user := User{
        Name:     "J",
        Email:    "invalid",
        Age:      15,
        Password: "weak",
        Role:     "superuser",
    }

    v := validate.New()
    if err := v.Struct(user); err != nil {
        for _, e := range err.(validate.Errors) {
            fmt.Printf("%s: %s\n", e.Field, e.Message)
        }
    }
}

// Output:
// name: name must be at least 2
// email: email must be a valid email address
// age: age must be at least 18
// password: password must be a strong password (min 8 chars, uppercase, lowercase, number, special char)
// role: role must be one of [admin user guest]
```

## Using the Default Validator

For simple cases, use the package-level functions:

```go
// Validate a struct
err := validate.Struct(user)

// Validate a single value
err := validate.Var("test@email.com", "required,email")

// Register a custom rule
validate.RegisterRule("customrule", myRuleFunc)
```

## Creating a Custom Validator

```go
v := validate.New(
    validate.WithTagName("valid"),           // Use `valid:"..."` instead of `validate:"..."`
    validate.WithStopOnFirstError(),         // Stop after first error
    validate.WithMessages(customMessages),   // Custom message provider
)

if err := v.Struct(user); err != nil {
    // Handle errors
}
```

## Validation Rules

### Required

```go
Name string `validate:"required"` // Must not be empty
```

### Type Validations

```go
Email       string `validate:"email"`        // Valid email address
URL         string `validate:"url"`          // Valid HTTP/HTTPS URL
URI         string `validate:"uri"`          // Valid URI
UUID        string `validate:"uuid"`         // Valid UUID (any version)
UUID4       string `validate:"uuid4"`        // Valid UUID v4
ULID        string `validate:"ulid"`         // Valid ULID
Alpha       string `validate:"alpha"`        // Letters only
AlphaNum    string `validate:"alphanum"`     // Letters and numbers only
Numeric     string `validate:"numeric"`      // Numbers only (string)
Hexadecimal string `validate:"hexadecimal"`  // Hex characters only
HexColor    string `validate:"hexcolor"`     // Valid hex color (#RGB or #RRGGBB)
RGB         string `validate:"rgb"`          // Valid rgb(r,g,b)
RGBA        string `validate:"rgba"`         // Valid rgba(r,g,b,a)
HSL         string `validate:"hsl"`          // Valid hsl(h,s%,l%)
HSLA        string `validate:"hsla"`         // Valid hsla(h,s%,l%,a)
JSON        string `validate:"json"`         // Valid JSON string
JWT         string `validate:"jwt"`          // Valid JWT token format
Base64      string `validate:"base64"`       // Valid base64 string
Base64URL   string `validate:"base64url"`    // Valid base64url string
ISBN        string `validate:"isbn"`         // Valid ISBN-10 or ISBN-13
ISBN10      string `validate:"isbn10"`       // Valid ISBN-10
ISBN13      string `validate:"isbn13"`       // Valid ISBN-13
ISSN        string `validate:"issn"`         // Valid ISSN
ASCII       string `validate:"ascii"`        // ASCII characters only
PrintASCII  string `validate:"printascii"`   // Printable ASCII only
Lowercase   string `validate:"lowercase"`    // Must be lowercase
Uppercase   string `validate:"uppercase"`    // Must be uppercase
```

### Length/Size Validations

```go
// For strings: character count
// For slices/maps/arrays: element count
// For numbers: the value itself

Name   string   `validate:"min=2"`        // At least 2 characters
Name   string   `validate:"max=50"`       // At most 50 characters
Name   string   `validate:"len=10"`       // Exactly 10 characters
Name   string   `validate:"between=2|50"` // Between 2 and 50 characters

Age    int      `validate:"min=18"`       // Value at least 18
Age    int      `validate:"max=120"`      // Value at most 120
Age    int      `validate:"between=18|120"` // Value between 18 and 120

Tags   []string `validate:"min=1,max=5"`  // 1 to 5 elements
```

### Comparison Validations

```go
Status string `validate:"eq=active"`   // Must equal "active"
Status string `validate:"ne=deleted"`  // Must not equal "deleted"

Age int `validate:"gt=17"`   // Greater than 17
Age int `validate:"gte=18"`  // Greater than or equal to 18
Age int `validate:"lt=121"`  // Less than 121
Age int `validate:"lte=120"` // Less than or equal to 120
```

### Field Comparison Validations

```go
type Form struct {
    Password        string `validate:"required,min=8"`
    ConfirmPassword string `validate:"required,eqfield=Password"`

    StartDate time.Time `validate:"required"`
    EndDate   time.Time `validate:"required,gtfield=StartDate"`

    Min int `validate:"required"`
    Max int `validate:"required,gtefield=Min"`
}
```

Available field comparisons:
- `eqfield=Field` - Must equal Field
- `nefield=Field` - Must not equal Field
- `gtfield=Field` - Must be greater than Field
- `gtefield=Field` - Must be greater than or equal to Field
- `ltfield=Field` - Must be less than Field
- `ltefield=Field` - Must be less than or equal to Field

### Network Validations

```go
IP       string `validate:"ip"`      // Valid IPv4 or IPv6
IPv4     string `validate:"ipv4"`    // Valid IPv4 only
IPv6     string `validate:"ipv6"`    // Valid IPv6 only
CIDR     string `validate:"cidr"`    // Valid CIDR notation
CIDRv4   string `validate:"cidrv4"`  // Valid CIDR v4
CIDRv6   string `validate:"cidrv6"`  // Valid CIDR v6
MAC      string `validate:"mac"`     // Valid MAC address
Hostname string `validate:"hostname"` // Valid hostname
FQDN     string `validate:"fqdn"`    // Fully qualified domain name
```

### String Validations

```go
Bio string `validate:"contains=hello"`      // Must contain "hello"
Bio string `validate:"containsany=aeiou"`   // Must contain any vowel
Bio string `validate:"excludes=spam"`       // Must not contain "spam"
Bio string `validate:"excludesall=<>"`      // Must not contain < or >
Bio string `validate:"startswith=Hello"`    // Must start with "Hello"
Bio string `validate:"endswith=."`          // Must end with "."
Bio string `validate:"startsnotwith=http"`  // Must not start with "http"
Bio string `validate:"endsnotwith=!"`       // Must not end with "!"
```

### Format Validations

```go
Code     string `validate:"regex=^[A-Z]{3}[0-9]{4}$"` // Match pattern
DateTime string `validate:"datetime=2006-01-02"`      // Parse with layout
Date     string `validate:"date"`                      // Valid date
Time     string `validate:"time"`                      // Valid time
Timezone string `validate:"timezone"`                  // Valid timezone (e.g., "America/New_York")
Duration string `validate:"duration"`                  // Valid duration (e.g., "1h30m")
```

### Enum/OneOf Validation

```go
// Values separated by spaces
Status   string `validate:"oneof=pending active completed"`
Priority string `validate:"enum=low medium high critical"`
```

### Special Validations

```go
Tags []string `validate:"unique"` // All elements must be unique

// Optional field - skip validation if empty
Email string `validate:"omitempty,email"`
```

### Credit Card & Payment

```go
CardNumber string `validate:"creditcard"` // Valid credit card (Luhn algorithm)
CVV        string `validate:"cvv"`        // Valid CVV (3-4 digits)
```

### Phone Numbers

```go
Phone string `validate:"e164"` // E.164 format (+1234567890)
```

### Geographic

```go
Lat  float64 `validate:"latitude"`  // -90 to 90
Long float64 `validate:"longitude"` // -180 to 180
```

### Country & Language Codes

```go
Country  string `validate:"countrycode"`  // ISO 3166-1 alpha-2 (US, GB, etc.)
Language string `validate:"languagecode"` // ISO 639-1 (en, es, fr, etc.)
Locale   string `validate:"bcp47"`        // BCP 47 tag (en-US, zh-Hans, etc.)
```

### Other Validations

```go
Version    string `validate:"semver"`         // Semantic version (1.2.3, v1.0.0-alpha)
Slug       string `validate:"slug"`           // URL slug (lowercase, hyphens)
PostalCode string `validate:"postcode"`       // Postal/ZIP code
FilePath   string `validate:"filepath"`       // Valid file path
DirPath    string `validate:"dirpath"`        // Valid directory path
Boolean    string `validate:"boolean"`        // Boolean string (true, false, 1, 0, yes, no)
Password   string `validate:"strongpassword"` // Strong password
```

## Custom Validators

### Simple Custom Rule

```go
v := validate.New()

// Register a simple rule
v.RegisterRuleFunc("even", func(value any) bool {
    if n, ok := value.(int); ok {
        return n%2 == 0
    }
    return false
}, "even")

// Add custom message
v.SetMessages(customMessages)
```

### Advanced Custom Rule

```go
v.RegisterRule("notempty", func(value any, param string, structVal reflect.Value) string {
    s, ok := value.(string)
    if !ok {
        return ""
    }
    if strings.TrimSpace(s) == "" {
        return "notempty" // Return message key
    }
    return "" // Empty string = valid
})
```

### Using Struct Context

```go
v.RegisterRule("confirmmatch", func(value any, param string, structVal reflect.Value) string {
    // Access other fields in the struct
    otherField := structVal.FieldByName(param)
    if !otherField.IsValid() {
        return ""
    }

    if value != otherField.Interface() {
        return "confirmmatch"
    }
    return ""
})

type Form struct {
    Password string `validate:"required"`
    Confirm  string `validate:"confirmmatch=Password"`
}
```

## Internationalization (i18n)

### Using Built-in Locales

```go
// Create validator with Spanish messages
messages := validate.MessagesForLocale("es")
v := validate.New(validate.WithMessages(messages))

// Available locales: en, es, fr, de, pt, zh, ja
```

### Changing Locale at Runtime

```go
messages := validate.DefaultMessages()
messages.RegisterBuiltinLocales() // Register all locales
messages.SetLocale("fr")          // Switch to French

v := validate.New(validate.WithMessages(messages))
```

### Custom Messages

```go
messages := validate.NewMessageProvider()

// Register English messages
messages.RegisterLocale("en", map[string]string{
    "required": "{field} cannot be empty",
    "email":    "{field} is not a valid email",
    "min":      "{field} must have at least {param} characters",
    "between":  "{field} must be between {min} and {max}",
})

// Register Spanish messages
messages.RegisterLocale("es", map[string]string{
    "required": "{field} no puede estar vacío",
    "email":    "{field} no es un correo válido",
})

messages.SetLocale("es")
v := validate.New(validate.WithMessages(messages))
```

### Adding Messages for Custom Rules

```go
messages := validate.DefaultMessages()
messages.AddMessage("en", "even", "{field} must be an even number")
messages.AddMessage("es", "even", "{field} debe ser un número par")
```

## Error Handling

### Checking for Errors

```go
err := v.Struct(user)
if err != nil {
    errs := err.(validate.Errors)

    // Check if there are errors
    if errs.HasErrors() {
        // Process errors
    }

    // Get first error
    if first := errs.First(); first != nil {
        fmt.Println(first.Message)
    }

    // Get errors for a specific field
    for _, e := range errs.FieldErrors("email") {
        fmt.Println(e.Message)
    }

    // Convert to map for JSON response
    errorMap := errs.ToMap()
    // {"email": ["email is required", "email must be valid"]}
}
```

### Error Structure

```go
type Error struct {
    Field   string // Field name (uses json tag if available)
    Rule    string // Rule name (e.g., "required", "email")
    Param   string // Rule parameter (e.g., "18" for min=18)
    Value   any    // The actual value that failed validation
    Message string // Formatted error message
}
```

### JSON API Response

```go
func handleValidation(w http.ResponseWriter, r *http.Request) {
    var user User
    json.NewDecoder(r.Body).Decode(&user)

    if err := validate.Struct(user); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]any{
            "error":  "validation_failed",
            "fields": err.(validate.Errors).ToMap(),
        })
        return
    }

    // Process valid user...
}
```

## Nested Struct Validation

```go
type Address struct {
    Street  string `validate:"required"`
    City    string `validate:"required"`
    ZipCode string `validate:"required,postcode"`
}

type User struct {
    Name    string   `validate:"required"`
    Address Address  // Automatically validated
}

// Errors will have field names like "Address.City"
```

## Pointer and Slice Validation

```go
type Order struct {
    ID     string  `validate:"required,uuid"`
    Items  []Item  // Each item is validated
    Coupon *Coupon // Validated if not nil
}

type Item struct {
    SKU      string `validate:"required"`
    Quantity int    `validate:"required,min=1"`
}

// Errors will have field names like "Items[0].SKU"
```

## Conditional Validation (omitempty)

```go
type Profile struct {
    // Required field
    Name string `validate:"required"`

    // Optional, but if provided must be valid
    Email   string `validate:"omitempty,email"`
    Website string `validate:"omitempty,url"`
    Age     int    `validate:"omitempty,min=18"`
}

profile := Profile{Name: "John"} // Valid - optional fields empty
profile := Profile{Name: "John", Email: "invalid"} // Invalid - email provided but invalid
```

## Legacy API

The package also provides the original simple validators:

### SimpleEmailValid

```go
func SimpleEmailValid(s string) bool
```

Reports whether `s` looks like a valid email address. This is a lightweight check, not an RFC 5322 validator.

```go
validate.SimpleEmailValid("user@example.com")     // true
validate.SimpleEmailValid("user@localhost")       // false (no dot in domain)
validate.SimpleEmailValid("userexample.com")      // false (no @)
```

## Complete Example

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/dalemusser/waffle/pantry/validate"
)

type CreateUserRequest struct {
    Username        string `json:"username" validate:"required,min=3,max=20,alphanum"`
    Email           string `json:"email" validate:"required,email"`
    Password        string `json:"password" validate:"required,strongpassword"`
    ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=Password"`
    Age             int    `json:"age" validate:"omitempty,min=13,max=120"`
    Country         string `json:"country" validate:"omitempty,countrycode"`
    Role            string `json:"role" validate:"required,oneof=user admin moderator"`
}

var validator = validate.New()

func init() {
    // Set up Spanish messages
    messages := validate.MessagesForLocale("es")
    validator.SetMessages(messages)
}

func createUser(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    if err := validator.Struct(req); err != nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusUnprocessableEntity)
        json.NewEncoder(w).Encode(map[string]any{
            "success": false,
            "errors":  err.(validate.Errors).ToMap(),
        })
        return
    }

    // Create user...
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]any{
        "success": true,
        "message": "User created",
    })
}

func main() {
    http.HandleFunc("/users", createUser)
    http.ListenAndServe(":8080", nil)
}
```

## Best Practices

1. **Use JSON Tags**: Field names in errors use JSON tags when available
2. **Use omitempty**: For optional fields that should only be validated when provided
3. **Validate Early**: Validate input as early as possible in your handlers
4. **Custom Messages**: Customize messages for better UX
5. **Reuse Validators**: Create one validator instance and reuse it
6. **Stop on First Error**: Use `WithStopOnFirstError()` for form validation where you only need to show one error per field
