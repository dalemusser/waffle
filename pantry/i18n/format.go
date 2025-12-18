// i18n/format.go
package i18n

import (
	"strconv"
	"strings"
	"time"
)

// NumberFormat holds locale-specific number formatting settings.
type NumberFormat struct {
	DecimalSeparator  string
	ThousandSeparator string
	GroupSize         int
}

// CurrencyFormat holds locale-specific currency formatting settings.
type CurrencyFormat struct {
	Symbol          string
	Code            string
	DecimalPlaces   int
	SymbolPosition  string // "before" or "after"
	SymbolSpace     bool   // space between symbol and number
	NumberFormat    NumberFormat
}

// DateFormat holds locale-specific date formatting settings.
type DateFormat struct {
	Short      string // e.g., "01/02/2006"
	Medium     string // e.g., "Jan 2, 2006"
	Long       string // e.g., "January 2, 2006"
	Full       string // e.g., "Monday, January 2, 2006"
	Time       string // e.g., "3:04 PM"
	Time24     string // e.g., "15:04"
	DateTime   string // e.g., "Jan 2, 2006 3:04 PM"
	MonthNames []string
	DayNames   []string
}

// LocaleFormat holds all formatting settings for a locale.
type LocaleFormat struct {
	Tag      string
	Number   NumberFormat
	Currency map[string]CurrencyFormat // keyed by currency code
	Date     DateFormat
	ListAnd  string // "and" in this locale
	ListOr   string // "or" in this locale
	IsRTL    bool
}

// GetLocaleFormat returns formatting settings for a locale.
func GetLocaleFormat(locale string) *LocaleFormat {
	// Normalize locale
	base := locale
	if idx := strings.IndexAny(locale, "-_"); idx > 0 {
		base = locale[:idx]
	}
	base = strings.ToLower(base)

	if fmt, ok := localeFormats[base]; ok {
		return fmt
	}

	// Return English as default
	return localeFormats["en"]
}

// FormatNumber formats a number according to locale conventions.
func (l *Localizer) FormatNumber(n float64) string {
	return l.FormatNumberWithPrecision(n, -1)
}

// FormatNumberWithPrecision formats a number with specific decimal places.
func (l *Localizer) FormatNumberWithPrecision(n float64, precision int) string {
	fmt := GetLocaleFormat(l.locale)
	return formatNumber(n, precision, fmt.Number)
}

// FormatInteger formats an integer according to locale conventions.
func (l *Localizer) FormatInteger(n int64) string {
	fmt := GetLocaleFormat(l.locale)
	return formatInteger(n, fmt.Number)
}

// FormatPercent formats a number as a percentage.
func (l *Localizer) FormatPercent(n float64) string {
	fmt := GetLocaleFormat(l.locale)
	return formatNumber(n*100, 0, fmt.Number) + "%"
}

// FormatPercentWithPrecision formats a percentage with specific decimal places.
func (l *Localizer) FormatPercentWithPrecision(n float64, precision int) string {
	fmt := GetLocaleFormat(l.locale)
	return formatNumber(n*100, precision, fmt.Number) + "%"
}

func formatNumber(n float64, precision int, nf NumberFormat) string {
	// Handle negative
	negative := n < 0
	if negative {
		n = -n
	}

	// Format with precision
	var str string
	if precision < 0 {
		str = strconv.FormatFloat(n, 'f', -1, 64)
	} else {
		str = strconv.FormatFloat(n, 'f', precision, 64)
	}

	// Split into integer and decimal parts
	parts := strings.Split(str, ".")
	intPart := parts[0]
	decPart := ""
	if len(parts) > 1 {
		decPart = parts[1]
	}

	// Add thousand separators
	if nf.ThousandSeparator != "" && len(intPart) > nf.GroupSize {
		var groups []string
		for len(intPart) > 0 {
			end := len(intPart)
			start := end - nf.GroupSize
			if start < 0 {
				start = 0
			}
			groups = append([]string{intPart[start:end]}, groups...)
			intPart = intPart[:start]
		}
		intPart = strings.Join(groups, nf.ThousandSeparator)
	}

	// Combine parts
	result := intPart
	if decPart != "" {
		result += nf.DecimalSeparator + decPart
	}

	if negative {
		result = "-" + result
	}

	return result
}

func formatInteger(n int64, nf NumberFormat) string {
	negative := n < 0
	if negative {
		n = -n
	}

	str := strconv.FormatInt(n, 10)

	// Add thousand separators
	if nf.ThousandSeparator != "" && len(str) > nf.GroupSize {
		var groups []string
		for len(str) > 0 {
			end := len(str)
			start := end - nf.GroupSize
			if start < 0 {
				start = 0
			}
			groups = append([]string{str[start:end]}, groups...)
			str = str[:start]
		}
		str = strings.Join(groups, nf.ThousandSeparator)
	}

	if negative {
		str = "-" + str
	}

	return str
}

// FormatCurrency formats an amount with currency symbol.
func (l *Localizer) FormatCurrency(amount float64, currencyCode string) string {
	locFmt := GetLocaleFormat(l.locale)

	// Get currency format
	curFmt, ok := locFmt.Currency[currencyCode]
	if !ok {
		// Use default format with code
		curFmt = CurrencyFormat{
			Symbol:         currencyCode,
			Code:           currencyCode,
			DecimalPlaces:  2,
			SymbolPosition: "before",
			SymbolSpace:    true,
			NumberFormat:   locFmt.Number,
		}
	}

	// Format the number
	numStr := formatNumber(amount, curFmt.DecimalPlaces, curFmt.NumberFormat)

	// Handle negative
	negative := amount < 0
	if negative {
		numStr = numStr[1:] // Remove the minus sign, we'll add it back
	}

	// Combine with symbol
	var result string
	if curFmt.SymbolPosition == "after" {
		if curFmt.SymbolSpace {
			result = numStr + " " + curFmt.Symbol
		} else {
			result = numStr + curFmt.Symbol
		}
	} else {
		if curFmt.SymbolSpace {
			result = curFmt.Symbol + " " + numStr
		} else {
			result = curFmt.Symbol + numStr
		}
	}

	if negative {
		result = "-" + result
	}

	return result
}

