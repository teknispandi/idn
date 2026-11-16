package v3

// ArabicLanguageValidator menerapkan aturan yang dinyatakan di header
// idn/Arabic-IDN.txt: "Registrations using either Arabic-Indic or Arabic
// digits are allowed, but digit types are not allowed to be mixed.
// Homogeneity is enforced by the registry."
//
// Campuran Arabic-Indic (U+0660–0669) dengan Extended Arabic-Indic
// (U+06F0–06F9) sudah ditolak oleh CONTEXTO RFC 5892 A.8/A.9; validator ini
// menambahkan larangan mencampur digit ASCII 0–9 dengan Arabic-Indic.
//
// Pada jalur lengkap (IdnManager.CheckDomain) campuran itu juga ditolak
// aturan bidi RFC 5893 no. 4 (EN dan AN tidak boleh bersama dalam label
// RTL); validator ini memastikan IdnTable.IsValidLabel pun menolaknya.
type ArabicLanguageValidator struct{}

func (a *ArabicLanguageValidator) IsValidLabelForLanguage(label string) bool {
	return a.CheckLabelForLanguage(label) == nil
}

// CheckLabelForLanguage mengembalikan ValidationError pada digit pertama
// yang jenisnya berbeda dari digit sebelumnya dalam label.
func (a *ArabicLanguageValidator) CheckLabelForLanguage(label string) error {
	const (
		none = iota
		european
		arabicIndic
	)
	seen := none
	for i, r := range []rune(label) {
		kind := none
		switch {
		case r >= '0' && r <= '9':
			kind = european
		case r >= 0x0660 && r <= 0x0669:
			kind = arabicIndic
		}
		if kind == none {
			continue
		}
		if seen != none && seen != kind {
			e := newRuneError(ErrLanguageRule, label, i, r)
			e.Rule = "arabic-digit-homogeneity"
			return e
		}
		seen = kind
	}
	return nil
}
