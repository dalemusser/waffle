// i18n/plural.go
package i18n

import "strings"

// PluralForm represents a plural category.
type PluralForm string

const (
	PluralZero  PluralForm = "zero"
	PluralOne   PluralForm = "one"
	PluralTwo   PluralForm = "two"
	PluralFew   PluralForm = "few"
	PluralMany  PluralForm = "many"
	PluralOther PluralForm = "other"
)

// PluralFunc determines the plural form for a count.
type PluralFunc func(n int) PluralForm

// GetPluralFunc returns the plural function for a locale.
func GetPluralFunc(locale string) PluralFunc {
	// Normalize locale
	locale = strings.ToLower(locale)
	if idx := strings.IndexAny(locale, "-_"); idx > 0 {
		locale = locale[:idx]
	}

	switch locale {
	// Germanic family (1 vs other)
	case "en", "de", "nl", "sv", "da", "no", "nb", "nn", "is", "fo":
		return PluralEnglish

	// Romance family (0-1 vs other)
	case "fr", "pt":
		return PluralFrench

	// Spanish, Italian (1 vs other)
	case "es", "it", "ca", "eu":
		return PluralEnglish

	// Slavic family - Russian, Ukrainian, etc (complex rules)
	case "ru", "uk", "be", "hr", "sr", "bs":
		return PluralRussian

	// Polish (complex rules)
	case "pl":
		return PluralPolish

	// Czech, Slovak
	case "cs", "sk":
		return PluralCzech

	// Arabic (complex - 6 forms)
	case "ar":
		return PluralArabic

	// Japanese, Chinese, Korean, Vietnamese (no plural)
	case "ja", "zh", "ko", "vi", "th", "id", "ms":
		return PluralAsian

	// Turkish (1 vs other)
	case "tr":
		return PluralEnglish

	// Romanian
	case "ro":
		return PluralRomanian

	// Lithuanian
	case "lt":
		return PluralLithuanian

	// Latvian
	case "lv":
		return PluralLatvian

	// Irish
	case "ga":
		return PluralIrish

	// Welsh
	case "cy":
		return PluralWelsh

	default:
		return PluralEnglish
	}
}

// PluralEnglish handles English-style pluralization.
// one: n == 1
// other: everything else
func PluralEnglish(n int) PluralForm {
	if n == 1 {
		return PluralOne
	}
	return PluralOther
}

// PluralFrench handles French-style pluralization.
// one: n == 0 or n == 1
// other: everything else
func PluralFrench(n int) PluralForm {
	if n == 0 || n == 1 {
		return PluralOne
	}
	return PluralOther
}

// PluralRussian handles Russian-style pluralization.
// one: n mod 10 == 1 and n mod 100 != 11
// few: n mod 10 in 2..4 and n mod 100 not in 12..14
// many: n mod 10 == 0 or n mod 10 in 5..9 or n mod 100 in 11..14
// other: everything else (fractional numbers, but we use int)
func PluralRussian(n int) PluralForm {
	if n < 0 {
		n = -n
	}

	mod10 := n % 10
	mod100 := n % 100

	if mod10 == 1 && mod100 != 11 {
		return PluralOne
	}

	if mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14) {
		return PluralFew
	}

	if mod10 == 0 || (mod10 >= 5 && mod10 <= 9) || (mod100 >= 11 && mod100 <= 14) {
		return PluralMany
	}

	return PluralOther
}

// PluralPolish handles Polish pluralization.
// one: n == 1
// few: n mod 10 in 2..4 and n mod 100 not in 12..14
// many: n != 1 and n mod 10 in 0..1 or n mod 10 in 5..9 or n mod 100 in 12..14
// other: everything else
func PluralPolish(n int) PluralForm {
	if n == 1 {
		return PluralOne
	}

	mod10 := n % 10
	mod100 := n % 100

	if mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14) {
		return PluralFew
	}

	if mod10 == 0 || mod10 == 1 || (mod10 >= 5 && mod10 <= 9) || (mod100 >= 12 && mod100 <= 14) {
		return PluralMany
	}

	return PluralOther
}

