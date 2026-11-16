package v3

import (
	"errors"
	"fmt"
	"strings"
)

// Jenis-jenis kesalahan validasi. Gunakan errors.Is(err, ErrXxx) untuk
// membedakannya — misalnya registry memetakannya ke result code EPP, dan
// registrar menerjemahkannya ke pesan untuk pengguna.
var (
	// Struktur label (RFC 1035, RFC 5891 §4.2.3.1)
	ErrEmptyLabel     = errors.New("empty label")
	ErrLabelTooLong   = errors.New("label exceeds 63 octets in A-label form")
	ErrDomainTooLong  = errors.New("domain name exceeds 253 octets in A-label form")
	ErrLeadingHyphen  = errors.New("label starts with a hyphen")
	ErrTrailingHyphen = errors.New("label ends with a hyphen")
	ErrHyphen34       = errors.New("hyphens in the 3rd and 4th positions are reserved for the xn-- prefix")
	ErrInvalidLDH     = errors.New("ASCII label may only contain a-z, 0-9 and hyphen")
	ErrInvalidALabel  = errors.New("invalid A-label (punycode)")

	// IDNA2008 (RFC 5891, 5892, 5893)
	ErrNotNFC               = errors.New("label is not in Unicode NFC")
	ErrLeadingCombiningMark = errors.New("label starts with a combining mark")
	ErrDisallowedCodepoint  = errors.New("code point is not PVALID under IDNA2008")
	ErrContextRule          = errors.New("code point violates an IDNA2008 contextual rule")
	ErrBidiRule             = errors.New("label violates the IDNA2008 bidi rule")

	// Kebijakan registry (tabel IDN per TLD)
	ErrCodepointNotInTable = errors.New("code point is not in the IDN table")
	ErrLanguageRule        = errors.New("label violates the IDN table's language rules")
	ErrNoIdnTable          = errors.New("no IDN table of this TLD accepts the label")

	// Konfigurasi TLD / bentuk nama domain
	ErrUnknownTld     = errors.New("TLD is not registered")
	ErrTldDisabled    = errors.New("IDN is disabled for this TLD")
	ErrNotRegistrable = errors.New("domain must be exactly one label under a registered zone")
)

// ValidationError menjelaskan kenapa sebuah label ditolak: jenis kesalahan
// (Kind), di code point mana, dan oleh tabel/aturan apa. Registrar dapat
// menampilkannya apa adanya; registry dapat mencatat Table dan Rule.
type ValidationError struct {
	Kind  error  // salah satu Err* di atas
	Label string // label yang divalidasi (U-label jika sudah didekode)
	Pos   int    // indeks code point (mulai 0); -1 jika tidak relevan
	Rune  rune   // code point bermasalah; 0 jika tidak relevan
	Table string // nama tabel IDN, jika kesalahan berasal dari tabel
	Rule  string // nama aturan (mis. "follows-c", "RFC 5892 A.7")
	Err   error  // penyebab dari library lain (mis. x/net/idna), opsional
}

func (e *ValidationError) Error() string {
	var b strings.Builder
	b.WriteString("idn: ")
	if e.Label != "" {
		fmt.Fprintf(&b, "label %q: ", e.Label)
	}
	b.WriteString(e.Kind.Error())
	if e.Rune != 0 {
		fmt.Fprintf(&b, ": %U %q at position %d", e.Rune, e.Rune, e.Pos)
	}
	var extra []string
	if e.Table != "" {
		extra = append(extra, "table "+e.Table)
	}
	if e.Rule != "" {
		extra = append(extra, "rule "+e.Rule)
	}
	if len(extra) > 0 {
		b.WriteString(" (" + strings.Join(extra, ", ") + ")")
	}
	if e.Err != nil {
		b.WriteString(": " + e.Err.Error())
	}
	return b.String()
}

// Unwrap membuat errors.Is bekerja untuk Kind maupun Err.
func (e *ValidationError) Unwrap() []error {
	if e.Err != nil {
		return []error{e.Kind, e.Err}
	}
	return []error{e.Kind}
}

// newLabelError membuat ValidationError tanpa posisi code point.
func newLabelError(kind error, label string) *ValidationError {
	return &ValidationError{Kind: kind, Label: label, Pos: -1}
}

// newRuneError membuat ValidationError yang menunjuk code point ke-pos.
func newRuneError(kind error, label string, pos int, r rune) *ValidationError {
	return &ValidationError{Kind: kind, Label: label, Pos: pos, Rune: r}
}
