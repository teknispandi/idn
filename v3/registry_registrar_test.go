package v3

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"unicode"

	idn "github.com/al0fayz/idn"
	"golang.org/x/text/secure/precis"
)

// Sampel label Bali (lihat juga balinese_language_validator_test.go).
const (
	baliOK          = bBA + bLA + bULU                                               // ᬩᬮᬶ
	baliSecondVir   = bKA + bADEG + bKA + bADEG + bKA                                // C ᭄ C ᭄ C
	baliLeadingSign = bADEG + bKA                                                    // diawali tanda kombinasi
	baliWithLatin   = bBA + "a"                                                      // campuran Bali + Latin
	hiragana60      = "ああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああ" // A-label > 63 oktet
)

func TestPrepareLabel(t *testing.T) {
	tests := []struct {
		label  string
		ulabel string
		alabel string
		want   error
	}{
		// LDH
		{"example", "example", "example", nil},
		{"EXAMPLE", "example", "example", nil},
		{"", "", "", ErrEmptyLabel},
		{"-abc", "", "", ErrLeadingHyphen},
		{"abc-", "", "", ErrTrailingHyphen},
		{"ab--cd", "", "", ErrHyphen34},
		{"ab_cd", "", "", ErrInvalidLDH},
		{strings.Repeat("a", 64), "", "", ErrLabelTooLong},

		// A-label: harus didekode, divalidasi, dan kanonik
		{"xn--caf-dma", "café", "xn--caf-dma", nil},
		{"XN--CAF-DMA", "café", "xn--caf-dma", nil},
		{"xn--ls8h", "", "", ErrDisallowedCodepoint}, // emoji; dulu lolos sebagai "Latin"
		{"xn--abc-", "", "", ErrInvalidALabel},       // dekode ke ASCII saja
		{"xn--cafe-yvc", "", "", ErrNotNFC},          // "cafe" + U+0301
		{"xn--", "", "", ErrInvalidALabel},

		// U-label
		{"café", "café", "xn--caf-dma", nil},
		{"Café", "café", "xn--caf-dma", nil},               // hanya ASCII yang diturunkan
		{"CAFÉ", "", "", ErrDisallowedCodepoint},           // É huruf besar: normalisasi dulu
		{"☃", "", "", ErrDisallowedCodepoint},              // NV8: x/net/idna meloloskannya
		{"Ꭰ", "Ꭰ", "", nil},                                // huruf besar Cherokee stabil → PVALID
		{"ꭰ", "", "", ErrDisallowedCodepoint},              // huruf kecil Cherokee dilipat → Unstable
		{"éé--a", "", "", ErrHyphen34},                     // posisi 3–4 dihitung per code point
		{hiragana60, "", "", ErrLabelTooLong},              // panjang diukur di A-label
		{"café", "", "", ErrNotNFC},                       // tidak NFC
		{baliLeadingSign, "", "", ErrLeadingCombiningMark}, // RFC 5891 §4.2.3.2

		// CONTEXTO / CONTEXTJ (RFC 5892 Appendix A)
		{"col·legi", "col·legi", "xn--collegi-xma", nil},
		{"a·b", "", "", ErrContextRule},
		{"・", "", "", ErrContextRule},
		{"カ・カ", "カ・カ", "", nil},
		{"١۱", "", "", ErrContextRule},
		{"a‍b", "", "", ErrContextRule},

		// Bidi (RFC 5893)
		{"مثال", "مثال", "xn--mgbh0fb", nil},
		{"1مثال", "", "", ErrBidiRule},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			p, err := prepareLabel(tt.label)
			if tt.want != nil {
				if !errors.Is(err, tt.want) {
					t.Fatalf("prepareLabel(%q) error = %v, want %v", tt.label, err, tt.want)
				}
				var ve *ValidationError
				if !errors.As(err, &ve) {
					t.Fatalf("error %T is not *ValidationError", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("prepareLabel(%q) unexpected error: %v", tt.label, err)
			}
			if p.ulabel != tt.ulabel {
				t.Errorf("ulabel = %q, want %q", p.ulabel, tt.ulabel)
			}
			if tt.alabel != "" && p.alabel != tt.alabel {
				t.Errorf("alabel = %q, want %q", p.alabel, tt.alabel)
			}
		})
	}
}