// PluralCzech handles Czech/Slovak pluralization.
// one: n == 1
// few: n in 2..4
// other: everything else
func PluralCzech(n int) PluralForm {
	if n == 1 {
		return PluralOne
	}
	if n >= 2 && n <= 4 {
		return PluralFew
	}
	return PluralOther
}

// PluralArabic handles Arabic pluralization.
// zero: n == 0
// one: n == 1
// two: n == 2
// few: n mod 100 in 3..10
// many: n mod 100 in 11..99
// other: everything else
func PluralArabic(n int) PluralForm {
	if n == 0 {
		return PluralZero
	}
	if n == 1 {
		return PluralOne
	}
	if n == 2 {
		return PluralTwo
	}

	mod100 := n % 100
	if mod100 >= 3 && mod100 <= 10 {
		return PluralFew
	}
	if mod100 >= 11 && mod100 <= 99 {
		return PluralMany
	}

	return PluralOther
}

// PluralAsian handles Asian languages with no grammatical plural.
// Always returns "other".
func PluralAsian(n int) PluralForm {
	return PluralOther
}

// PluralRomanian handles Romanian pluralization.
// one: n == 1
// few: n == 0 or n mod 100 in 1..19
// other: everything else
func PluralRomanian(n int) PluralForm {
	if n == 1 {
		return PluralOne
	}

	mod100 := n % 100
	if n == 0 || (mod100 >= 1 && mod100 <= 19) {
		return PluralFew
	}

	return PluralOther
}

// PluralLithuanian handles Lithuanian pluralization.
// one: n mod 10 == 1 and n mod 100 not in 11..19
// few: n mod 10 in 2..9 and n mod 100 not in 11..19
// other: everything else
func PluralLithuanian(n int) PluralForm {
	mod10 := n % 10
	mod100 := n % 100

	if mod10 == 1 && (mod100 < 11 || mod100 > 19) {
		return PluralOne
	}

	if mod10 >= 2 && mod10 <= 9 && (mod100 < 11 || mod100 > 19) {
		return PluralFew
	}

	return PluralOther
}

// PluralLatvian handles Latvian pluralization.
// zero: n == 0
// one: n mod 10 == 1 and n mod 100 != 11
// other: everything else
func PluralLatvian(n int) PluralForm {
	if n == 0 {
		return PluralZero
	}

	mod10 := n % 10
	mod100 := n % 100

	if mod10 == 1 && mod100 != 11 {
		return PluralOne
	}

	return PluralOther
}

// PluralIrish handles Irish pluralization.
// one: n == 1
// two: n == 2
// few: n in 3..6
// many: n in 7..10
// other: everything else
func PluralIrish(n int) PluralForm {
	if n == 1 {
		return PluralOne
	}
	if n == 2 {
		return PluralTwo
	}
	if n >= 3 && n <= 6 {
		return PluralFew
	}
	if n >= 7 && n <= 10 {
		return PluralMany
	}
	return PluralOther
}

// PluralWelsh handles Welsh pluralization.
// zero: n == 0
// one: n == 1
// two: n == 2
// few: n == 3
// many: n == 6
// other: everything else
func PluralWelsh(n int) PluralForm {
	switch n {
	case 0:
		return PluralZero
	case 1:
		return PluralOne
	case 2:
		return PluralTwo
	case 3:
		return PluralFew
	case 6:
		return PluralMany
	default:
		return PluralOther
	}
}

// RegisterPluralFunc registers a custom plural function for a locale.
func (b *Bundle) RegisterPluralFunc(locale string, fn PluralFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()

	loc, exists := b.locales[locale]
	if !exists {
		loc = &Locale{
			Tag:      locale,
			Messages: make(map[string]string),
		}
		b.locales[locale] = loc
	}

	loc.PluralRules = fn
}