// FormatDate formats a date in short format.
func (l *Localizer) FormatDate(t time.Time) string {
	return l.FormatDateStyle(t, "short")
}

// FormatDateStyle formats a date with a specific style.
// Styles: "short", "medium", "long", "full"
func (l *Localizer) FormatDateStyle(t time.Time, style string) string {
	locFmt := GetLocaleFormat(l.locale)
	df := locFmt.Date

	var format string
	switch style {
	case "medium":
		format = df.Medium
	case "long":
		format = df.Long
	case "full":
		format = df.Full
	default:
		format = df.Short
	}

	return formatDateWithLocale(t, format, df)
}

// FormatTime formats time.
func (l *Localizer) FormatTime(t time.Time) string {
	locFmt := GetLocaleFormat(l.locale)
	return t.Format(locFmt.Date.Time)
}

// FormatTime24 formats time in 24-hour format.
func (l *Localizer) FormatTime24(t time.Time) string {
	locFmt := GetLocaleFormat(l.locale)
	return t.Format(locFmt.Date.Time24)
}

// FormatDateTime formats date and time together.
func (l *Localizer) FormatDateTime(t time.Time) string {
	locFmt := GetLocaleFormat(l.locale)
	return formatDateWithLocale(t, locFmt.Date.DateTime, locFmt.Date)
}

func formatDateWithLocale(t time.Time, format string, df DateFormat) string {
	// Replace month and day names if locale has them
	result := t.Format(format)

	// Replace English month names with localized versions
	if len(df.MonthNames) == 12 {
		englishMonths := []string{
			"January", "February", "March", "April", "May", "June",
			"July", "August", "September", "October", "November", "December",
		}
		englishMonthsShort := []string{
			"Jan", "Feb", "Mar", "Apr", "May", "Jun",
			"Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
		}

		month := int(t.Month()) - 1
		if month >= 0 && month < 12 {
			result = strings.ReplaceAll(result, englishMonths[month], df.MonthNames[month])
			// Handle short month names (first 3 chars)
			if len(df.MonthNames[month]) >= 3 {
				result = strings.ReplaceAll(result, englishMonthsShort[month], df.MonthNames[month][:3])
			}
		}
	}

	// Replace English day names with localized versions
	if len(df.DayNames) == 7 {
		englishDays := []string{
			"Sunday", "Monday", "Tuesday", "Wednesday",
			"Thursday", "Friday", "Saturday",
		}
		englishDaysShort := []string{
			"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat",
		}

		day := int(t.Weekday())
		if day >= 0 && day < 7 {
			result = strings.ReplaceAll(result, englishDays[day], df.DayNames[day])
			if len(df.DayNames[day]) >= 3 {
				result = strings.ReplaceAll(result, englishDaysShort[day], df.DayNames[day][:3])
			}
		}
	}

	return result
}

// FormatRelativeTime formats a duration as relative time (e.g., "2 hours ago").
func (l *Localizer) FormatRelativeTime(d time.Duration) string {
	return l.FormatRelativeTimeFrom(time.Now(), d)
}

// FormatRelativeTimeFrom formats relative time from a reference point.
func (l *Localizer) FormatRelativeTimeFrom(from time.Time, d time.Duration) string {
	// Get relative time translations
	past := d < 0
	if past {
		d = -d
	}

	var key string
	var count int

	seconds := int(d.Seconds())
	minutes := int(d.Minutes())
	hours := int(d.Hours())
	days := hours / 24
	weeks := days / 7
	months := days / 30
	years := days / 365

	switch {
	case seconds < 60:
		if seconds <= 1 {
			key = "relative.now"
			count = 0
		} else {
			key = "relative.seconds"
			count = seconds
		}
	case minutes < 60:
		key = "relative.minutes"
		count = minutes
	case hours < 24:
		key = "relative.hours"
		count = hours
	case days < 7:
		key = "relative.days"
		count = days
	case weeks < 4:
		key = "relative.weeks"
		count = weeks
	case months < 12:
		key = "relative.months"
		count = months
	default:
		key = "relative.years"
		count = years
	}

	// Try to get translated relative time
	if msg, found := l.bundle.getMessage(l.locale, key); found {
		// Handle past/future suffix
		var formatted string
		if strings.Contains(msg, "{{") {
			formatted = l.executeTemplate(msg, []any{map[string]any{"Count": count}})
		} else {
			formatted = l.TPlural(key, count)
		}

		if past {
			if pastMsg, found := l.bundle.getMessage(l.locale, "relative.past"); found {
				return strings.ReplaceAll(pastMsg, "{{.Time}}", formatted)
			}
			return formatted + " ago"
		}

		if futureMsg, found := l.bundle.getMessage(l.locale, "relative.future"); found {
			return strings.ReplaceAll(futureMsg, "{{.Time}}", formatted)
		}
		return "in " + formatted
	}

	// Fallback to English
	return formatRelativeTimeEnglish(count, key, past)
}

func formatRelativeTimeEnglish(count int, key string, past bool) string {
	var unit string
	switch {
	case strings.Contains(key, "now"):
		return "just now"
	case strings.Contains(key, "seconds"):
		unit = "second"
	case strings.Contains(key, "minutes"):
		unit = "minute"
	case strings.Contains(key, "hours"):
		unit = "hour"
	case strings.Contains(key, "days"):
		unit = "day"
	case strings.Contains(key, "weeks"):
		unit = "week"
	case strings.Contains(key, "months"):
		unit = "month"
	case strings.Contains(key, "years"):
		unit = "year"
	}

	if count != 1 {
		unit += "s"
	}

	formatted := strconv.Itoa(count) + " " + unit

	if past {
		return formatted + " ago"
	}
	return "in " + formatted
}

