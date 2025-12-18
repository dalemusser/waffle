package validate

import (
	"encoding/json"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Required validation rules

func ruleRequired(value any, param string, sv reflect.Value) string {
	if value == nil {
		return "required"
	}

	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.String:
		if strings.TrimSpace(val.String()) == "" {
			return "required"
		}
	case reflect.Slice, reflect.Map, reflect.Array:
		if val.Len() == 0 {
			return "required"
		}
	case reflect.Ptr, reflect.Interface:
		if val.IsNil() {
			return "required"
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Zero is considered valid for numbers unless explicitly required
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// Zero is considered valid for numbers unless explicitly required
	case reflect.Float32, reflect.Float64:
		// Zero is considered valid for numbers unless explicitly required
	case reflect.Bool:
		// False is considered valid for bool
	}

	return ""
}

// Type validation rules

func ruleEmail(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !emailRegex.MatchString(s) {
		return "email"
	}
	return ""
}

func ruleURL(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	u, err := url.Parse(s)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "url"
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "url"
	}
	return ""
}

func ruleURI(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	_, err := url.Parse(s)
	if err != nil {
		return "uri"
	}
	return ""
}

func ruleUUID(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !uuidRegex.MatchString(s) {
		return "uuid"
	}
	return ""
}

func ruleUUID3(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !uuid3Regex.MatchString(s) {
		return "uuid3"
	}
	return ""
}

func ruleUUID4(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !uuid4Regex.MatchString(s) {
		return "uuid4"
	}
	return ""
}

func ruleUUID5(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !uuid5Regex.MatchString(s) {
		return "uuid5"
	}
	return ""
}

func ruleULID(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !ulidRegex.MatchString(s) {
		return "ulid"
	}
	return ""
}

func ruleAlpha(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !alphaRegex.MatchString(s) {
		return "alpha"
	}
	return ""
}

func ruleAlphaNum(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !alphaNumRegex.MatchString(s) {
		return "alphanum"
	}
	return ""
}

func ruleAlphaNumSpace(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !alphaNumSpaceRegex.MatchString(s) {
		return "alphanumspace"
	}
	return ""
}

func ruleNumeric(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !numericRegex.MatchString(s) {
		return "numeric"
	}
	return ""
}

func ruleHexadecimal(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !hexRegex.MatchString(s) {
		return "hexadecimal"
	}
	return ""
}

func ruleHexColor(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !hexColorRegex.MatchString(s) {
		return "hexcolor"
	}
	return ""
}

func ruleRGB(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !rgbRegex.MatchString(s) {
		return "rgb"
	}
	return ""
}

func ruleRGBA(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !rgbaRegex.MatchString(s) {
		return "rgba"
	}
	return ""
}

func ruleHSL(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !hslRegex.MatchString(s) {
		return "hsl"
	}
	return ""
}

func ruleHSLA(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !hslaRegex.MatchString(s) {
		return "hsla"
	}
	return ""
}

func ruleJSON(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	var js json.RawMessage
	if json.Unmarshal([]byte(s), &js) != nil {
		return "json"
	}
	return ""
}

func ruleJWT(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !jwtRegex.MatchString(s) {
		return "jwt"
	}
	return ""
}

func ruleBase64(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if len(s)%4 != 0 {
		return "base64"
	}
	if !base64Regex.MatchString(s) {
		return "base64"
	}
	return ""
}

func ruleBase64URL(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !base64URLRegex.MatchString(s) {
		return "base64url"
	}
	return ""
}

func ruleISBN(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	// Remove hyphens
	s = strings.ReplaceAll(s, "-", "")
	if len(s) == 10 {
		return ruleISBN10(s, param, sv)
	}
	if len(s) == 13 {
		return ruleISBN13(s, param, sv)
	}
	return "isbn"
}

