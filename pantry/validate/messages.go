package validate

import (
	"fmt"
	"strings"
	"sync"
)

// MessageProvider provides validation error messages with i18n support.
type MessageProvider struct {
	mu       sync.RWMutex
	messages map[string]map[string]string // locale -> key -> message
	locale   string
	fallback string
}

// NewMessageProvider creates a new message provider.
func NewMessageProvider() *MessageProvider {
	return &MessageProvider{
		messages: make(map[string]map[string]string),
		locale:   "en",
		fallback: "en",
	}
}

// DefaultMessages returns a message provider with default English messages.
func DefaultMessages() *MessageProvider {
	m := NewMessageProvider()
	m.RegisterLocale("en", defaultEnglishMessages)
	return m
}

// SetLocale sets the current locale.
func (m *MessageProvider) SetLocale(locale string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.locale = locale
}

// SetFallback sets the fallback locale.
func (m *MessageProvider) SetFallback(locale string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fallback = locale
}

// RegisterLocale registers messages for a locale.
func (m *MessageProvider) RegisterLocale(locale string, messages map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages[locale] = messages
}

// AddMessage adds or updates a message for a locale.
func (m *MessageProvider) AddMessage(locale, key, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.messages[locale] == nil {
		m.messages[locale] = make(map[string]string)
	}
	m.messages[locale][key] = message
}

// Get retrieves a message for the current locale.
// It supports placeholders: {field} for field name, {param} for parameter.
func (m *MessageProvider) Get(key, field, param string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try current locale
	if msgs, ok := m.messages[m.locale]; ok {
		if msg, ok := msgs[key]; ok {
			return m.format(msg, field, param)
		}
	}

	// Try fallback locale
	if msgs, ok := m.messages[m.fallback]; ok {
		if msg, ok := msgs[key]; ok {
			return m.format(msg, field, param)
		}
	}

	// Return generic message
	return fmt.Sprintf("%s validation failed for %s", key, field)
}

// format replaces placeholders in a message.
func (m *MessageProvider) format(msg, field, param string) string {
	msg = strings.ReplaceAll(msg, "{field}", field)
	msg = strings.ReplaceAll(msg, "{param}", param)

	// Handle param with multiple values (e.g., "1|10" for between)
	if strings.Contains(param, "|") {
		parts := strings.Split(param, "|")
		if len(parts) >= 1 {
			msg = strings.ReplaceAll(msg, "{min}", parts[0])
		}
		if len(parts) >= 2 {
			msg = strings.ReplaceAll(msg, "{max}", parts[1])
		}
	}

	return msg
}

// Clone creates a copy of the message provider.
func (m *MessageProvider) Clone() *MessageProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clone := NewMessageProvider()
	clone.locale = m.locale
	clone.fallback = m.fallback

	for locale, msgs := range m.messages {
		clone.messages[locale] = make(map[string]string)
		for k, v := range msgs {
			clone.messages[locale][k] = v
		}
	}

	return clone
}

