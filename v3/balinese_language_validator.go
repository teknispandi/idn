package v3

// --- Balinese Validator (und-Bali) ---
//
// Mengikuti ICANN Second Level Reference LGR for the Balinese Script
// (2026-08-13; secara normatif sama dengan 2024-10-25),
// lgrs/second-level-reference/und-Bali.xml. Rule XML-nya yang
// normatif, bukan prosa di deskripsinya — membaca prosanya secara harfiah
// adalah asal bug "dua konsonan sebelum virama" dan "virama di akhir label".

type BalineseLanguageValidator struct{}

// Repertoire map untuk Balinese berdasarkan LGR: 65 code point Bali + hyphen.
// U+1B1F DA MURDA ALPAPRANA sengaja tidak ada — LGR mengecualikannya.
var balineseRepertoire = map[rune]bool{
	0x002D: true, // hyphen
	// Independent vowels 1B05–1B12
	0x1B05: true, 0x1B06: true, 0x1B07: true, 0x1B08: true,
	0x1B09: true, 0x1B0A: true, 0x1B0B: true, 0x1B0C: true,
	0x1B0D: true, 0x1B0E: true, 0x1B0F: true, 0x1B10: true,
	0x1B11: true, 0x1B12: true,

	// Consonants 1B13–1B33, kecuali 1B1F
	0x1B13: true, 0x1B14: true, 0x1B15: true, 0x1B16: true,
	0x1B17: true, 0x1B18: true, 0x1B19: true, 0x1B1A: true,
	0x1B1B: true, 0x1B1C: true, 0x1B1D: true, 0x1B1E: true,
	0x1B20: true, 0x1B21: true, 0x1B22: true, 0x1B23: true,
	0x1B24: true, 0x1B25: true, 0x1B26: true, 0x1B27: true,
	0x1B28: true, 0x1B29: true, 0x1B2A: true, 0x1B2B: true,
	0x1B2C: true, 0x1B2D: true, 0x1B2E: true, 0x1B2F: true,
	0x1B30: true, 0x1B31: true, 0x1B32: true, 0x1B33: true,

	// Dependent vowels 1B35–1B43
	0x1B35: true, 0x1B36: true, 0x1B37: true, 0x1B38: true,
	0x1B39: true, 0x1B3A: true, 0x1B3B: true, 0x1B3C: true,
	0x1B3D: true, 0x1B3E: true, 0x1B3F: true, 0x1B40: true,
	0x1B41: true, 0x1B42: true, 0x1B43: true,

	// Signs 1B02–1B04 dan 1B44
	0x1B02: true, 0x1B03: true, 0x1B04: true, 0x1B44: true,
}

func isBalineseConsonant(r rune) bool {
	return r >= 0x1B13 && r <= 0x1B33 && r != 0x1B1F
}

func isBalineseIndependentVowel(r rune) bool {
	return r >= 0x1B05 && r <= 0x1B12
}

func isBalineseDependentVowel(r rune) bool {
	return r >= 0x1B35 && r <= 0x1B43
}

// isBalineseSign is the LGR class "s" (tag "sign"), which includes 1B44 itself.
func isBalineseSign(r rune) bool {
	return r == 0x1B02 || r == 0x1B03 || r == 0x1B04 || r == 0x1B44
}

func (b *BalineseLanguageValidator) IsValidLabelForLanguage(label string) bool {
	return b.CheckLabelForLanguage(label) == nil
}

// CheckLabelForLanguage seperti IsValidLabelForLanguage, tetapi menyebut
// rule LGR yang dilanggar beserta posisi code point-nya.
func (b *BalineseLanguageValidator) CheckLabelForLanguage(label string) error {
	runes := []rune(label)
	n := len(runes)
	if n == 0 {
		return newLabelError(ErrEmptyLabel, label)
	}
	fail := func(i int, rule string) error {
		e := newRuneError(ErrLanguageRule, label, i, runes[i])
		e.Rule = rule
		return e
	}

	// at returns the rune at i, or -1 outside the label.
	at := func(i int) rune {
		if i < 0 || i >= n {
			return -1
		}
		return runes[i]
	}

	for i, r := range runes {
		// 1. Must be in repertoire
		if !balineseRepertoire[r] {
			e := newRuneError(ErrCodepointNotInTable, label, i, r)
			e.Rule = "repertoire"
			return e
		}

		prev := at(i - 1)

		switch {
		case r == 0x002D:
			// hyphen-minus-disallowed (RFC 5891 §4.2.3.1): not first, not
			// last, no "--" at positions 3–4. Counted in runes here because
			// x/net/idna checks it in bytes and so never fires for Balinese.
			if i == 0 || i == n-1 || (i == 3 && prev == 0x002D) {
				return fail(i, "hyphen-minus-disallowed")
			}

		case r == 0x1B35:
			// follows-c
			if !isBalineseConsonant(prev) {
				return fail(i, "follows-c")
			}

		case isBalineseDependentVowel(r):
			// follows-c-or-iv
			if !isBalineseConsonant(prev) && !isBalineseIndependentVowel(prev) {
				return fail(i, "follows-c-or-iv")
			}

		case r == 0x1B44:
			// disallow-for-1B44: not first, not after a hyphen, a sign
			// (itself included) or a dependent vowel.
			if i == 0 || prev == 0x002D || isBalineseSign(prev) || isBalineseDependentVowel(prev) {
				return fail(i, "disallow-for-1B44")
			}
			// ...and not as the second virama of C ᭄ C ᭄ C, which would stack
			// two gantungan under one base.
			//
			// There is deliberately no end-of-label rule: a final adeg-adeg
			// is how Balinese writes a word-final consonant — ᬲᭂᬧᬢ᭄, ᬢᬩᬦᬦ᭄.
			if isBalineseConsonant(at(i-3)) && at(i-2) == 0x1B44 &&
				isBalineseConsonant(prev) && isBalineseConsonant(at(i+1)) {
				return fail(i, "disallow-for-1B44")
			}

		case isBalineseSign(r):
			// follows-c-or-iv-or-dv (1B02–1B04)
			if !isBalineseConsonant(prev) && !isBalineseIndependentVowel(prev) && !isBalineseDependentVowel(prev) {
				return fail(i, "follows-c-or-iv-or-dv")
			}
		}
	}

	return nil
}
