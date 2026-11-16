package v3

import (
	"strings"
	"unicode"

	"golang.org/x/net/idna"
	"golang.org/x/text/secure/bidirule"
	"golang.org/x/text/unicode/bidi"
	"golang.org/x/text/unicode/norm"
)

// Lapisan IDNA2008 (RFC 5890–5893) yang dipakai semua jalur validasi.
//
// golang.org/x/net/idna dipakai sebagai jaring pengaman terakhir, tetapi
// tidak cukup sendirian untuk registry:
//   - posisi tanda hubung 3–4 dicek dalam byte, bukan code point;
//   - aturan CONTEXTO (RFC 5892 A.3–A.9) tidak diterapkan;
//   - karakter "NV8" (valid di UTS46 tapi bukan PVALID IDNA2008,
//     mis. ☃ U+2603) lolos;
//   - A-label tidak dicek kanonik (round-trip).
// Semua itu ditangani di sini.

const (
	acePrefix      = "xn--"
	maxLabelOctets = 63
	maxNameOctets  = 253
)

// registrationProfile: IDNA2008 ketat untuk pendaftaran — tanpa mapping,
// input huruf besar non-ASCII ditolak, bidi dan joiner (CONTEXTJ) dicek.
var registrationProfile = idna.Registration

// mapProfile: hanya pemetaan UTS46 untuk input pengguna di sisi registrar
// (huruf kecil, lebar penuh → normal, NFC, titik ideografik → '.', A-label
// didekode). Validasi sengaja dimatikan; alasan penolakan yang tepat datang
// dari prepareLabel.
var mapProfile = idna.New(idna.MapForLookup(), idna.ValidateLabels(false), idna.Transitional(false))

// preparedLabel adalah label yang sudah lolos pemeriksaan IDNA2008,
// tersedia dalam kedua bentuk.
type preparedLabel struct {
	input  string // label apa adanya dari pemanggil
	ulabel string // bentuk Unicode (sama dengan alabel untuk label LDH)
	alabel string // bentuk ASCII; "xn--..." untuk IDN
	isIDN  bool   // true jika mengandung code point non-ASCII
}

// prepareLabel memvalidasi satu label terhadap IDNA2008 dan mengembalikan
// bentuk U-label dan A-label-nya. Input boleh U-label maupun A-label.
// Huruf ASCII A–Z diturunkan ke huruf kecil (DNS tidak membedakan kapital
// ASCII); huruf besar non-ASCII ditolak, panggil IdnUtils.NormalizeDomainName
// dulu untuk input pengguna.
func prepareLabel(label string) (preparedLabel, error) {
	if label == "" {
		return preparedLabel{}, newLabelError(ErrEmptyLabel, label)
	}
	lower := asciiLower(label)
	if !isASCII(lower) {
		return prepareULabel(label, lower)
	}
	if strings.HasPrefix(lower, acePrefix) {
		return prepareALabel(label, lower)
	}
	return prepareLDH(label, lower)
}

// prepareLDH memvalidasi label ASCII biasa (letter-digit-hyphen).
func prepareLDH(input, l string) (preparedLabel, error) {
	for i, r := range l {
		if !isLDH(r) {
			return preparedLabel{}, newRuneError(ErrInvalidLDH, l, i, r)
		}
	}
	if err := checkHyphens(l, []rune(l)); err != nil {
		return preparedLabel{}, err
	}
	if len(l) > maxLabelOctets {
		return preparedLabel{}, newLabelError(ErrLabelTooLong, l)
	}
	return preparedLabel{input: input, ulabel: l, alabel: l}, nil
}

// prepareALabel mendekode A-label lalu memvalidasi hasilnya sebagai U-label.
// A-label hanya sah bila kanonik: hasil encode ulang harus sama persis
// (RFC 5891 §5.4), sehingga satu nama tidak punya dua ejaan ASCII.
func prepareALabel(input, a string) (preparedLabel, error) {
	if len(a) > maxLabelOctets {
		return preparedLabel{}, newLabelError(ErrLabelTooLong, a)
	}
	u, err := idna.Punycode.ToUnicode(a)
	if err != nil {
		return preparedLabel{}, &ValidationError{Kind: ErrInvalidALabel, Label: a, Pos: -1, Err: err}
	}
	if u == "" || isASCII(u) {
		return preparedLabel{}, &ValidationError{Kind: ErrInvalidALabel, Label: a, Pos: -1, Rule: "decodes to an ASCII-only label"}
	}
	p, err := prepareULabel(input, u)
	if err != nil {
		return preparedLabel{}, err
	}
	if p.alabel != a {
		return preparedLabel{}, &ValidationError{Kind: ErrInvalidALabel, Label: a, Pos: -1, Rule: "non-canonical encoding of " + p.alabel}
	}
	return p, nil
}