// Default English messages
var defaultEnglishMessages = map[string]string{
	// Required
	"required": "{field} is required",

	// Type validations
	"email":         "{field} must be a valid email address",
	"url":           "{field} must be a valid URL",
	"uri":           "{field} must be a valid URI",
	"uuid":          "{field} must be a valid UUID",
	"uuid3":         "{field} must be a valid UUID v3",
	"uuid4":         "{field} must be a valid UUID v4",
	"uuid5":         "{field} must be a valid UUID v5",
	"ulid":          "{field} must be a valid ULID",
	"alpha":         "{field} must contain only letters",
	"alphanum":      "{field} must contain only letters and numbers",
	"alphanumspace": "{field} must contain only letters, numbers, and spaces",
	"numeric":       "{field} must be a number",
	"hexadecimal":   "{field} must be a valid hexadecimal value",
	"hexcolor":      "{field} must be a valid hex color",
	"rgb":           "{field} must be a valid RGB color",
	"rgba":          "{field} must be a valid RGBA color",
	"hsl":           "{field} must be a valid HSL color",
	"hsla":          "{field} must be a valid HSLA color",
	"json":          "{field} must be valid JSON",
	"jwt":           "{field} must be a valid JWT token",
	"base64":        "{field} must be valid Base64",
	"base64url":     "{field} must be valid Base64URL",
	"isbn":          "{field} must be a valid ISBN",
	"isbn10":        "{field} must be a valid ISBN-10",
	"isbn13":        "{field} must be a valid ISBN-13",
	"issn":          "{field} must be a valid ISSN",
	"ascii":         "{field} must contain only ASCII characters",
	"printascii":    "{field} must contain only printable ASCII characters",
	"multibyte":     "{field} must contain multibyte characters",
	"lowercase":     "{field} must be lowercase",
	"uppercase":     "{field} must be uppercase",

	// Length/size validations
	"min":     "{field} must be at least {param}",
	"max":     "{field} must be at most {param}",
	"len":     "{field} must be exactly {param}",
	"between": "{field} must be between {min} and {max}",

	// Comparison validations
	"eq":  "{field} must equal {param}",
	"ne":  "{field} must not equal {param}",
	"gt":  "{field} must be greater than {param}",
	"gte": "{field} must be greater than or equal to {param}",
	"lt":  "{field} must be less than {param}",
	"lte": "{field} must be less than or equal to {param}",

	// Field comparison validations
	"eqfield":  "{field} must equal {param}",
	"nefield":  "{field} must not equal {param}",
	"gtfield":  "{field} must be greater than {param}",
	"gtefield": "{field} must be greater than or equal to {param}",
	"ltfield":  "{field} must be less than {param}",
	"ltefield": "{field} must be less than or equal to {param}",

	// Network validations
	"ip":       "{field} must be a valid IP address",
	"ipv4":     "{field} must be a valid IPv4 address",
	"ipv6":     "{field} must be a valid IPv6 address",
	"cidr":     "{field} must be a valid CIDR notation",
	"cidrv4":   "{field} must be a valid CIDR v4 notation",
	"cidrv6":   "{field} must be a valid CIDR v6 notation",
	"mac":      "{field} must be a valid MAC address",
	"hostname": "{field} must be a valid hostname",
	"fqdn":     "{field} must be a valid FQDN",

	// String validations
	"contains":      "{field} must contain '{param}'",
	"containsany":   "{field} must contain at least one of '{param}'",
	"containsrune":  "{field} must contain the character '{param}'",
	"excludes":      "{field} must not contain '{param}'",
	"excludesall":   "{field} must not contain any of '{param}'",
	"excludesrune":  "{field} must not contain the character '{param}'",
	"startswith":    "{field} must start with '{param}'",
	"endswith":      "{field} must end with '{param}'",
	"startsnotwith": "{field} must not start with '{param}'",
	"endsnotwith":   "{field} must not end with '{param}'",

	// Format validations
	"regex":    "{field} must match the pattern {param}",
	"datetime": "{field} must be a valid datetime in format {param}",
	"date":     "{field} must be a valid date",
	"time":     "{field} must be a valid time",
	"timezone": "{field} must be a valid timezone",
	"duration": "{field} must be a valid duration",

	// Special validations
	"oneof":  "{field} must be one of [{param}]",
	"enum":   "{field} must be one of [{param}]",
	"unique": "{field} must contain unique values",

	// Credit card
	"creditcard": "{field} must be a valid credit card number",

	// Phone
	"e164": "{field} must be a valid E.164 phone number",

	// Country/language codes
	"countrycode":  "{field} must be a valid country code",
	"languagecode": "{field} must be a valid language code",
	"bcp47":        "{field} must be a valid BCP 47 language tag",

	// File validations
	"filepath": "{field} must be a valid file path",
	"dirpath":  "{field} must be a valid directory path",

	// Semantic versioning
	"semver": "{field} must be a valid semantic version",

	// Boolean string
	"boolean": "{field} must be a boolean value",

	// CVV
	"cvv": "{field} must be a valid CVV",

	// Latitude/Longitude
	"latitude":  "{field} must be a valid latitude",
	"longitude": "{field} must be a valid longitude",

	// Postal code
	"postcode": "{field} must be a valid postal code",

	// Slug
	"slug": "{field} must be a valid URL slug",

	// Strong password
	"strongpassword": "{field} must be a strong password (min 8 chars, uppercase, lowercase, number, special char)",
}