// FormatList formats a list of items with locale-appropriate separators.
func (l *Localizer) FormatList(items []string, conjunction string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}

	locFmt := GetLocaleFormat(l.locale)

	// Determine conjunction word
	var conj string
	switch conjunction {
	case "or":
		conj = locFmt.ListOr
	default:
		conj = locFmt.ListAnd
	}

	if len(items) == 2 {
		return items[0] + " " + conj + " " + items[1]
	}

	// Oxford comma style for English, no Oxford comma for others
	if strings.HasPrefix(l.locale, "en") {
		return strings.Join(items[:len(items)-1], ", ") + ", " + conj + " " + items[len(items)-1]
	}

	return strings.Join(items[:len(items)-1], ", ") + " " + conj + " " + items[len(items)-1]
}

// FormatListAnd formats a list with "and" conjunction.
func (l *Localizer) FormatListAnd(items []string) string {
	return l.FormatList(items, "and")
}

// FormatListOr formats a list with "or" conjunction.
func (l *Localizer) FormatListOr(items []string) string {
	return l.FormatList(items, "or")
}

// IsRTL returns true if the locale uses right-to-left text direction.
func (l *Localizer) IsRTL() bool {
	locFmt := GetLocaleFormat(l.locale)
	return locFmt.IsRTL
}

// TextDirection returns "rtl" or "ltr" for the locale.
func (l *Localizer) TextDirection() string {
	if l.IsRTL() {
		return "rtl"
	}
	return "ltr"
}

