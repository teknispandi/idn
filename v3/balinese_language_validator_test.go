package v3

import "testing"

// Codepoint shorthands used across the cases below.
const (
	bKA   = "ᬓ" // ᬓ
	bNA   = "ᬦ" // ᬦ
	bTA   = "ᬢ" // ᬢ
	bBA   = "ᬩ" // ᬩ
	bMA   = "ᬫ" // ᬫ
	bRA   = "ᬭ" // ᬭ
	bLA   = "ᬮ" // ᬮ
	bSA   = "ᬲ" // ᬲ
	bDAMA = "ᬟ" // ᬟ DA MURDA ALPAPRANA
	bULU  = "ᬶ" // ◌ᬶ dependent vowel i
	bAKRA = "ᬅ" // ᬅ independent vowel a
	bADEG = "᭄" // ◌᭄ adeg-adeg (virama)
	bCECK = "ᬂ" // ◌ᬂ cecek
)

func TestBalineseLanguageValidator_IsValidLabelForLanguage(t *testing.T) {
	validator := &BalineseLanguageValidator{}

	tests := []struct {
		name     string
		label    string
		expected bool
	}{
		// --- gantungan (consonant clusters via adeg-adeg) ---
		// Regression: a virama preceded by two bare consonants used to be
		// rejected, which made any cluster past the first syllable invalid.
		{"baskara, cluster mid-word", bBA + bSA + bADEG + bKA + bRA, true},
		{"mantri, cluster mid-word", bMA + bNA + bADEG + bTA + bRA + bULU, true},
		{"skara, cluster at start", bSA + bADEG + bKA + bRA, true},
		{"two stacked clusters", bKA + bADEG + bSA + bADEG + bKA, true},
		{"gemination", bTA + bADEG + bTA, true},

		// --- adeg-adeg placement ---
		{"adeg-adeg cannot start a label", bADEG + bKA + bRA, false},
		{"adeg-adeg cannot end a label", bKA + bADEG, false},
		{"adeg-adeg cannot follow a dependent vowel", bKA + bULU + bADEG + bKA, false},
		{"adeg-adeg cannot follow another sign", bKA + bCECK + bADEG + bKA, false},
		{"adeg-adeg cannot double", bKA + bADEG + bADEG + bKA, false},

		// --- repertoire coverage ---
		// Regression: U+1B1F was missing from the repertoire map while
		// isBalineseConsonant() still counted it, so it could never be used.
		{"DA MURDA ALPAPRANA is in repertoire", bDAMA, true},
		{"DA MURDA ALPAPRANA in a cluster", bDAMA + bADEG + bKA, true},

		// --- plain labels ---
		{"single consonant", bKA, true},
		{"bali", bBA + bLA + bULU, true},
		{"bare consonant run", bBA + bSA + bKA + bRA, true},
		{"independent vowel", bAKRA, true},
		{"hyphen is allowed", bKA + "-" + bRA, true},

		// --- outside the repertoire ---
		{"latin is rejected", "baskara", false},
		{"rerekan is not in the repertoire", bKA + "᬴", false},
		{"ulu candra is not in the repertoire", bKA + "ᬁ", false},

		// --- dependent vowel rules ---
		{"dependent vowel after consonant", bKA + bULU, true},
		{"dependent vowel cannot start a label", bULU + bKA, false},
		{"tedung must follow a consonant", bAKRA + "ᬵ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validator.IsValidLabelForLanguage(tt.label)
			if got != tt.expected {
				t.Errorf("IsValidLabelForLanguage(%q) = %v, expected %v", tt.label, got, tt.expected)
			}
		})
	}
}

// Every consonant in the 1B13–1B33 range that isBalineseConsonant() accepts
// must also be present in balineseRepertoire, or it is silently unusable.
func TestBalineseRepertoireCoversAllConsonants(t *testing.T) {
	for r := rune(0x1B13); r <= 0x1B33; r++ {
		if isBalineseConsonant(r) && !balineseRepertoire[r] {
			t.Errorf("U+%04X %q is a consonant but missing from balineseRepertoire", r, string(r))
		}
	}
}
