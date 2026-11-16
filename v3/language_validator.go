package v3

import (
	"strings"
)

// LanguageValidator adalah interface untuk memvalidasi aturan konteks bahasa tertentu
type LanguageValidator interface {
	// IsValidLabelForLanguage mengembalikan true jika label memenuhi aturan konteks untuk bahasa ini
	IsValidLabelForLanguage(label string) bool
}

// LanguageRuleChecker adalah interface opsional untuk LanguageValidator yang
// bisa menjelaskan alasan penolakan (rule dan posisi code point) lewat
// *ValidationError. Validator yang tidak mengimplementasikannya tetap
// dipakai; penolakannya dilaporkan sebagai ErrLanguageRule tanpa detail.
type LanguageRuleChecker interface {
	CheckLabelForLanguage(label string) error
}

// languageValidators adalah registry dari semua validator bahasa yang dikenal, diindeks berdasarkan kode bahasa.
//
// "ja" hanya untuk tabel legacy v3/japanese.txt (seperti di Nomulus).
// idn/Japanese-IDN.txt v2.0 sengaja tidak memakainya: tabel itu sudah tidak
// memuat U+30FB/U+30FC dan header-nya hanya menyebut aturan tanda hubung.
var languageValidators = map[string]LanguageValidator{
	"ja": &JapaneseLanguageValidator{},
}

// scriptValidators diindeks berdasarkan header "# Script:" tabel (huruf
// kecil), sehingga validator ikut terpasang apa pun nama file tabelnya.
var scriptValidators = map[string]LanguageValidator{
	"arab":     &ArabicLanguageValidator{},
	"und-bali": &BalineseLanguageValidator{},
}

// GetLanguageValidator mengembalikan validator bahasa untuk kode bahasa yang diberikan (jika ada)
func GetLanguageValidator(language string) LanguageValidator {
	return languageValidators[language]
}

// validatorForScript mengembalikan validator untuk tag script tabel (jika ada).
func validatorForScript(script string) LanguageValidator {
	return scriptValidators[strings.ToLower(strings.TrimSpace(script))]
}

// checkLanguageRules menjalankan validator dan mengubah hasilnya menjadi
// *ValidationError yang menyebut tabel.
func checkLanguageRules(v LanguageValidator, table, label string) error {
	if checker, ok := v.(LanguageRuleChecker); ok {
		err := checker.CheckLabelForLanguage(label)
		if ve, ok := err.(*ValidationError); ok {
			ve.Table = table
		}
		return err
	}
	if !v.IsValidLabelForLanguage(label) {
		e := newLabelError(ErrLanguageRule, label)
		e.Table = table
		return e
	}
	return nil
}

// JapaneseLanguageValidator memvalidasi label domain bahasa Jepang.
// Kelas ini hanya boleh digunakan dengan tabel IDN bahasa Jepang.
type JapaneseLanguageValidator struct{}

const (
	// Setiap string dengan karakter Jepang dapat memiliki maksimal 15 karakter
	maxLengthJapaneseString = 15

	// Sama dengan codepoint untuk karakter '〆'
	ideographicClosingMark = 0x3006

	// Sama dengan codepoint untuk karakter '・'
	katakanaNiddleDot = 0x30FB

	// Sama dengan codepoint untuk karakter 'ー'
	katakanaHiraganaProlongedSoundMark = 0x30FC
)

// japaneseBlock adalah blok Unicode (bukan script) yang dianggap Jepang,
// sama seperti Character.UnicodeBlock di JapaneseLanguageValidator Nomulus.
// Blok berbeda dari script: U+30FB dan U+30FC ada di blok Katakana, tetapi
// script-nya Common — memakai unicode.Katakana membuat "カーー" ditolak,
// padahal Nomulus menerimanya.
type japaneseBlock int

const (
	blockOther      japaneseBlock = iota
	blockCJKSymbols               // U+3000–303F CJK Symbols and Punctuation
	blockHiragana                 // U+3040–309F
	blockKatakana                 // U+30A0–30FF
	blockCJKUnified               // U+4E00–9FFF CJK Unified Ideographs
)

// japaneseBlockOf mengembalikan blok Unicode Jepang untuk rune, atau blockOther
func japaneseBlockOf(r rune) japaneseBlock {
	switch {
	case r >= 0x3000 && r <= 0x303F:
		return blockCJKSymbols
	case r >= 0x3040 && r <= 0x309F:
		return blockHiragana
	case r >= 0x30A0 && r <= 0x30FF:
		return blockKatakana
	case r >= 0x4E00 && r <= 0x9FFF:
		return blockCJKUnified
	}
	return blockOther
}

// Codepoint yang secara teknis dianggap dalam bahasa Jepang, tetapi merupakan "pengecualian"
// karena tidak dapat muncul dalam label dengan KATAKANA MIDDLE DOT atau IDEOGRAPHIC_CLOSING_MARK
// kecuali codepoint Jepang non-pengecualian lainnya juga ada.
var japaneseExceptionCodepoints = map[rune]bool{
	ideographicClosingMark:             true,
	katakanaNiddleDot:                  true,
	katakanaHiraganaProlongedSoundMark: true,
}

// IsValidLabelForLanguage mengimplementasikan interface LanguageValidator untuk bahasa Jepang
func (j *JapaneseLanguageValidator) IsValidLabelForLanguage(label string) bool {
	requiresJapaneseNonExceptionCodepoint := false
	hasJapaneseCodepoint := false
	hasJapaneseNonExceptionCodepoint := false

	runes := []rune(label)
	precedingUnicodeBlock := blockOther

	for i, r := range runes {
		unicodeBlock := japaneseBlockOf(r)
		isException := japaneseExceptionCodepoints[r]
		isJapanese := unicodeBlock != blockOther

		// Label yang berisi KATAKANA_MIDDLE_DOT atau IDEOGRAPHIC_CLOSING_MARK memerlukan
		// codepoint bahasa Jepang juga muncul dalam label
		if r == katakanaNiddleDot || r == ideographicClosingMark {
			requiresJapaneseNonExceptionCodepoint = true
		}

		// KATAKANA_HIRAGANA_PROLONGED_SOUND_MARK hanya dapat muncul setelah karakter HIRAGANA atau KATAKANA
		if r == katakanaHiraganaProlongedSoundMark {
			if i == 0 || (precedingUnicodeBlock != blockHiragana && precedingUnicodeBlock != blockKatakana) {
				return false
			}
		}

		// Jika codepoint adalah Jepang tetapi bukan codepoint "pengecualian",
		// maka itu harus berupa codepoint Jepang non-pengecualian
		if isJapanese && !isException {
			hasJapaneseNonExceptionCodepoint = true
		}

		// Catat jika kita telah melihat codepoint Jepang apa pun
		if isJapanese {
			hasJapaneseCodepoint = true
		}

		precedingUnicodeBlock = unicodeBlock
	}

	// Label dengan codepoint KATAKANA MIDDLE DOT atau IDEOGRAPHIC_CLOSING_MARK
	// juga harus memiliki beberapa karakter Jepang dalam label.
	// Karakter "pengecualian" Jepang tidak dihitung dalam hal ini.
	if requiresJapaneseNonExceptionCodepoint && !hasJapaneseNonExceptionCodepoint {
		return false
	}

	// Setiap label dengan karakter Jepang (termasuk karakter "pengecualian")
	// hanya boleh sepanjang 15 codepoint
	if hasJapaneseCodepoint && len(runes) > maxLengthJapaneseString {
		return false
	}

	return true
}