// Spanish messages
var spanishMessages = map[string]string{
	"required":       "{field} es obligatorio",
	"email":          "{field} debe ser una dirección de correo válida",
	"url":            "{field} debe ser una URL válida",
	"min":            "{field} debe ser al menos {param}",
	"max":            "{field} debe ser como máximo {param}",
	"len":            "{field} debe tener exactamente {param}",
	"between":        "{field} debe estar entre {min} y {max}",
	"eq":             "{field} debe ser igual a {param}",
	"ne":             "{field} no debe ser igual a {param}",
	"gt":             "{field} debe ser mayor que {param}",
	"gte":            "{field} debe ser mayor o igual que {param}",
	"lt":             "{field} debe ser menor que {param}",
	"lte":            "{field} debe ser menor o igual que {param}",
	"alpha":          "{field} solo debe contener letras",
	"alphanum":       "{field} solo debe contener letras y números",
	"numeric":        "{field} debe ser un número",
	"oneof":          "{field} debe ser uno de [{param}]",
	"uuid":           "{field} debe ser un UUID válido",
	"ip":             "{field} debe ser una dirección IP válida",
	"ipv4":           "{field} debe ser una dirección IPv4 válida",
	"ipv6":           "{field} debe ser una dirección IPv6 válida",
	"creditcard":     "{field} debe ser un número de tarjeta de crédito válido",
	"strongpassword": "{field} debe ser una contraseña segura (mín 8 caracteres, mayúscula, minúscula, número, carácter especial)",
}

// French messages
var frenchMessages = map[string]string{
	"required":       "{field} est obligatoire",
	"email":          "{field} doit être une adresse email valide",
	"url":            "{field} doit être une URL valide",
	"min":            "{field} doit être au moins {param}",
	"max":            "{field} doit être au maximum {param}",
	"len":            "{field} doit être exactement {param}",
	"between":        "{field} doit être entre {min} et {max}",
	"eq":             "{field} doit être égal à {param}",
	"ne":             "{field} ne doit pas être égal à {param}",
	"gt":             "{field} doit être supérieur à {param}",
	"gte":            "{field} doit être supérieur ou égal à {param}",
	"lt":             "{field} doit être inférieur à {param}",
	"lte":            "{field} doit être inférieur ou égal à {param}",
	"alpha":          "{field} ne doit contenir que des lettres",
	"alphanum":       "{field} ne doit contenir que des lettres et des chiffres",
	"numeric":        "{field} doit être un nombre",
	"oneof":          "{field} doit être l'un des [{param}]",
	"uuid":           "{field} doit être un UUID valide",
	"ip":             "{field} doit être une adresse IP valide",
	"creditcard":     "{field} doit être un numéro de carte de crédit valide",
	"strongpassword": "{field} doit être un mot de passe fort (min 8 caractères, majuscule, minuscule, chiffre, caractère spécial)",
}

// German messages
var germanMessages = map[string]string{
	"required":       "{field} ist erforderlich",
	"email":          "{field} muss eine gültige E-Mail-Adresse sein",
	"url":            "{field} muss eine gültige URL sein",
	"min":            "{field} muss mindestens {param} sein",
	"max":            "{field} darf höchstens {param} sein",
	"len":            "{field} muss genau {param} sein",
	"between":        "{field} muss zwischen {min} und {max} liegen",
	"eq":             "{field} muss gleich {param} sein",
	"ne":             "{field} darf nicht gleich {param} sein",
	"gt":             "{field} muss größer als {param} sein",
	"gte":            "{field} muss größer oder gleich {param} sein",
	"lt":             "{field} muss kleiner als {param} sein",
	"lte":            "{field} muss kleiner oder gleich {param} sein",
	"alpha":          "{field} darf nur Buchstaben enthalten",
	"alphanum":       "{field} darf nur Buchstaben und Zahlen enthalten",
	"numeric":        "{field} muss eine Zahl sein",
	"oneof":          "{field} muss einer von [{param}] sein",
	"uuid":           "{field} muss eine gültige UUID sein",
	"ip":             "{field} muss eine gültige IP-Adresse sein",
	"creditcard":     "{field} muss eine gültige Kreditkartennummer sein",
	"strongpassword": "{field} muss ein starkes Passwort sein (min 8 Zeichen, Großbuchstabe, Kleinbuchstabe, Zahl, Sonderzeichen)",
}

