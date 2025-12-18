// pantry/text/fold.go
package text

import (
	"strings"
	"sync"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// High is the sentinel used to form the exclusive upper bound for prefix ranges.
// U+10FFFD is the highest valid scalar value that's not a noncharacter.
// Using this instead of U+FFFF ensures astral-plane code points (e.g., emoji)
// still fall inside [prefix, prefix+High) ranges.
const High = "\U0010FFFD"

// chainPool avoids per-call allocations.
// We create a fresh NFD → strip combining marks (Mn) → NFC pipeline per borrower.
var chainPool = sync.Pool{
	New: func() any {
		return transform.Chain(
			norm.NFD,
			runes.Remove(runes.In(unicode.Mn)), // remove combining diacritics
			norm.NFC,
		)
	},
}

// Fold lowercases and strips *combining* diacritics via NFD→remove(Mn)→NFC.
// It does not guarantee ASCII; characters like "ø" or "ß" remain.
// Safe for user input; returns "" for blank/whitespace-only strings.
func Fold(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	// ASCII fast path: if already ASCII and has no A..Z, we can skip ToLower+transform.
	if isASCIIAndLower(s) {
		return s
	}

	// Unicode-aware lowercasing first, then strip combining marks.
	s = strings.ToLower(s)

	t := chainPool.Get().(transform.Transformer)
	defer func() {
		t.Reset()
		chainPool.Put(t)
	}()

	out, _, _ := transform.String(t, s)
	return out
}

// FoldTokens folds and then splits on whitespace.
// Handy for building prefix/term queries from a single input field.
func FoldTokens(s string) []string {
	f := Fold(s)
	if f == "" {
		return nil
	}
	return strings.Fields(f)
}

// PrefixRange returns the half-open range [lo, hi) for a raw query string q:
//
//	lo = Fold(q)
//	hi = lo + High
//
// If q is empty after folding, both lo and hi are "".
func PrefixRange(q string) (lo, hi string) {
	lo = Fold(q)
	if lo == "" {
		return "", ""
	}
	return lo, lo + High
}

// HiFromFolded returns folded + High. If folded is empty, returns "".
// Use this when you've already computed Fold(q) and want the upper bound.
func HiFromFolded(folded string) string {
	if folded == "" {
		return ""
	}
	return folded + High
}

// isASCIIAndLower reports whether s contains only ASCII bytes and no A..Z.
// (Digits, spaces, punctuation are fine.)
func isASCIIAndLower(s string) bool {
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b >= 0x80 {
			return false
		}
		if b >= 'A' && b <= 'Z' {
			return false
		}
	}
	return true
}
