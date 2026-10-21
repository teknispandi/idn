package v3

// --- Balinese Validator (und-Bali) ---

type BalineseLanguageValidator struct{}

// Repertoire map untuk Balinese berdasarkan LGR
var balineseRepertoire = map[rune]bool{
	0x002D: true, // hyphen
	// Independent vowels 1B05–1B12
	0x1B05: true, 0x1B06: true, 0x1B07: true, 0x1B08: true,
	0x1B09: true, 0x1B0A: true, 0x1B0B: true, 0x1B0C: true,
	0x1B0D: true, 0x1B0E: true, 0x1B0F: true, 0x1B10: true,
	0x1B11: true, 0x1B12: true,

	// Consonants 1B13–1B33
	0x1B13: true, 0x1B14: true, 0x1B15: true, 0x1B16: true,
	0x1B17: true, 0x1B18: true, 0x1B19: true, 0x1B1A: true,
	0x1B1B: true, 0x1B1C: true, 0x1B1D: true, 0x1B1E: true,
	0x1B1F: true, 0x1B20: true, 0x1B21: true, 0x1B22: true, 0x1B23: true,
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
	return r >= 0x1B13 && r <= 0x1B33
}

func isBalineseIndependentVowel(r rune) bool {
	return r >= 0x1B05 && r <= 0x1B12
}

func isBalineseDependentVowel(r rune) bool {
	return r >= 0x1B35 && r <= 0x1B43
}

func isBalineseSign(r rune) bool {
	return r == 0x1B02 || r == 0x1B03 || r == 0x1B04 || r == 0x1B44
}

func (b *BalineseLanguageValidator) IsValidLabelForLanguage(label string) bool {
	runes := []rune(label)

	for i, r := range runes {
		// 2. Must be in repertoire
		if !balineseRepertoire[r] {
			return false
		}

		// get previous rune safely
		prev := rune(0)
		if i > 0 {
			prev = runes[i-1]
		}

		// 3. Dependent vowel rules
		if isBalineseDependentVowel(r) {
			if r == 0x1B35 {
				// must follow consonant
				if !isBalineseConsonant(prev) {
					return false
				}
			} else {
				// must follow consonant or independent vowel
				if !(isBalineseConsonant(prev) || isBalineseIndependentVowel(prev)) {
					return false
				}
			}
		}

		// 4. Sign rules
		if isBalineseSign(r) {
			// sign cannot be first
			if i == 0 {
				return false
			}
			// must follow consonant, iv, or dv
			if !(isBalineseConsonant(prev) || isBalineseIndependentVowel(prev) || isBalineseDependentVowel(prev)) {
				return false
			}

			// Special rules for 1B44 (adeg-adeg / virama)
			if r == 0x1B44 {
				// It suppresses the inherent vowel of the consonant it attaches
				// to, so it is only meaningful directly after a consonant. This
				// subsumes the older "not after a sign" and "not after a
				// dependent vowel" checks.
				//
				// Note it must NOT also reject a virama whose two preceding
				// runes are both consonants: Balinese is an abugida, so a bare
				// consonant carries an inherent /a/ and CC is simply two
				// syllables. Rejecting that made every word with a cluster past
				// the first syllable unregisterable — ᬩᬲ᭄ᬓᬭ (baskara),
				// ᬫᬦ᭄ᬢᬭᬶ (mantri) — while leaving ᬲ᭄ᬓᬭ (skara) valid.
				if !isBalineseConsonant(prev) {
					return false
				}
				// A label may not end on a virama: there is no following
				// consonant for the gantungan to attach to.
				if i == len(runes)-1 {
					return false
				}
			}
		}
	}

	return true
}