// Portuguese messages
var portugueseMessages = map[string]string{
	"required":       "{field} é obrigatório",
	"email":          "{field} deve ser um endereço de email válido",
	"url":            "{field} deve ser uma URL válida",
	"min":            "{field} deve ter no mínimo {param}",
	"max":            "{field} deve ter no máximo {param}",
	"len":            "{field} deve ter exatamente {param}",
	"between":        "{field} deve estar entre {min} e {max}",
	"eq":             "{field} deve ser igual a {param}",
	"ne":             "{field} não deve ser igual a {param}",
	"gt":             "{field} deve ser maior que {param}",
	"gte":            "{field} deve ser maior ou igual a {param}",
	"lt":             "{field} deve ser menor que {param}",
	"lte":            "{field} deve ser menor ou igual a {param}",
	"alpha":          "{field} deve conter apenas letras",
	"alphanum":       "{field} deve conter apenas letras e números",
	"numeric":        "{field} deve ser um número",
	"oneof":          "{field} deve ser um de [{param}]",
	"uuid":           "{field} deve ser um UUID válido",
	"ip":             "{field} deve ser um endereço IP válido",
	"creditcard":     "{field} deve ser um número de cartão de crédito válido",
	"strongpassword": "{field} deve ser uma senha forte (mín 8 caracteres, maiúscula, minúscula, número, caractere especial)",
}

// Chinese (Simplified) messages
var chineseMessages = map[string]string{
	"required":       "{field}是必填项",
	"email":          "{field}必须是有效的电子邮件地址",
	"url":            "{field}必须是有效的URL",
	"min":            "{field}必须至少为{param}",
	"max":            "{field}必须最多为{param}",
	"len":            "{field}必须正好为{param}",
	"between":        "{field}必须在{min}和{max}之间",
	"eq":             "{field}必须等于{param}",
	"ne":             "{field}不能等于{param}",
	"gt":             "{field}必须大于{param}",
	"gte":            "{field}必须大于或等于{param}",
	"lt":             "{field}必须小于{param}",
	"lte":            "{field}必须小于或等于{param}",
	"alpha":          "{field}只能包含字母",
	"alphanum":       "{field}只能包含字母和数字",
	"numeric":        "{field}必须是数字",
	"oneof":          "{field}必须是[{param}]之一",
	"uuid":           "{field}必须是有效的UUID",
	"ip":             "{field}必须是有效的IP地址",
	"creditcard":     "{field}必须是有效的信用卡号",
	"strongpassword": "{field}必须是强密码（至少8个字符，大写字母，小写字母，数字，特殊字符）",
}

// Japanese messages
var japaneseMessages = map[string]string{
	"required":       "{field}は必須です",
	"email":          "{field}は有効なメールアドレスである必要があります",
	"url":            "{field}は有効なURLである必要があります",
	"min":            "{field}は{param}以上である必要があります",
	"max":            "{field}は{param}以下である必要があります",
	"len":            "{field}は正確に{param}である必要があります",
	"between":        "{field}は{min}から{max}の間である必要があります",
	"eq":             "{field}は{param}と等しい必要があります",
	"ne":             "{field}は{param}と等しくない必要があります",
	"gt":             "{field}は{param}より大きい必要があります",
	"gte":            "{field}は{param}以上である必要があります",
	"lt":             "{field}は{param}より小さい必要があります",
	"lte":            "{field}は{param}以下である必要があります",
	"alpha":          "{field}は文字のみを含む必要があります",
	"alphanum":       "{field}は文字と数字のみを含む必要があります",
	"numeric":        "{field}は数字である必要があります",
	"oneof":          "{field}は[{param}]のいずれかである必要があります",
	"uuid":           "{field}は有効なUUIDである必要があります",
	"ip":             "{field}は有効なIPアドレスである必要があります",
	"creditcard":     "{field}は有効なクレジットカード番号である必要があります",
	"strongpassword": "{field}は強力なパスワードである必要があります（8文字以上、大文字、小文字、数字、特殊文字）",
}

// RegisterBuiltinLocales registers all built-in locales.
func (m *MessageProvider) RegisterBuiltinLocales() {
	m.RegisterLocale("en", defaultEnglishMessages)
	m.RegisterLocale("es", spanishMessages)
	m.RegisterLocale("fr", frenchMessages)
	m.RegisterLocale("de", germanMessages)
	m.RegisterLocale("pt", portugueseMessages)
	m.RegisterLocale("zh", chineseMessages)
	m.RegisterLocale("ja", japaneseMessages)
}

// MessagesForLocale creates a message provider for a specific locale.
func MessagesForLocale(locale string) *MessageProvider {
	m := NewMessageProvider()
	m.RegisterBuiltinLocales()
	m.SetLocale(locale)
	return m
}