// localeFormats contains formatting data for various locales.
var localeFormats = map[string]*LocaleFormat{
	"en": {
		Tag: "en",
		Number: NumberFormat{
			DecimalSeparator:  ".",
			ThousandSeparator: ",",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"USD": {Symbol: "$", Code: "USD", DecimalPlaces: 2, SymbolPosition: "before", NumberFormat: NumberFormat{DecimalSeparator: ".", ThousandSeparator: ",", GroupSize: 3}},
			"EUR": {Symbol: "€", Code: "EUR", DecimalPlaces: 2, SymbolPosition: "before", NumberFormat: NumberFormat{DecimalSeparator: ".", ThousandSeparator: ",", GroupSize: 3}},
			"GBP": {Symbol: "£", Code: "GBP", DecimalPlaces: 2, SymbolPosition: "before", NumberFormat: NumberFormat{DecimalSeparator: ".", ThousandSeparator: ",", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "01/02/2006",
			Medium:   "Jan 2, 2006",
			Long:     "January 2, 2006",
			Full:     "Monday, January 2, 2006",
			Time:     "3:04 PM",
			Time24:   "15:04",
			DateTime: "Jan 2, 2006 3:04 PM",
		},
		ListAnd: "and",
		ListOr:  "or",
		IsRTL:   false,
	},
	"de": {
		Tag: "de",
		Number: NumberFormat{
			DecimalSeparator:  ",",
			ThousandSeparator: ".",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"EUR": {Symbol: "€", Code: "EUR", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: ".", GroupSize: 3}},
			"USD": {Symbol: "$", Code: "USD", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: ".", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "02.01.2006",
			Medium:   "2. Jan. 2006",
			Long:     "2. January 2006",
			Full:     "Monday, 2. January 2006",
			Time:     "15:04",
			Time24:   "15:04",
			DateTime: "2. Jan. 2006 15:04",
			MonthNames: []string{
				"Januar", "Februar", "März", "April", "Mai", "Juni",
				"Juli", "August", "September", "Oktober", "November", "Dezember",
			},
			DayNames: []string{
				"Sonntag", "Montag", "Dienstag", "Mittwoch",
				"Donnerstag", "Freitag", "Samstag",
			},
		},
		ListAnd: "und",
		ListOr:  "oder",
		IsRTL:   false,
	},
	"fr": {
		Tag: "fr",
		Number: NumberFormat{
			DecimalSeparator:  ",",
			ThousandSeparator: " ",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"EUR": {Symbol: "€", Code: "EUR", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: " ", GroupSize: 3}},
			"USD": {Symbol: "$", Code: "USD", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: " ", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "02/01/2006",
			Medium:   "2 janv. 2006",
			Long:     "2 January 2006",
			Full:     "Monday 2 January 2006",
			Time:     "15:04",
			Time24:   "15:04",
			DateTime: "2 janv. 2006 15:04",
			MonthNames: []string{
				"janvier", "février", "mars", "avril", "mai", "juin",
				"juillet", "août", "septembre", "octobre", "novembre", "décembre",
			},
			DayNames: []string{
				"dimanche", "lundi", "mardi", "mercredi",
				"jeudi", "vendredi", "samedi",
			},
		},
		ListAnd: "et",
		ListOr:  "ou",
		IsRTL:   false,
	},
	"es": {
		Tag: "es",
		Number: NumberFormat{
			DecimalSeparator:  ",",
			ThousandSeparator: ".",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"EUR": {Symbol: "€", Code: "EUR", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: ".", GroupSize: 3}},
			"USD": {Symbol: "$", Code: "USD", DecimalPlaces: 2, SymbolPosition: "before", NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: ".", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "02/01/2006",
			Medium:   "2 ene. 2006",
			Long:     "2 de January de 2006",
			Full:     "Monday, 2 de January de 2006",
			Time:     "15:04",
			Time24:   "15:04",
			DateTime: "2 ene. 2006 15:04",
			MonthNames: []string{
				"enero", "febrero", "marzo", "abril", "mayo", "junio",
				"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre",
			},
			DayNames: []string{
				"domingo", "lunes", "martes", "miércoles",
				"jueves", "viernes", "sábado",
			},
		},
		ListAnd: "y",
		ListOr:  "o",
		IsRTL:   false,
	},
	"it": {
		Tag: "it",
		Number: NumberFormat{
			DecimalSeparator:  ",",
			ThousandSeparator: ".",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"EUR": {Symbol: "€", Code: "EUR", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: ".", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "02/01/2006",
			Medium:   "2 gen 2006",
			Long:     "2 January 2006",
			Full:     "Monday 2 January 2006",
			Time:     "15:04",
			Time24:   "15:04",
			DateTime: "2 gen 2006 15:04",
			MonthNames: []string{
				"gennaio", "febbraio", "marzo", "aprile", "maggio", "giugno",
				"luglio", "agosto", "settembre", "ottobre", "novembre", "dicembre",
			},
			DayNames: []string{
				"domenica", "lunedì", "martedì", "mercoledì",
				"giovedì", "venerdì", "sabato",
			},
		},
		ListAnd: "e",
		ListOr:  "o",
		IsRTL:   false,
	},
	"pt": {
		Tag: "pt",
		Number: NumberFormat{
			DecimalSeparator:  ",",
			ThousandSeparator: ".",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"BRL": {Symbol: "R$", Code: "BRL", DecimalPlaces: 2, SymbolPosition: "before", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: ".", GroupSize: 3}},
			"EUR": {Symbol: "€", Code: "EUR", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: ".", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "02/01/2006",
			Medium:   "2 de jan. de 2006",
			Long:     "2 de January de 2006",
			Full:     "Monday, 2 de January de 2006",
			Time:     "15:04",
			Time24:   "15:04",
			DateTime: "2 de jan. de 2006 15:04",
			MonthNames: []string{
				"janeiro", "fevereiro", "março", "abril", "maio", "junho",
				"julho", "agosto", "setembro", "outubro", "novembro", "dezembro",
			},
			DayNames: []string{
				"domingo", "segunda-feira", "terça-feira", "quarta-feira",
				"quinta-feira", "sexta-feira", "sábado",
			},
		},
		ListAnd: "e",
		ListOr:  "ou",
		IsRTL:   false,
	},
	"ja": {
		Tag: "ja",
		Number: NumberFormat{
			DecimalSeparator:  ".",
			ThousandSeparator: ",",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"JPY": {Symbol: "¥", Code: "JPY", DecimalPlaces: 0, SymbolPosition: "before", NumberFormat: NumberFormat{DecimalSeparator: ".", ThousandSeparator: ",", GroupSize: 3}},
			"USD": {Symbol: "$", Code: "USD", DecimalPlaces: 2, SymbolPosition: "before", NumberFormat: NumberFormat{DecimalSeparator: ".", ThousandSeparator: ",", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "2006/01/02",
			Medium:   "2006年1月2日",
			Long:     "2006年1月2日",
			Full:     "2006年1月2日 Monday",
			Time:     "15:04",
			Time24:   "15:04",
			DateTime: "2006年1月2日 15:04",
			MonthNames: []string{
				"1月", "2月", "3月", "4月", "5月", "6月",
				"7月", "8月", "9月", "10月", "11月", "12月",
			},
			DayNames: []string{
				"日曜日", "月曜日", "火曜日", "水曜日",
				"木曜日", "金曜日", "土曜日",
			},
		},
		ListAnd: "と",
		ListOr:  "または",
		IsRTL:   false,
	},
	"zh": {
		Tag: "zh",
		Number: NumberFormat{
			DecimalSeparator:  ".",
			ThousandSeparator: ",",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"CNY": {Symbol: "¥", Code: "CNY", DecimalPlaces: 2, SymbolPosition: "before", NumberFormat: NumberFormat{DecimalSeparator: ".", ThousandSeparator: ",", GroupSize: 3}},
			"USD": {Symbol: "$", Code: "USD", DecimalPlaces: 2, SymbolPosition: "before", NumberFormat: NumberFormat{DecimalSeparator: ".", ThousandSeparator: ",", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "2006/01/02",
			Medium:   "2006年1月2日",
			Long:     "2006年1月2日",
			Full:     "2006年1月2日 Monday",
			Time:     "下午3:04",
			Time24:   "15:04",
			DateTime: "2006年1月2日 15:04",
			MonthNames: []string{
				"一月", "二月", "三月", "四月", "五月", "六月",
				"七月", "八月", "九月", "十月", "十一月", "十二月",
			},
			DayNames: []string{
				"星期日", "星期一", "星期二", "星期三",
				"星期四", "星期五", "星期六",
			},
		},
		ListAnd: "和",
		ListOr:  "或",
		IsRTL:   false,
	},
	"ko": {
		Tag: "ko",
		Number: NumberFormat{
			DecimalSeparator:  ".",
			ThousandSeparator: ",",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"KRW": {Symbol: "₩", Code: "KRW", DecimalPlaces: 0, SymbolPosition: "before", NumberFormat: NumberFormat{DecimalSeparator: ".", ThousandSeparator: ",", GroupSize: 3}},
			"USD": {Symbol: "$", Code: "USD", DecimalPlaces: 2, SymbolPosition: "before", NumberFormat: NumberFormat{DecimalSeparator: ".", ThousandSeparator: ",", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "2006. 01. 02.",
			Medium:   "2006년 1월 2일",
			Long:     "2006년 1월 2일",
			Full:     "2006년 1월 2일 Monday",
			Time:     "오후 3:04",
			Time24:   "15:04",
			DateTime: "2006년 1월 2일 15:04",
			MonthNames: []string{
				"1월", "2월", "3월", "4월", "5월", "6월",
				"7월", "8월", "9월", "10월", "11월", "12월",
			},
			DayNames: []string{
				"일요일", "월요일", "화요일", "수요일",
				"목요일", "금요일", "토요일",
			},
		},
		ListAnd: "및",
		ListOr:  "또는",
		IsRTL:   false,
	},
	"ru": {
		Tag: "ru",
		Number: NumberFormat{
			DecimalSeparator:  ",",
			ThousandSeparator: " ",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"RUB": {Symbol: "₽", Code: "RUB", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: " ", GroupSize: 3}},
			"USD": {Symbol: "$", Code: "USD", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: " ", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "02.01.2006",
			Medium:   "2 янв. 2006 г.",
			Long:     "2 January 2006 г.",
			Full:     "Monday, 2 January 2006 г.",
			Time:     "15:04",
			Time24:   "15:04",
			DateTime: "2 янв. 2006 г., 15:04",
			MonthNames: []string{
				"января", "февраля", "марта", "апреля", "мая", "июня",
				"июля", "августа", "сентября", "октября", "ноября", "декабря",
			},
			DayNames: []string{
				"воскресенье", "понедельник", "вторник", "среда",
				"четверг", "пятница", "суббота",
			},
		},
		ListAnd: "и",
		ListOr:  "или",
		IsRTL:   false,
	},
	"ar": {
		Tag: "ar",
		Number: NumberFormat{
			DecimalSeparator:  "٫",
			ThousandSeparator: "٬",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"SAR": {Symbol: "ر.س", Code: "SAR", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: "٫", ThousandSeparator: "٬", GroupSize: 3}},
			"USD": {Symbol: "$", Code: "USD", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: "٫", ThousandSeparator: "٬", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "02/01/2006",
			Medium:   "2 يناير 2006",
			Long:     "2 January 2006",
			Full:     "Monday، 2 January 2006",
			Time:     "3:04 م",
			Time24:   "15:04",
			DateTime: "2 يناير 2006، 15:04",
			MonthNames: []string{
				"يناير", "فبراير", "مارس", "أبريل", "مايو", "يونيو",
				"يوليو", "أغسطس", "سبتمبر", "أكتوبر", "نوفمبر", "ديسمبر",
			},
			DayNames: []string{
				"الأحد", "الاثنين", "الثلاثاء", "الأربعاء",
				"الخميس", "الجمعة", "السبت",
			},
		},
		ListAnd: "و",
		ListOr:  "أو",
		IsRTL:   true,
	},
	"he": {
		Tag: "he",
		Number: NumberFormat{
			DecimalSeparator:  ".",
			ThousandSeparator: ",",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"ILS": {Symbol: "₪", Code: "ILS", DecimalPlaces: 2, SymbolPosition: "before", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ".", ThousandSeparator: ",", GroupSize: 3}},
			"USD": {Symbol: "$", Code: "USD", DecimalPlaces: 2, SymbolPosition: "before", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ".", ThousandSeparator: ",", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "02.01.2006",
			Medium:   "2 בינו׳ 2006",
			Long:     "2 בJanuary 2006",
			Full:     "יום Monday, 2 בJanuary 2006",
			Time:     "15:04",
			Time24:   "15:04",
			DateTime: "2 בינו׳ 2006, 15:04",
			MonthNames: []string{
				"ינואר", "פברואר", "מרץ", "אפריל", "מאי", "יוני",
				"יולי", "אוגוסט", "ספטמבר", "אוקטובר", "נובמבר", "דצמבר",
			},
			DayNames: []string{
				"יום ראשון", "יום שני", "יום שלישי", "יום רביעי",
				"יום חמישי", "יום שישי", "יום שבת",
			},
		},
		ListAnd: "ו",
		ListOr:  "או",
		IsRTL:   true,
	},
	"nl": {
		Tag: "nl",
		Number: NumberFormat{
			DecimalSeparator:  ",",
			ThousandSeparator: ".",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"EUR": {Symbol: "€", Code: "EUR", DecimalPlaces: 2, SymbolPosition: "before", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: ".", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "02-01-2006",
			Medium:   "2 jan. 2006",
			Long:     "2 January 2006",
			Full:     "Monday 2 January 2006",
			Time:     "15:04",
			Time24:   "15:04",
			DateTime: "2 jan. 2006 15:04",
			MonthNames: []string{
				"januari", "februari", "maart", "april", "mei", "juni",
				"juli", "augustus", "september", "oktober", "november", "december",
			},
			DayNames: []string{
				"zondag", "maandag", "dinsdag", "woensdag",
				"donderdag", "vrijdag", "zaterdag",
			},
		},
		ListAnd: "en",
		ListOr:  "of",
		IsRTL:   false,
	},
	"pl": {
		Tag: "pl",
		Number: NumberFormat{
			DecimalSeparator:  ",",
			ThousandSeparator: " ",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"PLN": {Symbol: "zł", Code: "PLN", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: " ", GroupSize: 3}},
			"EUR": {Symbol: "€", Code: "EUR", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: " ", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "02.01.2006",
			Medium:   "2 sty 2006",
			Long:     "2 January 2006",
			Full:     "Monday, 2 January 2006",
			Time:     "15:04",
			Time24:   "15:04",
			DateTime: "2 sty 2006, 15:04",
			MonthNames: []string{
				"stycznia", "lutego", "marca", "kwietnia", "maja", "czerwca",
				"lipca", "sierpnia", "września", "października", "listopada", "grudnia",
			},
			DayNames: []string{
				"niedziela", "poniedziałek", "wtorek", "środa",
				"czwartek", "piątek", "sobota",
			},
		},
		ListAnd: "i",
		ListOr:  "lub",
		IsRTL:   false,
	},
	"tr": {
		Tag: "tr",
		Number: NumberFormat{
			DecimalSeparator:  ",",
			ThousandSeparator: ".",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"TRY": {Symbol: "₺", Code: "TRY", DecimalPlaces: 2, SymbolPosition: "before", NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: ".", GroupSize: 3}},
			"USD": {Symbol: "$", Code: "USD", DecimalPlaces: 2, SymbolPosition: "before", NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: ".", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "02.01.2006",
			Medium:   "2 Oca 2006",
			Long:     "2 January 2006",
			Full:     "2 January 2006 Monday",
			Time:     "15:04",
			Time24:   "15:04",
			DateTime: "2 Oca 2006 15:04",
			MonthNames: []string{
				"Ocak", "Şubat", "Mart", "Nisan", "Mayıs", "Haziran",
				"Temmuz", "Ağustos", "Eylül", "Ekim", "Kasım", "Aralık",
			},
			DayNames: []string{
				"Pazar", "Pazartesi", "Salı", "Çarşamba",
				"Perşembe", "Cuma", "Cumartesi",
			},
		},
		ListAnd: "ve",
		ListOr:  "veya",
		IsRTL:   false,
	},
	"sv": {
		Tag: "sv",
		Number: NumberFormat{
			DecimalSeparator:  ",",
			ThousandSeparator: " ",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"SEK": {Symbol: "kr", Code: "SEK", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: " ", GroupSize: 3}},
			"EUR": {Symbol: "€", Code: "EUR", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: " ", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "2006-01-02",
			Medium:   "2 jan. 2006",
			Long:     "2 January 2006",
			Full:     "Monday 2 January 2006",
			Time:     "15:04",
			Time24:   "15:04",
			DateTime: "2 jan. 2006 15:04",
			MonthNames: []string{
				"januari", "februari", "mars", "april", "maj", "juni",
				"juli", "augusti", "september", "oktober", "november", "december",
			},
			DayNames: []string{
				"söndag", "måndag", "tisdag", "onsdag",
				"torsdag", "fredag", "lördag",
			},
		},
		ListAnd: "och",
		ListOr:  "eller",
		IsRTL:   false,
	},
	"da": {
		Tag: "da",
		Number: NumberFormat{
			DecimalSeparator:  ",",
			ThousandSeparator: ".",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"DKK": {Symbol: "kr.", Code: "DKK", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: ".", GroupSize: 3}},
			"EUR": {Symbol: "€", Code: "EUR", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: ".", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "02.01.2006",
			Medium:   "2. jan. 2006",
			Long:     "2. January 2006",
			Full:     "Monday den 2. January 2006",
			Time:     "15.04",
			Time24:   "15.04",
			DateTime: "2. jan. 2006 15.04",
			MonthNames: []string{
				"januar", "februar", "marts", "april", "maj", "juni",
				"juli", "august", "september", "oktober", "november", "december",
			},
			DayNames: []string{
				"søndag", "mandag", "tirsdag", "onsdag",
				"torsdag", "fredag", "lørdag",
			},
		},
		ListAnd: "og",
		ListOr:  "eller",
		IsRTL:   false,
	},
	"no": {
		Tag: "no",
		Number: NumberFormat{
			DecimalSeparator:  ",",
			ThousandSeparator: " ",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"NOK": {Symbol: "kr", Code: "NOK", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: " ", GroupSize: 3}},
			"EUR": {Symbol: "€", Code: "EUR", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: " ", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "02.01.2006",
			Medium:   "2. jan. 2006",
			Long:     "2. January 2006",
			Full:     "Monday 2. January 2006",
			Time:     "15:04",
			Time24:   "15:04",
			DateTime: "2. jan. 2006, 15:04",
			MonthNames: []string{
				"januar", "februar", "mars", "april", "mai", "juni",
				"juli", "august", "september", "oktober", "november", "desember",
			},
			DayNames: []string{
				"søndag", "mandag", "tirsdag", "onsdag",
				"torsdag", "fredag", "lørdag",
			},
		},
		ListAnd: "og",
		ListOr:  "eller",
		IsRTL:   false,
	},
	"fi": {
		Tag: "fi",
		Number: NumberFormat{
			DecimalSeparator:  ",",
			ThousandSeparator: " ",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"EUR": {Symbol: "€", Code: "EUR", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: " ", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "2.1.2006",
			Medium:   "2. tammik. 2006",
			Long:     "2. Januaryta 2006",
			Full:     "Monday 2. Januaryta 2006",
			Time:     "15.04",
			Time24:   "15.04",
			DateTime: "2. tammik. 2006 klo 15.04",
			MonthNames: []string{
				"tammikuuta", "helmikuuta", "maaliskuuta", "huhtikuuta", "toukokuuta", "kesäkuuta",
				"heinäkuuta", "elokuuta", "syyskuuta", "lokakuuta", "marraskuuta", "joulukuuta",
			},
			DayNames: []string{
				"sunnuntaina", "maanantaina", "tiistaina", "keskiviikkona",
				"torstaina", "perjantaina", "lauantaina",
			},
		},
		ListAnd: "ja",
		ListOr:  "tai",
		IsRTL:   false,
	},
	"th": {
		Tag: "th",
		Number: NumberFormat{
			DecimalSeparator:  ".",
			ThousandSeparator: ",",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"THB": {Symbol: "฿", Code: "THB", DecimalPlaces: 2, SymbolPosition: "before", NumberFormat: NumberFormat{DecimalSeparator: ".", ThousandSeparator: ",", GroupSize: 3}},
			"USD": {Symbol: "$", Code: "USD", DecimalPlaces: 2, SymbolPosition: "before", NumberFormat: NumberFormat{DecimalSeparator: ".", ThousandSeparator: ",", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "2/1/2006",
			Medium:   "2 ม.ค. 2006",
			Long:     "2 January 2006",
			Full:     "วันMonday ที่ 2 January 2006",
			Time:     "15:04",
			Time24:   "15:04",
			DateTime: "2 ม.ค. 2006 15:04",
			MonthNames: []string{
				"มกราคม", "กุมภาพันธ์", "มีนาคม", "เมษายน", "พฤษภาคม", "มิถุนายน",
				"กรกฎาคม", "สิงหาคม", "กันยายน", "ตุลาคม", "พฤศจิกายน", "ธันวาคม",
			},
			DayNames: []string{
				"วันอาทิตย์", "วันจันทร์", "วันอังคาร", "วันพุธ",
				"วันพฤหัสบดี", "วันศุกร์", "วันเสาร์",
			},
		},
		ListAnd: "และ",
		ListOr:  "หรือ",
		IsRTL:   false,
	},
	"vi": {
		Tag: "vi",
		Number: NumberFormat{
			DecimalSeparator:  ",",
			ThousandSeparator: ".",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"VND": {Symbol: "₫", Code: "VND", DecimalPlaces: 0, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: ".", GroupSize: 3}},
			"USD": {Symbol: "$", Code: "USD", DecimalPlaces: 2, SymbolPosition: "before", NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: ".", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "02/01/2006",
			Medium:   "2 thg 1, 2006",
			Long:     "Ngày 2 tháng 1 năm 2006",
			Full:     "Monday, 2 tháng 1, 2006",
			Time:     "15:04",
			Time24:   "15:04",
			DateTime: "2 thg 1, 2006, 15:04",
			MonthNames: []string{
				"tháng 1", "tháng 2", "tháng 3", "tháng 4", "tháng 5", "tháng 6",
				"tháng 7", "tháng 8", "tháng 9", "tháng 10", "tháng 11", "tháng 12",
			},
			DayNames: []string{
				"Chủ Nhật", "Thứ Hai", "Thứ Ba", "Thứ Tư",
				"Thứ Năm", "Thứ Sáu", "Thứ Bảy",
			},
		},
		ListAnd: "và",
		ListOr:  "hoặc",
		IsRTL:   false,
	},
	"id": {
		Tag: "id",
		Number: NumberFormat{
			DecimalSeparator:  ",",
			ThousandSeparator: ".",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"IDR": {Symbol: "Rp", Code: "IDR", DecimalPlaces: 0, SymbolPosition: "before", NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: ".", GroupSize: 3}},
			"USD": {Symbol: "$", Code: "USD", DecimalPlaces: 2, SymbolPosition: "before", NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: ".", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "02/01/2006",
			Medium:   "2 Jan 2006",
			Long:     "2 January 2006",
			Full:     "Monday, 2 January 2006",
			Time:     "15.04",
			Time24:   "15.04",
			DateTime: "2 Jan 2006 15.04",
			MonthNames: []string{
				"Januari", "Februari", "Maret", "April", "Mei", "Juni",
				"Juli", "Agustus", "September", "Oktober", "November", "Desember",
			},
			DayNames: []string{
				"Minggu", "Senin", "Selasa", "Rabu",
				"Kamis", "Jumat", "Sabtu",
			},
		},
		ListAnd: "dan",
		ListOr:  "atau",
		IsRTL:   false,
	},
	"ms": {
		Tag: "ms",
		Number: NumberFormat{
			DecimalSeparator:  ".",
			ThousandSeparator: ",",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"MYR": {Symbol: "RM", Code: "MYR", DecimalPlaces: 2, SymbolPosition: "before", NumberFormat: NumberFormat{DecimalSeparator: ".", ThousandSeparator: ",", GroupSize: 3}},
			"USD": {Symbol: "$", Code: "USD", DecimalPlaces: 2, SymbolPosition: "before", NumberFormat: NumberFormat{DecimalSeparator: ".", ThousandSeparator: ",", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "02/01/2006",
			Medium:   "2 Jan 2006",
			Long:     "2 January 2006",
			Full:     "Monday, 2 January 2006",
			Time:     "3:04 PM",
			Time24:   "15:04",
			DateTime: "2 Jan 2006, 3:04 PM",
			MonthNames: []string{
				"Januari", "Februari", "Mac", "April", "Mei", "Jun",
				"Julai", "Ogos", "September", "Oktober", "November", "Disember",
			},
			DayNames: []string{
				"Ahad", "Isnin", "Selasa", "Rabu",
				"Khamis", "Jumaat", "Sabtu",
			},
		},
		ListAnd: "dan",
		ListOr:  "atau",
		IsRTL:   false,
	},
	"hi": {
		Tag: "hi",
		Number: NumberFormat{
			DecimalSeparator:  ".",
			ThousandSeparator: ",",
			GroupSize:         3, // Note: Indian numbering uses 2,2,3 pattern but we simplify
		},
		Currency: map[string]CurrencyFormat{
			"INR": {Symbol: "₹", Code: "INR", DecimalPlaces: 2, SymbolPosition: "before", NumberFormat: NumberFormat{DecimalSeparator: ".", ThousandSeparator: ",", GroupSize: 3}},
			"USD": {Symbol: "$", Code: "USD", DecimalPlaces: 2, SymbolPosition: "before", NumberFormat: NumberFormat{DecimalSeparator: ".", ThousandSeparator: ",", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "02/01/2006",
			Medium:   "2 जन॰ 2006",
			Long:     "2 January 2006",
			Full:     "Monday, 2 January 2006",
			Time:     "3:04 pm",
			Time24:   "15:04",
			DateTime: "2 जन॰ 2006, 3:04 pm",
			MonthNames: []string{
				"जनवरी", "फ़रवरी", "मार्च", "अप्रैल", "मई", "जून",
				"जुलाई", "अगस्त", "सितंबर", "अक्तूबर", "नवंबर", "दिसंबर",
			},
			DayNames: []string{
				"रविवार", "सोमवार", "मंगलवार", "बुधवार",
				"गुरुवार", "शुक्रवार", "शनिवार",
			},
		},
		ListAnd: "और",
		ListOr:  "या",
		IsRTL:   false,
	},
	"uk": {
		Tag: "uk",
		Number: NumberFormat{
			DecimalSeparator:  ",",
			ThousandSeparator: " ",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"UAH": {Symbol: "₴", Code: "UAH", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: " ", GroupSize: 3}},
			"USD": {Symbol: "$", Code: "USD", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: " ", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "02.01.2006",
			Medium:   "2 січ. 2006 р.",
			Long:     "2 January 2006 р.",
			Full:     "Monday, 2 January 2006 р.",
			Time:     "15:04",
			Time24:   "15:04",
			DateTime: "2 січ. 2006 р., 15:04",
			MonthNames: []string{
				"січня", "лютого", "березня", "квітня", "травня", "червня",
				"липня", "серпня", "вересня", "жовтня", "листопада", "грудня",
			},
			DayNames: []string{
				"неділя", "понеділок", "вівторок", "середа",
				"четвер", "пʼятниця", "субота",
			},
		},
		ListAnd: "та",
		ListOr:  "або",
		IsRTL:   false,
	},
	"cs": {
		Tag: "cs",
		Number: NumberFormat{
			DecimalSeparator:  ",",
			ThousandSeparator: " ",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"CZK": {Symbol: "Kč", Code: "CZK", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: " ", GroupSize: 3}},
			"EUR": {Symbol: "€", Code: "EUR", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: " ", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "02.01.2006",
			Medium:   "2. 1. 2006",
			Long:     "2. January 2006",
			Full:     "Monday 2. January 2006",
			Time:     "15:04",
			Time24:   "15:04",
			DateTime: "2. 1. 2006 15:04",
			MonthNames: []string{
				"ledna", "února", "března", "dubna", "května", "června",
				"července", "srpna", "září", "října", "listopadu", "prosince",
			},
			DayNames: []string{
				"neděle", "pondělí", "úterý", "středa",
				"čtvrtek", "pátek", "sobota",
			},
		},
		ListAnd: "a",
		ListOr:  "nebo",
		IsRTL:   false,
	},
	"el": {
		Tag: "el",
		Number: NumberFormat{
			DecimalSeparator:  ",",
			ThousandSeparator: ".",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"EUR": {Symbol: "€", Code: "EUR", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: ".", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "02/01/2006",
			Medium:   "2 Ιαν 2006",
			Long:     "2 January 2006",
			Full:     "Monday 2 January 2006",
			Time:     "3:04 μ.μ.",
			Time24:   "15:04",
			DateTime: "2 Ιαν 2006, 3:04 μ.μ.",
			MonthNames: []string{
				"Ιανουαρίου", "Φεβρουαρίου", "Μαρτίου", "Απριλίου", "Μαΐου", "Ιουνίου",
				"Ιουλίου", "Αυγούστου", "Σεπτεμβρίου", "Οκτωβρίου", "Νοεμβρίου", "Δεκεμβρίου",
			},
			DayNames: []string{
				"Κυριακή", "Δευτέρα", "Τρίτη", "Τετάρτη",
				"Πέμπτη", "Παρασκευή", "Σάββατο",
			},
		},
		ListAnd: "και",
		ListOr:  "ή",
		IsRTL:   false,
	},
	"ro": {
		Tag: "ro",
		Number: NumberFormat{
			DecimalSeparator:  ",",
			ThousandSeparator: ".",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"RON": {Symbol: "lei", Code: "RON", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: ".", GroupSize: 3}},
			"EUR": {Symbol: "€", Code: "EUR", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: ".", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "02.01.2006",
			Medium:   "2 ian. 2006",
			Long:     "2 January 2006",
			Full:     "Monday, 2 January 2006",
			Time:     "15:04",
			Time24:   "15:04",
			DateTime: "2 ian. 2006, 15:04",
			MonthNames: []string{
				"ianuarie", "februarie", "martie", "aprilie", "mai", "iunie",
				"iulie", "august", "septembrie", "octombrie", "noiembrie", "decembrie",
			},
			DayNames: []string{
				"duminică", "luni", "marți", "miercuri",
				"joi", "vineri", "sâmbătă",
			},
		},
		ListAnd: "și",
		ListOr:  "sau",
		IsRTL:   false,
	},
	"hu": {
		Tag: "hu",
		Number: NumberFormat{
			DecimalSeparator:  ",",
			ThousandSeparator: " ",
			GroupSize:         3,
		},
		Currency: map[string]CurrencyFormat{
			"HUF": {Symbol: "Ft", Code: "HUF", DecimalPlaces: 0, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: " ", GroupSize: 3}},
			"EUR": {Symbol: "€", Code: "EUR", DecimalPlaces: 2, SymbolPosition: "after", SymbolSpace: true, NumberFormat: NumberFormat{DecimalSeparator: ",", ThousandSeparator: " ", GroupSize: 3}},
		},
		Date: DateFormat{
			Short:    "2006. 01. 02.",
			Medium:   "2006. jan. 2.",
			Long:     "2006. January 2.",
			Full:     "2006. January 2., Monday",
			Time:     "15:04",
			Time24:   "15:04",
			DateTime: "2006. jan. 2. 15:04",
			MonthNames: []string{
				"január", "február", "március", "április", "május", "június",
				"július", "augusztus", "szeptember", "október", "november", "december",
			},
			DayNames: []string{
				"vasárnap", "hétfő", "kedd", "szerda",
				"csütörtök", "péntek", "szombat",
			},
		},
		ListAnd: "és",
		ListOr:  "vagy",
		IsRTL:   false,
	},
}