func ruleISBN10(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "-", "")
	if !isbn10Regex.MatchString(s) {
		return "isbn10"
	}
	// Validate checksum
	var sum int
	for i := 0; i < 9; i++ {
		sum += int(s[i]-'0') * (10 - i)
	}
	check := s[9]
	if check == 'X' {
		sum += 10
	} else {
		sum += int(check - '0')
	}
	if sum%11 != 0 {
		return "isbn10"
	}
	return ""
}

func ruleISBN13(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "-", "")
	if !isbn13Regex.MatchString(s) {
		return "isbn13"
	}
	// Validate checksum
	var sum int
	for i := 0; i < 12; i++ {
		digit := int(s[i] - '0')
		if i%2 == 0 {
			sum += digit
		} else {
			sum += digit * 3
		}
	}
	check := (10 - (sum % 10)) % 10
	if int(s[12]-'0') != check {
		return "isbn13"
	}
	return ""
}

func ruleISSN(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !issnRegex.MatchString(s) {
		return "issn"
	}
	return ""
}

func ruleASCII(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	for _, r := range s {
		if r > 127 {
			return "ascii"
		}
	}
	return ""
}

func rulePrintASCII(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	for _, r := range s {
		if r < 32 || r > 126 {
			return "printascii"
		}
	}
	return ""
}

func ruleMultibyte(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	for _, r := range s {
		if r > 127 {
			return ""
		}
	}
	return "multibyte"
}

func ruleLowercase(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	for _, r := range s {
		if unicode.IsLetter(r) && !unicode.IsLower(r) {
			return "lowercase"
		}
	}
	return ""
}

func ruleUppercase(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	for _, r := range s {
		if unicode.IsLetter(r) && !unicode.IsUpper(r) {
			return "uppercase"
		}
	}
	return ""
}

// Length/size validation rules

