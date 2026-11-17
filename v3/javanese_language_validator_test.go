package v3

import (
	"bytes"
	"encoding/xml"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
)

// Singkatan code point untuk kasus di bawah.
const (
	jKA      = "ꦏ" // U+A98F, basis aksara rekan
	jJA      = "ꦗ" // U+A997, basis aksara rekan
	jWA      = "ꦮ" // U+A9AE, basis aksara rekan
	jNA      = "ꦤ" // U+A9A4
	jCA      = "ꦕ" // U+A995
	jRA      = "ꦫ" // U+A9AB
	jA       = "ꦄ" // U+A984 aksara swara A
	jLAYAR   = "ꦂ" // U+A982
	jWULU    = "ꦶ" // U+A9B6 sandhangan i
	jTALING  = "ꦺ" // U+A9BA
	jTARUNG  = "ꦴ" // U+A9B4
	jCAKRA   = "ꦿ" // U+A9BF medial ra
	jPANGKON = "꧀" // U+A9C0
	jTELU    = "꦳" // U+A9B3 cecak telu
)

func TestJavaneseLanguageValidator(t *testing.T) {
	v := &JavaneseLanguageValidator{}

	valid := []string{
		jJA + jWA,                       // jawa
		jJA + jPANGKON + jWA,            // pasangan
		jKA + jCAKRA + jWULU,            // medial lalu sandhangan
		jKA + jTALING + jTARUNG,         // taling-tarung: ko
		jA + jWULU,                      // sandhangan setelah aksara swara
		jKA + jLAYAR,                    // layar setelah konsonan
		jKA + jWULU + jLAYAR,            // tanda setelah sandhangan
		jJA + jTELU + jWULU,             // aksara rekan (za) + sandhangan
		jJA + jTELU + jPANGKON + jNA,    // pangkon setelah aksara rekan
		jKA + jPANGKON + jKA + jPANGKON, // pangkon kedua tidak diikuti basis ketiga
		jKA + jPANGKON + jJA + jTELU + jPANGKON + jKA, // aksara rekan memutus pola pangkon-disallowed
		jKA + "-" + jNA, // tanda hubung di tengah
		jNA + jPANGKON + jCA + jWA + jPANGKON + jRA, // dua pasangan terpisah
	}
	for _, label := range valid {
		if err := v.CheckLabelForLanguage(label); err != nil {
			t.Errorf("CheckLabelForLanguage(%q %U) = %v, want valid", label, []rune(label), err)
		}
	}

	invalid := []struct {
		label string
		kind  error
		rule  string
		pos   int
	}{
		{"", ErrEmptyLabel, "", -1},
		{"a" + jKA, ErrCodepointNotInTable, "repertoire", 0},
		{jTELU + jKA, ErrCodepointNotInTable, "aksara-rekan", 0},
		{jNA + jTELU, ErrCodepointNotInTable, "aksara-rekan", 1}, // NA bukan basis aksara rekan
		{jPANGKON + jKA, ErrLanguageRule, "follows-cons-aksara-rekan-vowel", 0},
		{jKA + jWULU + jPANGKON, ErrLanguageRule, "follows-cons-aksara-rekan-vowel", 2},
		{jWULU + jKA, ErrLanguageRule, "follows-cons-aksara-rekan-med-vowel", 0},
		{jKA + jWULU + jTALING, ErrLanguageRule, "follows-cons-aksara-rekan-med-vowel", 2},
		{jKA + jTARUNG, ErrLanguageRule, "follows-A9BA-or-A9BB", 1},
		{jA + jCAKRA, ErrLanguageRule, "follows-cons-aksara-rekan", 1},
		{jLAYAR + jKA, ErrLanguageRule, "follows-cons-aksara-rekan-med-vowel-dep", 0},
		{"-" + jKA, ErrLanguageRule, "hyphen-minus-disallowed", 0},
		{jKA + "-", ErrLanguageRule, "hyphen-minus-disallowed", 1},
		{jKA + jNA + "--" + jKA, ErrLanguageRule, "hyphen-minus-disallowed", 3},
		{jKA + jPANGKON + jKA + jPANGKON + jKA, ErrLanguageRule, "pangkon-disallowed", 3},
		{jA + jPANGKON + jKA + jPANGKON + jA, ErrLanguageRule, "pangkon-disallowed", 3},
	}
	for _, tt := range invalid {
		err := v.CheckLabelForLanguage(tt.label)
		var ve *ValidationError
		if !errors.Is(err, tt.kind) || !errors.As(err, &ve) || ve.Rule != tt.rule || ve.Pos != tt.pos {
			t.Errorf("CheckLabelForLanguage(%q %U) = %v, want %v rule %q at %d", tt.label, []rune(tt.label), err, tt.kind, tt.rule, tt.pos)
		}
	}
}

