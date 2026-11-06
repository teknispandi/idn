package v3

import (
	"bytes"
	"encoding/xml"
	"os"
	"strconv"
	"testing"
)

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
		{"gemination", bTA + bADEG + bTA, true},

		// --- 2-base stacking (LGR disallow-for-1B44, C ᭄ C ᭄ C) ---
		{"two stacked clusters are rejected", bKA + bADEG + bSA + bADEG + bKA, false},
		{"jembrana stacks two gantungan", "ᬚᭂᬫ᭄ᬩ᭄ᬭᬦ", false},
		{"second virama without a following consonant is not stacking", bKA + bADEG + bSA + bADEG, true},

		// --- adeg-adeg placement ---
		// Regression: a label-final adeg-adeg used to be rejected, but that
		// is how a word-final consonant is written, and the LGR allows it.
		{"adeg-adeg can end a label", bKA + bADEG, true},
		{"sepat", "ᬲᭂᬧᬢ᭄", true},
		{"tabanan", "ᬢᬩᬦᬦ᭄", true},
		{"ubud", "ᬉᬩᬸᬤ᭄", true},
		{"adeg-adeg may follow an independent vowel", bAKRA + bADEG + bKA, true},
		{"adeg-adeg cannot start a label", bADEG + bKA + bRA, false},
		{"adeg-adeg cannot follow a hyphen", bKA + "-" + bADEG + bKA, false},
		{"adeg-adeg cannot follow a dependent vowel", bKA + bULU + bADEG + bKA, false},
		{"adeg-adeg cannot follow another sign", bKA + bCECK + bADEG + bKA, false},
		{"adeg-adeg cannot double", bKA + bADEG + bADEG + bKA, false},

		// --- repertoire coverage ---
		{"DA MURDA ALPAPRANA is excluded by the LGR", bDAMA, false},
		{"DA MURDA ALPAPRANA in a cluster", bDAMA + bADEG + bKA, false},

		// --- plain labels ---
		{"single consonant", bKA, true},
		{"bali", bBA + bLA + bULU, true},
		{"bare consonant run", bBA + bSA + bKA + bRA, true},
		{"independent vowel", bAKRA, true},
		{"hyphen is allowed", bKA + "-" + bRA, true},

		// --- hyphen placement (RFC 5891 §4.2.3.1, counted in runes) ---
		{"hyphen cannot start a label", "-" + bKA, false},
		{"hyphen cannot end a label", bKA + "-", false},
		{"no -- at positions 3 and 4", bKA + bKA + "--" + bKA, false},
		{"-- elsewhere is allowed", bKA + "--" + bKA, true},

		// --- outside the repertoire ---
		{"latin is rejected", "baskara", false},
		{"rerekan is not in the repertoire", bKA + "᬴", false},
		{"ulu candra is not in the repertoire", bKA + "ᬁ", false},
		{"empty label", "", false},

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

// balineseRepertoire must be exactly the <data> repertoire of the reference
// LGR this validator implements — adding a code point the LGR excludes (as
// happened with U+1B1F) lets through labels the registry will refuse.
func TestBalineseRepertoireMatchesLGR(t *testing.T) {
	raw, err := os.ReadFile("../lgrs/second-level-reference/und-Bali.xml")
	if err != nil {
		t.Skipf("reference LGR not available: %v", err)
	}
	var doc struct {
		Chars []struct {
			CP string `xml:"cp,attr"`
		} `xml:"data>char"`
	}
	if err := xml.Unmarshal(bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf")), &doc); err != nil {
		t.Fatal(err)
	}
	lgr := map[rune]bool{}
	for _, c := range doc.Chars {
		cp, err := strconv.ParseUint(c.CP, 16, 32)
		if err != nil {
			t.Fatal(err)
		}
		lgr[rune(cp)] = true
	}
	if len(lgr) != 66 {
		t.Fatalf("parsed %d repertoire entries from the LGR, expected 66", len(lgr))
	}
	for r := range lgr {
		if !balineseRepertoire[r] {
			t.Errorf("U+%04X is in the LGR but missing from balineseRepertoire", r)
		}
	}
	for r := range balineseRepertoire {
		if !lgr[r] {
			t.Errorf("U+%04X is in balineseRepertoire but not in the LGR", r)
		}
	}
}

// isBalineseConsonant and balineseRepertoire must agree on 1B13–1B33, or a
// consonant is silently unusable (or an excluded one is treated as a base).
func TestBalineseConsonantClassMatchesRepertoire(t *testing.T) {
	for r := rune(0x1B13); r <= 0x1B33; r++ {
		if isBalineseConsonant(r) != balineseRepertoire[r] {
			t.Errorf("U+%04X: isBalineseConsonant=%v but in repertoire=%v", r, isBalineseConsonant(r), balineseRepertoire[r])
		}
	}
}