func ruleMin(value any, param string, sv reflect.Value) string {
	if value == nil {
		return ""
	}

	min, err := strconv.ParseFloat(param, 64)
	if err != nil {
		return ""
	}

	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.String:
		if float64(len([]rune(val.String()))) < min {
			return "min"
		}
	case reflect.Slice, reflect.Map, reflect.Array:
		if float64(val.Len()) < min {
			return "min"
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if float64(val.Int()) < min {
			return "min"
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if float64(val.Uint()) < min {
			return "min"
		}
	case reflect.Float32, reflect.Float64:
		if val.Float() < min {
			return "min"
		}
	}

	return ""
}

func ruleMax(value any, param string, sv reflect.Value) string {
	if value == nil {
		return ""
	}

	max, err := strconv.ParseFloat(param, 64)
	if err != nil {
		return ""
	}

	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.String:
		if float64(len([]rune(val.String()))) > max {
			return "max"
		}
	case reflect.Slice, reflect.Map, reflect.Array:
		if float64(val.Len()) > max {
			return "max"
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if float64(val.Int()) > max {
			return "max"
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if float64(val.Uint()) > max {
			return "max"
		}
	case reflect.Float32, reflect.Float64:
		if val.Float() > max {
			return "max"
		}
	}

	return ""
}

func ruleLen(value any, param string, sv reflect.Value) string {
	if value == nil {
		return ""
	}

	length, err := strconv.Atoi(param)
	if err != nil {
		return ""
	}

	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.String:
		if len([]rune(val.String())) != length {
			return "len"
		}
	case reflect.Slice, reflect.Map, reflect.Array:
		if val.Len() != length {
			return "len"
		}
	}

	return ""
}

func ruleBetween(value any, param string, sv reflect.Value) string {
	if value == nil {
		return ""
	}

	parts := strings.Split(param, "|")
	if len(parts) != 2 {
		return ""
	}

	min, err1 := strconv.ParseFloat(parts[0], 64)
	max, err2 := strconv.ParseFloat(parts[1], 64)
	if err1 != nil || err2 != nil {
		return ""
	}

	val := reflect.ValueOf(value)
	var current float64

	switch val.Kind() {
	case reflect.String:
		current = float64(len([]rune(val.String())))
	case reflect.Slice, reflect.Map, reflect.Array:
		current = float64(val.Len())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		current = float64(val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		current = float64(val.Uint())
	case reflect.Float32, reflect.Float64:
		current = val.Float()
	default:
		return ""
	}

	if current < min || current > max {
		return "between"
	}

	return ""
}

// Comparison validation rules

func ruleEq(value any, param string, sv reflect.Value) string {
	if value == nil {
		return ""
	}
	if toString(value) != param {
		return "eq"
	}
	return ""
}

func ruleNe(value any, param string, sv reflect.Value) string {
	if value == nil {
		return ""
	}
	if toString(value) == param {
		return "ne"
	}
	return ""
}

func ruleGt(value any, param string, sv reflect.Value) string {
	if value == nil {
		return ""
	}

	paramVal, err := strconv.ParseFloat(param, 64)
	if err != nil {
		return ""
	}

	if val, ok := toFloat(value); ok {
		if val <= paramVal {
			return "gt"
		}
	}

	return ""
}

func ruleGte(value any, param string, sv reflect.Value) string {
	if value == nil {
		return ""
	}

	paramVal, err := strconv.ParseFloat(param, 64)
	if err != nil {
		return ""
	}

	if val, ok := toFloat(value); ok {
		if val < paramVal {
			return "gte"
		}
	}

	return ""
}

func ruleLt(value any, param string, sv reflect.Value) string {
	if value == nil {
		return ""
	}

	paramVal, err := strconv.ParseFloat(param, 64)
	if err != nil {
		return ""
	}

	if val, ok := toFloat(value); ok {
		if val >= paramVal {
			return "lt"
		}
	}

	return ""
}

func ruleLte(value any, param string, sv reflect.Value) string {
	if value == nil {
		return ""
	}

	paramVal, err := strconv.ParseFloat(param, 64)
	if err != nil {
		return ""
	}

	if val, ok := toFloat(value); ok {
		if val > paramVal {
			return "lte"
		}
	}

	return ""
}

// Field comparison validation rules

func ruleEqField(value any, param string, sv reflect.Value) string {
	other, ok := getFieldValue(sv, param)
	if !ok {
		return ""
	}
	if !reflect.DeepEqual(value, other) {
		return "eqfield"
	}
	return ""
}

func ruleNeField(value any, param string, sv reflect.Value) string {
	other, ok := getFieldValue(sv, param)
	if !ok {
		return ""
	}
	if reflect.DeepEqual(value, other) {
		return "nefield"
	}
	return ""
}

func ruleGtField(value any, param string, sv reflect.Value) string {
	other, ok := getFieldValue(sv, param)
	if !ok {
		return ""
	}

	val, ok1 := toFloat(value)
	otherVal, ok2 := toFloat(other)
	if !ok1 || !ok2 {
		return ""
	}

	if val <= otherVal {
		return "gtfield"
	}
	return ""
}

func ruleGteField(value any, param string, sv reflect.Value) string {
	other, ok := getFieldValue(sv, param)
	if !ok {
		return ""
	}

	val, ok1 := toFloat(value)
	otherVal, ok2 := toFloat(other)
	if !ok1 || !ok2 {
		return ""
	}

	if val < otherVal {
		return "gtefield"
	}
	return ""
}

func ruleLtField(value any, param string, sv reflect.Value) string {
	other, ok := getFieldValue(sv, param)
	if !ok {
		return ""
	}

	val, ok1 := toFloat(value)
	otherVal, ok2 := toFloat(other)
	if !ok1 || !ok2 {
		return ""
	}

	if val >= otherVal {
		return "ltfield"
	}
	return ""
}

func ruleLteField(value any, param string, sv reflect.Value) string {
	other, ok := getFieldValue(sv, param)
	if !ok {
		return ""
	}

	val, ok1 := toFloat(value)
	otherVal, ok2 := toFloat(other)
	if !ok1 || !ok2 {
		return ""
	}

	if val > otherVal {
		return "ltefield"
	}
	return ""
}

// Network validation rules

func ruleIP(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if net.ParseIP(s) == nil {
		return "ip"
	}
	return ""
}

func ruleIPv4(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	ip := net.ParseIP(s)
	if ip == nil || ip.To4() == nil {
		return "ipv4"
	}
	return ""
}

func ruleIPv6(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	ip := net.ParseIP(s)
	if ip == nil || ip.To4() != nil {
		return "ipv6"
	}
	return ""
}

func ruleCIDR(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	_, _, err := net.ParseCIDR(s)
	if err != nil {
		return "cidr"
	}
	return ""
}

func ruleCIDRv4(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	ip, _, err := net.ParseCIDR(s)
	if err != nil || ip.To4() == nil {
		return "cidrv4"
	}
	return ""
}

func ruleCIDRv6(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	ip, _, err := net.ParseCIDR(s)
	if err != nil || ip.To4() != nil {
		return "cidrv6"
	}
	return ""
}

func ruleMAC(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	_, err := net.ParseMAC(s)
	if err != nil {
		return "mac"
	}
	return ""
}

func ruleHostname(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if len(s) > 253 {
		return "hostname"
	}
	if !hostnameRegex.MatchString(s) {
		return "hostname"
	}
	return ""
}

func ruleFQDN(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !fqdnRegex.MatchString(s) {
		return "fqdn"
	}
	return ""
}

// String validation rules

func ruleContains(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !strings.Contains(s, param) {
		return "contains"
	}
	return ""
}

func ruleContainsAny(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !strings.ContainsAny(s, param) {
		return "containsany"
	}
	return ""
}

func ruleContainsRune(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	runes := []rune(param)
	if len(runes) == 0 {
		return ""
	}
	if !strings.ContainsRune(s, runes[0]) {
		return "containsrune"
	}
	return ""
}

func ruleExcludes(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if strings.Contains(s, param) {
		return "excludes"
	}
	return ""
}

func ruleExcludesAll(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if strings.ContainsAny(s, param) {
		return "excludesall"
	}
	return ""
}

func ruleExcludesRune(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	runes := []rune(param)
	if len(runes) == 0 {
		return ""
	}
	if strings.ContainsRune(s, runes[0]) {
		return "excludesrune"
	}
	return ""
}

func ruleStartsWith(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !strings.HasPrefix(s, param) {
		return "startswith"
	}
	return ""
}

func ruleEndsWith(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !strings.HasSuffix(s, param) {
		return "endswith"
	}
	return ""
}

func ruleStartsNotWith(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, param) {
		return "startsnotwith"
	}
	return ""
}

func ruleEndsNotWith(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if strings.HasSuffix(s, param) {
		return "endsnotwith"
	}
	return ""
}

// Format validation rules

func ruleRegex(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	re, err := regexp.Compile(param)
	if err != nil {
		return ""
	}
	if !re.MatchString(s) {
		return "regex"
	}
	return ""
}

func ruleDatetime(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	_, err := time.Parse(param, s)
	if err != nil {
		return "datetime"
	}
	return ""
}

func ruleDate(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	layouts := []string{"2006-01-02", "01/02/2006", "02-01-2006", "2006/01/02"}
	for _, layout := range layouts {
		if _, err := time.Parse(layout, s); err == nil {
			return ""
		}
	}
	return "date"
}

func ruleTime(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	layouts := []string{"15:04:05", "15:04", "3:04 PM", "3:04:05 PM"}
	for _, layout := range layouts {
		if _, err := time.Parse(layout, s); err == nil {
			return ""
		}
	}
	return "time"
}

func ruleTimezone(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	_, err := time.LoadLocation(s)
	if err != nil {
		return "timezone"
	}
	return ""
}

func ruleDuration(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	_, err := time.ParseDuration(s)
	if err != nil {
		return "duration"
	}
	return ""
}

// Special validation rules

func ruleOneOf(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	options := strings.Split(param, " ")
	for _, opt := range options {
		if s == opt {
			return ""
		}
	}
	return "oneof"
}

func ruleUnique(value any, param string, sv reflect.Value) string {
	if value == nil {
		return ""
	}

	val := reflect.ValueOf(value)
	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		return ""
	}

	seen := make(map[any]bool)
	for i := 0; i < val.Len(); i++ {
		item := val.Index(i).Interface()
		if seen[item] {
			return "unique"
		}
		seen[item] = true
	}

	return ""
}