// prepareULabel memvalidasi label yang mengandung code point non-ASCII.
func prepareULabel(input, u string) (preparedLabel, error) {
	runes := []rune(u)
	if err := checkHyphens(u, runes); err != nil {
		return preparedLabel{}, err
	}
	if !norm.NFC.IsNormalString(u) {
		return preparedLabel{}, newLabelError(ErrNotNFC, u)
	}
	if unicode.In(runes[0], unicode.Mn, unicode.Mc, unicode.Me) {
		return preparedLabel{}, newRuneError(ErrLeadingCombiningMark, u, 0, runes[0])
	}
	for i, r := range runes {
		if !isIDNA2008Allowed(r) {
			return preparedLabel{}, newRuneError(ErrDisallowedCodepoint, u, i, r)
		}
	}
	if err := checkContextO(u, runes); err != nil {
		return preparedLabel{}, err
	}
	if bidirule.DirectionString(u) != bidi.LeftToRight && !bidirule.ValidString(u) {
		return preparedLabel{}, &ValidationError{Kind: ErrBidiRule, Label: u, Pos: -1, Rule: "RFC 5893 §2"}
	}

	a, err := idna.Punycode.ToASCII(u)
	if err != nil {
		return preparedLabel{}, &ValidationError{Kind: ErrInvalidALabel, Label: u, Pos: -1, Err: err}
	}
	if len(a) > maxLabelOctets {
		return preparedLabel{}, newLabelError(ErrLabelTooLong, u)
	}

	// Jaring pengaman: aturan UTS46/IDNA2008 yang tidak dicek di atas
	// (CONTEXTJ untuk ZWJ/ZWNJ, karakter yang di-mapping atau diabaikan).
	if _, err := registrationProfile.ToASCII(u); err != nil {
		kind, rule := ErrDisallowedCodepoint, ""
		if strings.ContainsAny(u, "‌‍") {
			kind, rule = ErrContextRule, "RFC 5892 A.1/A.2 (CONTEXTJ)"
		}
		return preparedLabel{}, &ValidationError{Kind: kind, Label: u, Pos: -1, Rule: rule, Err: err}
	}

	return preparedLabel{input: input, ulabel: u, alabel: a, isIDN: true}, nil
}

// checkHyphens menerapkan RFC 5891 §4.2.3.1. Posisi dihitung dalam code
// point, bukan byte — x/net/idna menghitung byte sehingga aturan posisi
// 3–4 tidak pernah berlaku untuk aksara multi-byte. A-label ("xn--") tidak
// pernah sampai ke sini; label itu didekode dulu oleh prepareALabel.
func checkHyphens(label string, runes []rune) error {
	n := len(runes)
	if runes[0] == '-' {
		return newRuneError(ErrLeadingHyphen, label, 0, '-')
	}
	if runes[n-1] == '-' {
		return newRuneError(ErrTrailingHyphen, label, n-1, '-')
	}
	if n >= 4 && runes[2] == '-' && runes[3] == '-' {
		return newRuneError(ErrHyphen34, label, 2, '-')
	}
	return nil
}

// Pengecualian RFC 5892 §2.6 (F.1).
var (
	idna2008PvalidExceptions = map[rune]bool{
		0x00DF: true, 0x03C2: true, 0x06FD: true, 0x06FE: true, 0x0F0B: true, 0x3007: true,
	}
	idna2008DisallowedExceptions = map[rune]bool{
		0x0640: true, 0x07FA: true, 0x302E: true, 0x302F: true,
		0x3031: true, 0x3032: true, 0x3033: true, 0x3034: true, 0x3035: true, 0x303B: true,
	}
)

// isIDNA2008Allowed mendekati properti turunan RFC 5892 §3: PVALID,
// CONTEXTO, atau CONTEXTJ. IgnorableProperties dan Unassigned yang tidak
// tertangkap di sini ditolak oleh registrationProfile. Diuji terhadap
// x/text/secure/precis di TestIDNA2008MatchesPrecis.
func isIDNA2008Allowed(r rune) bool {
	switch {
	case r < 0x80:
		return isLDH(r)
	case idna2008PvalidExceptions[r], isContextO(r), r == 0x200C, r == 0x200D:
		return true
	case idna2008DisallowedExceptions[r]:
		return false
	case r >= 0x1100 && r <= 0x11FF, r >= 0xA960 && r <= 0xA97F, r >= 0xD7B0 && r <= 0xD7FF:
		return false // OldHangulJamo (§2.9)
	case r >= 0x20D0 && r <= 0x20FF, r >= 0x1D100 && r <= 0x1D24F:
		return false // IgnorableBlocks (§2.4)
	case !isStable(r):
		return false // Unstable (§2.2), mis. huruf besar Latin
	}
	// LetterDigits (§2.1). Lu ikut: huruf besar yang stabil (Cherokee
	// U+13A0–13F5, karena case folding Cherokee menuju huruf besar) PVALID.
	return unicode.In(r, unicode.Ll, unicode.Lu, unicode.Lo, unicode.Lm, unicode.Mn, unicode.Mc, unicode.Nd)
}