// Setiap code point di setiap tabel harus lolos lapisan IDNA2008; kalau
// tidak, label yang diizinkan tabel akan ditolak sebelum tabelnya dicek.
func TestTablesAreIDNA2008Valid(t *testing.T) {
	check := func(name string, table *IdnTable) {
		for cp := range table.validCodepoints {
			if !isIDNA2008Allowed(rune(cp)) {
				t.Errorf("table %s: %U is not PVALID/CONTEXTO under IDNA2008", name, rune(cp))
			}
		}
	}

	loader := NewIdnLoaderFS(idn.Tables())
	languages, err := loader.ListAvailableLanguages()
	if err != nil {
		t.Fatal(err)
	}
	for _, language := range languages {
		table, err := loader.LoadTable(language)
		if err != nil {
			t.Fatal(err)
		}
		check(language, table)
	}
	for tableType, table := range NewIdnTableRegistry().GetAllTables() {
		check(tableType.String(), table)
	}
}

func newTestManager(t *testing.T) *IdnManager {
	t.Helper()
	m := NewIdnManagerFS(idn.Tables())
	for zone, languages := range map[string][]string{
		"id":     {"latin", "und-bali"},
		"co.id":  {"latin"},
		"multi":  {"latin", "arabic", "japanese"},
		"off.id": {"latin"},
	} {
		if err := m.RegisterTld(zone, languages); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.RegisterTldConfig(TldIdnConfig{TldName: "рф", Languages: []string{"cyrillic"}, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := m.DisableTld("off.id"); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestIdnManager_CheckDomain(t *testing.T) {
	m := newTestManager(t)

	tests := []struct {
		domain    string
		ascii     string
		unicode   string
		zone      string
		table     string
		isIDN     bool
		want      error
		wantTable string // tabel yang disebut di error
		wantRule  string
	}{
		{domain: "example.id", ascii: "example.id", unicode: "example.id", zone: "id"},
		{domain: " EXAMPLE.ID. ", ascii: "example.id", unicode: "example.id", zone: "id"},
		{domain: "café.id", ascii: "xn--caf-dma.id", unicode: "café.id", zone: "id", table: "latin", isIDN: true},
		{domain: "xn--caf-dma.id", ascii: "xn--caf-dma.id", unicode: "café.id", zone: "id", table: "latin", isIDN: true},
		{domain: baliOK + ".id", unicode: baliOK + ".id", zone: "id", table: "und-bali", isIDN: true},
		{domain: "example.co.id", ascii: "example.co.id", unicode: "example.co.id", zone: "co.id"},
		{domain: "пример.рф", ascii: "xn--e1afmkfd.xn--p1ai", unicode: "пример.рф", zone: "xn--p1ai", table: "cyrillic", isIDN: true},
		{domain: "пример.xn--p1ai", ascii: "xn--e1afmkfd.xn--p1ai", unicode: "пример.рф", zone: "xn--p1ai", table: "cyrillic", isIDN: true},

		{domain: "xn--ls8h.id", want: ErrDisallowedCodepoint},
		{domain: "-abc.id", want: ErrLeadingHyphen},
		{domain: "日本.id", want: ErrCodepointNotInTable},
		{domain: baliSecondVir + ".id", want: ErrLanguageRule, wantTable: "und-bali", wantRule: "disallow-for-1B44"},
		{domain: baliWithLatin + ".id", want: ErrCodepointNotInTable, wantTable: "und-bali"},
		{domain: "example.рф", want: ErrCodepointNotInTable, wantTable: "cyrillic"}, // zona khusus IDN
		{domain: "www.example.id", want: ErrNotRegistrable},
		{domain: "co.id", want: ErrNotRegistrable},
		{domain: "example.com", want: ErrUnknownTld},
		{domain: "example.off.id", want: ErrTldDisabled},
		{domain: "", want: ErrEmptyLabel},
		{domain: strings.Repeat(strings.Repeat("a", 63)+".", 4) + "id", want: ErrDomainTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			res, err := m.CheckDomain(tt.domain)
			if tt.want != nil {
				if !errors.Is(err, tt.want) {
					t.Fatalf("CheckDomain(%q) error = %v, want %v", tt.domain, err, tt.want)
				}
				var ve *ValidationError
				if errors.As(err, &ve) {
					if tt.wantTable != "" && ve.Table != tt.wantTable {
						t.Errorf("error table = %q, want %q (%v)", ve.Table, tt.wantTable, err)
					}
					if tt.wantRule != "" && ve.Rule != tt.wantRule {
						t.Errorf("error rule = %q, want %q (%v)", ve.Rule, tt.wantRule, err)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("CheckDomain(%q) unexpected error: %v", tt.domain, err)
			}
			if tt.ascii != "" && res.ASCIIName != tt.ascii {
				t.Errorf("ASCIIName = %q, want %q", res.ASCIIName, tt.ascii)
			}
			if res.UnicodeName != tt.unicode {
				t.Errorf("UnicodeName = %q, want %q", res.UnicodeName, tt.unicode)
			}
			if res.Zone != tt.zone {
				t.Errorf("Zone = %q, want %q", res.Zone, tt.zone)
			}
			if res.TableName() != tt.table {
				t.Errorf("TableName() = %q, want %q", res.TableName(), tt.table)
			}
			if res.IsIDN != tt.isIDN {
				t.Errorf("IsIDN = %v, want %v", res.IsIDN, tt.isIDN)
			}
		})
	}
}

func TestIdnManager_CheckDomainForTable(t *testing.T) {
	m := newTestManager(t)

	if res, err := m.CheckDomainForTable("café.id", "Latin"); err != nil || res.TableName() != "latin" {
		t.Errorf("latin: got %v, %v", res, err)
	}
	if _, err := m.CheckDomainForTable("café.id", "und-bali"); !errors.Is(err, ErrCodepointNotInTable) {
		t.Errorf("und-bali: error = %v, want ErrCodepointNotInTable", err)
	}
	if _, err := m.CheckDomainForTable("café.id", "chinese"); !errors.Is(err, ErrNoIdnTable) {
		t.Errorf("chinese (not configured): error = %v, want ErrNoIdnTable", err)
	}
}

func TestIdnManager_DisabledTldIsRejected(t *testing.T) {
	m := newTestManager(t)

	if ok, _, err := m.ValidateLabel("off.id", "abc"); ok || !errors.Is(err, ErrTldDisabled) {
		t.Errorf("ValidateLabel on disabled TLD = %v, %v; want false, ErrTldDisabled", ok, err)
	}
	if _, err := m.CheckLabel("off.id", "abc"); !errors.Is(err, ErrTldDisabled) {
		t.Errorf("CheckLabel on disabled TLD error = %v", err)
	}
	if err := m.EnableTld("off.id"); err != nil {
		t.Fatal(err)
	}
	if ok, _, err := m.ValidateLabel("off.id", "abc"); !ok || err != nil {
		t.Errorf("after EnableTld: %v, %v", ok, err)
	}
}

func TestIdnManager_ValidateLabelIsDeterministic(t *testing.T) {
	m := newTestManager(t)
	want := []string{"latin", "arabic", "japanese"} // urutan konfigurasi
	for i := 0; i < 50; i++ {
		ok, languages, err := m.ValidateLabel("multi", "123")
		if err != nil || !ok || !reflect.DeepEqual(languages, want) {
			t.Fatalf("run %d: ValidateLabel = %v, %v, %v; want true, %v", i, ok, languages, err, want)
		}
	}
}

func TestIdnManager_AcceptsALabels(t *testing.T) {
	m := newTestManager(t)
	ok, languages, err := m.ValidateLabel("id", "xn--ls8h")
	if ok || err != nil {
		t.Errorf("ValidateLabel(xn--ls8h) = %v, %v, %v; want rejected", ok, languages, err)
	}
	ok, err = m.ValidateLabelForLanguage("id", "latin", "xn--caf-dma")
	if !ok || err != nil {
		t.Errorf("ValidateLabelForLanguage(xn--caf-dma) = %v, %v; want true", ok, err)
	}
}

func TestIdnManager_ConcurrentUse(t *testing.T) {
	m := newTestManager(t)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				if _, err := m.CheckDomain("café.id"); err != nil {
					t.Error(err)
					return
				}
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = m.AddLanguageToTld("multi", "greek")
				_ = m.RemoveLanguageFromTld("multi", "greek")
			}
		}()
	}
	wg.Wait()
}

func TestIdnLoader_FindsAllTables(t *testing.T) {
	loader := NewIdnLoaderFS(idn.Tables())

	languages, err := loader.ListAvailableLanguages()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(languages[len(languages)-1], "und-bali") {
		t.Errorf("ListAvailableLanguages not sorted or missing und-bali: %v", languages)
	}

	for _, name := range []string{"und-bali", "UND-Bali", "Traditional Chinese", "traditional_chinese"} {
		if _, err := loader.LoadTable(name); err != nil {
			t.Errorf("LoadTable(%q): %v", name, err)
		}
	}

	bali, _ := loader.LoadTable("und-bali")
	if _, ok := bali.languageValidator.(*BalineseLanguageValidator); !ok {
		t.Errorf("und-bali validator = %T, want *BalineseLanguageValidator", bali.languageValidator)
	}
	arabic, _ := loader.LoadTable("arabic")
	if _, ok := arabic.languageValidator.(*ArabicLanguageValidator); !ok {
		t.Errorf("arabic validator = %T, want *ArabicLanguageValidator", arabic.languageValidator)
	}
}

func TestIdnTable_Metadata(t *testing.T) {
	loader := NewIdnLoaderFS(idn.Tables())

	latin, _ := loader.LoadTable("latin")
	if latin.GetScript() != "Latn" || latin.GetVersion() != "2.0" || latin.GetEffectiveDate() != "2023-04-04" {
		t.Errorf("latin metadata = %q %q %q", latin.GetScript(), latin.GetVersion(), latin.GetEffectiveDate())
	}
	if got := latin.GetURL().String(); got != "https://www.iana.org/domains/idn-tables/tables/google_latn_2.0.txt" {
		t.Errorf("latin URL = %q", got)
	}

	// Tabel tanpa header # URL: tidak boleh mendapat URL karangan.
	chinese, _ := loader.LoadTable("chinese")
	if chinese.GetURL() != nil || chinese.GetPolicy() != nil {
		t.Errorf("chinese URL/policy = %v / %v, want nil", chinese.GetURL(), chinese.GetPolicy())
	}
}

func TestArabicDigitHomogeneity(t *testing.T) {
	arabic, err := NewIdnLoaderFS(idn.Tables()).LoadTable("arabic")
	if err != nil {
		t.Fatal(err)
	}
	for label, want := range map[string]bool{
		"مثال١٢": true,
		"مثال12": true,
		"مثال1٢": false, // header tabel: jenis digit tidak boleh dicampur
	} {
		if got := arabic.IsValidLabel(label); got != want {
			t.Errorf("IsValidLabel(%q) = %v, want %v (%v)", label, got, want, arabic.CheckLabel(label))
		}
	}
}

func TestJapaneseValidatorUsesUnicodeBlocks(t *testing.T) {
	v := &JapaneseLanguageValidator{}
	for label, want := range map[string]bool{
		"カーー":  true,  // ー di blok Katakana (seperti Nomulus)
		"ーカ":   false, // ー tidak boleh di awal
		"aー":   false, // ー harus setelah Hiragana/Katakana
		"・〆":   false, // hanya karakter pengecualian
		"日本・語": true,
		"あいうえおかきくけこさしすせそた": false, // > 15 code point
	} {
		if got := v.IsValidLabelForLanguage(label); got != want {
			t.Errorf("IsValidLabelForLanguage(%q) = %v, want %v", label, got, want)
		}
	}
}

func TestIdnLabelValidator_RejectsWhatNomulusRejects(t *testing.T) {
	v := NewIdnLabelValidator()
	latin := &MockTld{idnTables: []IdnTableType{UnconfusableLatin}}

	for _, label := range []string{"", "xn--ls8h", "-abc", "ab--cd", "CAFÉ"} {
		if name, ok := v.FindValidIdnTableForTld(label, latin); ok {
			t.Errorf("FindValidIdnTableForTld(%q) = %q, true; want rejected", label, name)
		}
	}
	if name, ok := v.FindValidIdnTableForTld("xn--caf-dma", latin); !ok || name != "unconfusable_latin" {
		t.Errorf("FindValidIdnTableForTld(xn--caf-dma) = %q, %v", name, ok)
	}
	if !v.ValidateLabel("xn--caf-dma", UnconfusableLatin) {
		t.Error("ValidateLabel(xn--caf-dma) = false, want true")
	}
}

func TestIdnUtils_RegistrarHelpers(t *testing.T) {
	u := DefaultIdnUtils

	for in, want := range map[string]string{
		"  ＥＸＡＭＰＬＥ。ＩＤ ":  "example.id",
		"Café.ID.":       "café.id",
		"xn--caf-dma.id": "café.id",
		"Straße.id":      "straße.id", // non-transitional: ß tetap
	} {
		got, err := u.NormalizeDomainName(in)
		if err != nil || got != want {
			t.Errorf("NormalizeDomainName(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := u.NormalizeDomainName(" . "); !errors.Is(err, ErrEmptyLabel) {
		t.Errorf("NormalizeDomainName(blank) error = %v", err)
	}

	if _, err := u.ToASCII("CAFÉ.id"); !errors.Is(err, ErrDisallowedCodepoint) {
		t.Errorf("ToASCII(CAFÉ.id) error = %v, want ErrDisallowedCodepoint", err)
	}
	if got, err := u.ToASCII("example.id."); err != nil || got != "example.id." {
		t.Errorf("ToASCII keeps trailing dot: %q, %v", got, err)
	}
	if _, err := u.ToUnicode("xn--CAF-pia.id"); err == nil {
		t.Error("ToUnicode(xn--CAF-pia.id) should fail: decodes to uppercase É")
	}

	for label, want := range map[string]bool{"ab--cd": false, "☃": false, hiragana60: false, "例子": true} {
		if got := u.IsValidDomainLabel(label); got != want {
			t.Errorf("IsValidDomainLabel(%q) = %v, want %v", label, got, want)
		}
	}

	for s, want := range map[string]bool{"ひらがなカタカナ漢字": false, "한국語": false, "abcабв": true, "café-1": false} {
		if got := u.HasMixedScripts(s); got != want {
			t.Errorf("HasMixedScripts(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestValidationError_Message(t *testing.T) {
	m := newTestManager(t)
	_, err := m.CheckDomain(baliSecondVir + ".id")
	want := `idn: label "` + baliSecondVir + `": label violates the IDN table's language rules: U+1B44 '᭄' at position 3 (table und-bali, rule disallow-for-1B44)`
	if err == nil || err.Error() != want {
		t.Errorf("error message:\n got: %v\nwant: %s", err, want)
	}
}

// Aproksimasi PVALID (isIDNA2008Allowed + registrationProfile) dibandingkan
// dengan PRECIS IdentifierClass (RFC 8264, x/text), yang diturunkan dari
// kategori RFC 5892 yang sama. Selisih yang diketahui dan disengaja:
//   - IgnorableBlocks (RFC 5892 §2.4) ditolak di sini, diterima PRECIS;
//   - sejumlah huruf pengubah Lm (mis. U+02B9, U+A717) PVALID di IDNA2008
//     tetapi ditolak PRECIS; status PVALID-nya dicek terhadap tabel IANA di
//     TestIDNA2008AgainstIANA.
func TestIDNA2008MatchesPrecis(t *testing.T) {
	ignorableBlock := func(r rune) bool {
		return (r >= 0x20D0 && r <= 0x20FF) || (r >= 0x1D100 && r <= 0x1D24F)
	}
	overRejected, lenient := 0, 0
	for r := rune(0x80); r <= unicode.MaxRune; r++ {
		if (r >= 0xD800 && r <= 0xDFFF) || isContextO(r) || r == 0x200C || r == 0x200D {
			continue
		}
		s := string(r)
		if unicode.In(r, unicode.Mn, unicode.Mc, unicode.Me) {
			s = "a" + s // tanda kombinasi tidak boleh di awal label
		}
		_, precisErr := precis.UsernameCasePreserved.String(s)
		_, idnaErr := registrationProfile.ToASCII(s)
		pipeline := isIDNA2008Allowed(r) && idnaErr == nil
		oracle := precisErr == nil && idnaErr == nil

		if oracle && !pipeline && !ignorableBlock(r) {
			overRejected++
			if overRejected <= 10 {
				t.Errorf("%U rejected here but valid in PRECIS and UTS46", r)
			}
		}
		if pipeline && !oracle {
			lenient++
			if !unicode.Is(unicode.Lm, r) {
				t.Errorf("%U accepted here but rejected by PRECIS (not a modifier letter)", r)
			}
		}
	}
	t.Logf("modifier letters accepted here but not by PRECIS: %d", lenient)
}

// Baris tabel turunan IANA (IDNA Parameters, idna-tables-12.0.0), dikutip
// dari https://www.iana.org/assignments/idna-tables-12.0.0/ — termasuk huruf
// pengubah yang ditolak PRECIS tetapi PVALID di IDNA2008.
func TestIDNA2008AgainstIANA(t *testing.T) {
	rows := []struct {
		first, last rune
		pvalid      bool
	}{
		{0x02B9, 0x02C1, true},  // MODIFIER LETTER PRIME..MODIFIER LETTER REVERSED GLOTTAL STOP
		{0x02C6, 0x02D1, true},  // MODIFIER LETTER CIRCUMFLEX ACCENT..MODIFIER LETTER HALF TRIANGULAR COLON
		{0x02EC, 0x02EC, true},  // MODIFIER LETTER VOICING
		{0x2E2F, 0x2E2F, true},  // VERTICAL TILDE
		{0xA67F, 0xA67F, true},  // CYRILLIC PAYEROK
		{0xA717, 0xA71F, true},  // MODIFIER LETTER DOT VERTICAL BAR..LOW INVERTED EXCLAMATION MARK
		{0xA787, 0xA788, true},  // LATIN SMALL LETTER INSULAR T..MODIFIER LETTER LOW CIRCUMFLEX ACCENT
		{0x13A0, 0x13F5, true},  // CHEROKEE LETTER A..CHEROKEE LETTER MV
		{0xAB70, 0xABBF, false}, // CHEROKEE SMALL LETTER A..CHEROKEE SMALL LETTER YA
		{0x20D0, 0x20F0, false}, // COMBINING LEFT HARPOON ABOVE..COMBINING ASTERISK ABOVE
	}
	for _, row := range rows {
		for r := row.first; r <= row.last; r++ {
			s := string(r)
			if unicode.In(r, unicode.Mn, unicode.Mc, unicode.Me) {
				s = "a" + s
			}
			_, err := prepareLabel(s)
			if got := err == nil; got != row.pvalid {
				t.Errorf("%U: accepted=%v, IANA says PVALID=%v (%v)", r, got, row.pvalid, err)
			}
		}
	}
}