func ruleDive(value any, param string, sv reflect.Value) string {
	// Dive is handled specially in validateStruct
	return ""
}

// Credit card validation

func ruleCreditCard(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}

	// Remove spaces and dashes
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")

	// Check if all digits
	for _, r := range s {
		if r < '0' || r > '9' {
			return "creditcard"
		}
	}

	// Luhn algorithm
	if !luhnValid(s) {
		return "creditcard"
	}

	return ""
}

func luhnValid(s string) bool {
	n := len(s)
	if n < 13 || n > 19 {
		return false
	}

	var sum int
	alternate := false

	for i := n - 1; i >= 0; i-- {
		digit := int(s[i] - '0')
		if alternate {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		alternate = !alternate
	}

	return sum%10 == 0
}

// Phone validation

func ruleE164(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !e164Regex.MatchString(s) {
		return "e164"
	}
	return ""
}

// Country/language code validation

var countryCodes = map[string]bool{
	"AD": true, "AE": true, "AF": true, "AG": true, "AI": true, "AL": true, "AM": true, "AO": true,
	"AQ": true, "AR": true, "AS": true, "AT": true, "AU": true, "AW": true, "AX": true, "AZ": true,
	"BA": true, "BB": true, "BD": true, "BE": true, "BF": true, "BG": true, "BH": true, "BI": true,
	"BJ": true, "BL": true, "BM": true, "BN": true, "BO": true, "BQ": true, "BR": true, "BS": true,
	"BT": true, "BV": true, "BW": true, "BY": true, "BZ": true, "CA": true, "CC": true, "CD": true,
	"CF": true, "CG": true, "CH": true, "CI": true, "CK": true, "CL": true, "CM": true, "CN": true,
	"CO": true, "CR": true, "CU": true, "CV": true, "CW": true, "CX": true, "CY": true, "CZ": true,
	"DE": true, "DJ": true, "DK": true, "DM": true, "DO": true, "DZ": true, "EC": true, "EE": true,
	"EG": true, "EH": true, "ER": true, "ES": true, "ET": true, "FI": true, "FJ": true, "FK": true,
	"FM": true, "FO": true, "FR": true, "GA": true, "GB": true, "GD": true, "GE": true, "GF": true,
	"GG": true, "GH": true, "GI": true, "GL": true, "GM": true, "GN": true, "GP": true, "GQ": true,
	"GR": true, "GS": true, "GT": true, "GU": true, "GW": true, "GY": true, "HK": true, "HM": true,
	"HN": true, "HR": true, "HT": true, "HU": true, "ID": true, "IE": true, "IL": true, "IM": true,
	"IN": true, "IO": true, "IQ": true, "IR": true, "IS": true, "IT": true, "JE": true, "JM": true,
	"JO": true, "JP": true, "KE": true, "KG": true, "KH": true, "KI": true, "KM": true, "KN": true,
	"KP": true, "KR": true, "KW": true, "KY": true, "KZ": true, "LA": true, "LB": true, "LC": true,
	"LI": true, "LK": true, "LR": true, "LS": true, "LT": true, "LU": true, "LV": true, "LY": true,
	"MA": true, "MC": true, "MD": true, "ME": true, "MF": true, "MG": true, "MH": true, "MK": true,
	"ML": true, "MM": true, "MN": true, "MO": true, "MP": true, "MQ": true, "MR": true, "MS": true,
	"MT": true, "MU": true, "MV": true, "MW": true, "MX": true, "MY": true, "MZ": true, "NA": true,
	"NC": true, "NE": true, "NF": true, "NG": true, "NI": true, "NL": true, "NO": true, "NP": true,
	"NR": true, "NU": true, "NZ": true, "OM": true, "PA": true, "PE": true, "PF": true, "PG": true,
	"PH": true, "PK": true, "PL": true, "PM": true, "PN": true, "PR": true, "PS": true, "PT": true,
	"PW": true, "PY": true, "QA": true, "RE": true, "RO": true, "RS": true, "RU": true, "RW": true,
	"SA": true, "SB": true, "SC": true, "SD": true, "SE": true, "SG": true, "SH": true, "SI": true,
	"SJ": true, "SK": true, "SL": true, "SM": true, "SN": true, "SO": true, "SR": true, "SS": true,
	"ST": true, "SV": true, "SX": true, "SY": true, "SZ": true, "TC": true, "TD": true, "TF": true,
	"TG": true, "TH": true, "TJ": true, "TK": true, "TL": true, "TM": true, "TN": true, "TO": true,
	"TR": true, "TT": true, "TV": true, "TW": true, "TZ": true, "UA": true, "UG": true, "UM": true,
	"US": true, "UY": true, "UZ": true, "VA": true, "VC": true, "VE": true, "VG": true, "VI": true,
	"VN": true, "VU": true, "WF": true, "WS": true, "YE": true, "YT": true, "ZA": true, "ZM": true,
	"ZW": true,
}

var languageCodes = map[string]bool{
	"aa": true, "ab": true, "ae": true, "af": true, "ak": true, "am": true, "an": true, "ar": true,
	"as": true, "av": true, "ay": true, "az": true, "ba": true, "be": true, "bg": true, "bh": true,
	"bi": true, "bm": true, "bn": true, "bo": true, "br": true, "bs": true, "ca": true, "ce": true,
	"ch": true, "co": true, "cr": true, "cs": true, "cu": true, "cv": true, "cy": true, "da": true,
	"de": true, "dv": true, "dz": true, "ee": true, "el": true, "en": true, "eo": true, "es": true,
	"et": true, "eu": true, "fa": true, "ff": true, "fi": true, "fj": true, "fo": true, "fr": true,
	"fy": true, "ga": true, "gd": true, "gl": true, "gn": true, "gu": true, "gv": true, "ha": true,
	"he": true, "hi": true, "ho": true, "hr": true, "ht": true, "hu": true, "hy": true, "hz": true,
	"ia": true, "id": true, "ie": true, "ig": true, "ii": true, "ik": true, "io": true, "is": true,
	"it": true, "iu": true, "ja": true, "jv": true, "ka": true, "kg": true, "ki": true, "kj": true,
	"kk": true, "kl": true, "km": true, "kn": true, "ko": true, "kr": true, "ks": true, "ku": true,
	"kv": true, "kw": true, "ky": true, "la": true, "lb": true, "lg": true, "li": true, "ln": true,
	"lo": true, "lt": true, "lu": true, "lv": true, "mg": true, "mh": true, "mi": true, "mk": true,
	"ml": true, "mn": true, "mr": true, "ms": true, "mt": true, "my": true, "na": true, "nb": true,
	"nd": true, "ne": true, "ng": true, "nl": true, "nn": true, "no": true, "nr": true, "nv": true,
	"ny": true, "oc": true, "oj": true, "om": true, "or": true, "os": true, "pa": true, "pi": true,
	"pl": true, "ps": true, "pt": true, "qu": true, "rm": true, "rn": true, "ro": true, "ru": true,
	"rw": true, "sa": true, "sc": true, "sd": true, "se": true, "sg": true, "si": true, "sk": true,
	"sl": true, "sm": true, "sn": true, "so": true, "sq": true, "sr": true, "ss": true, "st": true,
	"su": true, "sv": true, "sw": true, "ta": true, "te": true, "tg": true, "th": true, "ti": true,
	"tk": true, "tl": true, "tn": true, "to": true, "tr": true, "ts": true, "tt": true, "tw": true,
	"ty": true, "ug": true, "uk": true, "ur": true, "uz": true, "ve": true, "vi": true, "vo": true,
	"wa": true, "wo": true, "xh": true, "yi": true, "yo": true, "za": true, "zh": true, "zu": true,
}

func ruleCountryCode(value any, param string, sv reflect.Value) string {
	s := strings.ToUpper(toString(value))
	if s == "" {
		return ""
	}
	if !countryCodes[s] {
		return "countrycode"
	}
	return ""
}

func ruleLanguageCode(value any, param string, sv reflect.Value) string {
	s := strings.ToLower(toString(value))
	if s == "" {
		return ""
	}
	if !languageCodes[s] {
		return "languagecode"
	}
	return ""
}

func ruleBCP47(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	// Simple BCP47 validation: language[-script][-region]
	parts := strings.Split(s, "-")
	if len(parts) == 0 || len(parts) > 4 {
		return "bcp47"
	}
	// Validate language code
	if !languageCodes[strings.ToLower(parts[0])] {
		return "bcp47"
	}
	return ""
}

// File validation rules

func ruleFilePath(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	// Basic path validation - no null bytes, reasonable length
	if strings.ContainsRune(s, 0) || len(s) > 4096 {
		return "filepath"
	}
	return ""
}

func ruleDirPath(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	// Basic path validation
	if strings.ContainsRune(s, 0) || len(s) > 4096 {
		return "dirpath"
	}
	return ""
}

// Semantic versioning

func ruleSemver(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !semverRegex.MatchString(s) {
		return "semver"
	}
	return ""
}

// Boolean string

func ruleBoolean(value any, param string, sv reflect.Value) string {
	s := strings.ToLower(toString(value))
	if s == "" {
		return ""
	}
	valid := map[string]bool{
		"true": true, "false": true, "1": true, "0": true,
		"yes": true, "no": true, "on": true, "off": true,
	}
	if !valid[s] {
		return "boolean"
	}
	return ""
}

// CVV

func ruleCVV(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	// CVV is 3 or 4 digits
	if len(s) < 3 || len(s) > 4 {
		return "cvv"
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return "cvv"
		}
	}
	return ""
}