// isStable: NFKC_Casefold(r) == r (RFC 5892 §2.2), lewat pemetaan UTS46
// yang diturunkan dari NFKC_Casefold: code point yang dipetakan atau
// dibuang tidak stabil. (x/text/cases.Fold tidak bisa dipakai: untuk
// Cherokee hasilnya bolak-balik U+13A0 ↔ U+AB70.)
func isStable(r rune) bool {
	s := string(r)
	mapped, _ := mapProfile.ToUnicode(s)
	return mapped == s
}

// isContextO melaporkan code point berkategori CONTEXTO (RFC 5892 F.1).
func isContextO(r rune) bool {
	switch {
	case r == 0x00B7, r == 0x0375, r == 0x05F3, r == 0x05F4, r == 0x30FB:
		return true
	case r >= 0x0660 && r <= 0x0669, r >= 0x06F0 && r <= 0x06F9:
		return true
	}
	return false
}

// checkContextO menerapkan aturan CONTEXTO RFC 5892 Appendix A.3–A.9.
func checkContextO(label string, runes []rune) error {
	n := len(runes)
	var hasArabicIndic, hasExtArabicIndic, hasJapanese bool
	for _, r := range runes {
		switch {
		case r >= 0x0660 && r <= 0x0669:
			hasArabicIndic = true
		case r >= 0x06F0 && r <= 0x06F9:
			hasExtArabicIndic = true
		case r != 0x30FB && unicode.In(r, unicode.Hiragana, unicode.Katakana, unicode.Han):
			hasJapanese = true
		}
	}

	for i, r := range runes {
		var rule string
		switch {
		case r == 0x00B7:
			if i == 0 || i == n-1 || runes[i-1] != 'l' || runes[i+1] != 'l' {
				rule = "RFC 5892 A.3 MIDDLE DOT"
			}
		case r == 0x0375:
			if i == n-1 || !unicode.Is(unicode.Greek, runes[i+1]) {
				rule = "RFC 5892 A.4 GREEK LOWER NUMERAL SIGN"
			}
		case r == 0x05F3 || r == 0x05F4:
			if i == 0 || !unicode.Is(unicode.Hebrew, runes[i-1]) {
				rule = "RFC 5892 A.5/A.6 HEBREW GERESH/GERSHAYIM"
			}
		case r == 0x30FB:
			if !hasJapanese {
				rule = "RFC 5892 A.7 KATAKANA MIDDLE DOT"
			}
		case r >= 0x0660 && r <= 0x0669:
			if hasExtArabicIndic {
				rule = "RFC 5892 A.8 ARABIC-INDIC DIGITS"
			}
		case r >= 0x06F0 && r <= 0x06F9:
			if hasArabicIndic {
				rule = "RFC 5892 A.9 EXTENDED ARABIC-INDIC DIGITS"
			}
		}
		if rule != "" {
			e := newRuneError(ErrContextRule, label, i, r)
			e.Rule = rule
			return e
		}
	}
	return nil
}

// prepareDomain memvalidasi setiap label nama domain dan panjang total.
// Bila salah satu label RTL, aturan bidi berlaku untuk semua label
// (RFC 5893 §1.4 "Bidi domain name").
func prepareDomain(name string) ([]preparedLabel, error) {
	name = strings.TrimSuffix(name, ".")
	if name == "" {
		return nil, newLabelError(ErrEmptyLabel, name)
	}
	parts := strings.Split(name, ".")
	labels := make([]preparedLabel, 0, len(parts))
	octets := len(parts) - 1
	bidiDomain := false
	for _, part := range parts {
		p, err := prepareLabel(part)
		if err != nil {
			return nil, err
		}
		octets += len(p.alabel)
		bidiDomain = bidiDomain || bidirule.DirectionString(p.ulabel) != bidi.LeftToRight
		labels = append(labels, p)
	}
	if octets > maxNameOctets {
		return nil, newLabelError(ErrDomainTooLong, name)
	}
	if bidiDomain {
		for _, p := range labels {
			if !bidirule.ValidString(p.ulabel) {
				return nil, &ValidationError{Kind: ErrBidiRule, Label: p.ulabel, Pos: -1, Rule: "RFC 5893 §2 (bidi domain name)"}
			}
		}
	}
	return labels, nil
}

func isLDH(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-'
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// asciiLower menurunkan hanya A–Z; code point non-ASCII tidak disentuh.
func asciiLower(s string) string {
	for i := 0; i < len(s); i++ {
		if c := s[i]; c >= 'A' && c <= 'Z' {
			b := []byte(s)
			for j := i; j < len(b); j++ {
				if b[j] >= 'A' && b[j] <= 'Z' {
					b[j] += 'a' - 'A'
				}
			}
			return string(b)
		}
	}
	return s
}