// Repertoire validator harus sama dengan <data> LGR: code point tunggal +
// U+A9B3 yang hanya muncul di sekuens aksara rekan.
func TestJavaneseRepertoireMatchesLGR(t *testing.T) {
	raw, err := os.ReadFile("../lgrs/second-level-reference/und-Java.xml")
	if err != nil {
		t.Skipf("reference LGR not available: %v", err)
	}
	var doc struct {
		Chars []struct {
			CP  string `xml:"cp,attr"`
			Tag string `xml:"tag,attr"`
		} `xml:"data>char"`
	}
	if err := xml.Unmarshal(bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf")), &doc); err != nil {
		t.Fatal(err)
	}
	singles := map[rune]string{}
	inSequences := map[rune]bool{}
	for _, c := range doc.Chars {
		fields := strings.Fields(c.CP)
		for _, f := range fields {
			cp, err := strconv.ParseUint(f, 16, 32)
			if err != nil {
				t.Fatal(err)
			}
			if len(fields) == 1 {
				singles[rune(cp)] = c.Tag
			} else {
				inSequences[rune(cp)] = true
			}
		}
	}
	if len(singles) != 52 {
		t.Fatalf("parsed %d single code points from the LGR, expected 52", len(singles))
	}
	for r, tag := range singles {
		if !isJavaneseRepertoire(r) {
			t.Errorf("U+%04X is in the LGR but not in the validator", r)
		}
		classes := map[string]bool{
			"Consonant":        javaneseConsonants[r],
			"Vowel":            javaneseVowels[r],
			"Dependent-Vowel":  javaneseDependentVowels[r],
			"Medial-Consonant": javaneseMedials[r],
			"Sign":             javaneseSigns[r],
			"Virama":           r == javanesePangkon,
			"Hyphen":           r == '-',
		}
		for class, member := range classes {
			if member != strings.Contains(" "+tag+" ", " "+class+" ") {
				t.Errorf("U+%04X: class %s = %v, but LGR tag is %q", r, class, member, tag)
			}
		}
	}
	for r := rune(0xA980); r <= 0xA9DF; r++ {
		if isJavaneseRepertoire(r) && singles[r] == "" {
			t.Errorf("U+%04X is in the validator but not a single code point in the LGR", r)
		}
	}
	if !inSequences[javaneseCecakTelu] || singles[javaneseCecakTelu] != "" {
		t.Error("U+A9B3 must appear only inside sequences")
	}
}

// Tabel idn/und-Java.txt memasang validator ini lewat "# Script: und-Java".
func TestJavaneseTableUsesValidator(t *testing.T) {
	m := NewIdnManager("../idn")
	if err := m.RegisterTld("id", []string{"latin", "und-bali", "und-java"}); err != nil {
		t.Fatal(err)
	}
	res, err := m.CheckDomain(jJA + jWA + ".id")
	if err != nil || res.TableName() != "und-java" {
		t.Fatalf("CheckDomain(jawa.id) = %v, %v", res, err)
	}
	_, err = m.CheckDomain(jKA + jPANGKON + jKA + jPANGKON + jKA + ".id")
	var ve *ValidationError
	if !errors.As(err, &ve) || ve.Rule != "pangkon-disallowed" || ve.Table != "und-java" {
		t.Errorf("CheckDomain(C꧀C꧀C.id) error = %v, want pangkon-disallowed from und-java", err)
	}
}