// Latitude/Longitude

func ruleLatitude(value any, param string, sv reflect.Value) string {
	if value == nil {
		return ""
	}
	lat, ok := toFloat(value)
	if !ok {
		return "latitude"
	}
	if lat < -90 || lat > 90 {
		return "latitude"
	}
	return ""
}

func ruleLongitude(value any, param string, sv reflect.Value) string {
	if value == nil {
		return ""
	}
	lon, ok := toFloat(value)
	if !ok {
		return "longitude"
	}
	if lon < -180 || lon > 180 {
		return "longitude"
	}
	return ""
}

// Postal code (basic - just checks format, not validity for specific countries)

func rulePostalCode(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	// Basic postal code: alphanumeric with optional spaces/dashes, 3-10 chars
	cleaned := strings.ReplaceAll(strings.ReplaceAll(s, " ", ""), "-", "")
	if len(cleaned) < 3 || len(cleaned) > 10 {
		return "postcode"
	}
	for _, r := range cleaned {
		if !((r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')) {
			return "postcode"
		}
	}
	return ""
}

// Slug

func ruleSlug(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}
	if !slugRegex.MatchString(s) {
		return "slug"
	}
	return ""
}

// Strong password

func ruleStrongPassword(value any, param string, sv reflect.Value) string {
	s := toString(value)
	if s == "" {
		return ""
	}

	if len(s) < 8 {
		return "strongpassword"
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range s {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
		return "strongpassword"
	}

	return ""
}
