package v2

import (
	"unicode"
)

// LanguageValidator adalah interface untuk memvalidasi aturan konteks bahasa tertentu
type LanguageValidator interface {
	// IsValidLabelForLanguage mengembalikan true jika label memenuhi aturan konteks untuk bahasa ini
	IsValidLabelForLanguage(label string) bool
}

// languageValidators adalah registry dari semua validator bahasa yang dikenal, diindeks berdasarkan kode bahasa
var languageValidators = map[string]LanguageValidator{
	"ja": &JapaneseLanguageValidator{},
}

// GetLanguageValidator mengembalikan validator bahasa untuk kode bahasa yang diberikan (jika ada)
func GetLanguageValidator(language string) LanguageValidator {
	return languageValidators[language]
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

// Set dari UnicodeRange yang berisi codepoint Jepang yang valid
var japaneseUnicodeBlocks = map[*unicode.RangeTable]bool{
	unicode.Han:      true, // CJK Unified Ideographs
	unicode.Hiragana: true,
	unicode.Katakana: true,
	// Note: unicode package di Go tidak memiliki CJK_SYMBOLS_AND_PUNCTUATION secara langsung
	// Kita akan menanganinya dengan cek manual untuk range tersebut
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
	var precedingUnicodeBlock *unicode.RangeTable

	for i, r := range runes {
		unicodeBlock := getUnicodeBlock(r)
		isException := japaneseExceptionCodepoints[r]
		isJapanese := isJapaneseCodepoint(r, unicodeBlock)

		// Label yang berisi KATAKANA_MIDDLE_DOT atau IDEOGRAPHIC_CLOSING_MARK memerlukan
		// codepoint bahasa Jepang juga muncul dalam label
		if r == katakanaNiddleDot || r == ideographicClosingMark {
			requiresJapaneseNonExceptionCodepoint = true
		}

		// KATAKANA_HIRAGANA_PROLONGED_SOUND_MARK hanya dapat muncul setelah karakter HIRAGANA atau KATAKANA
		if r == katakanaHiraganaProlongedSoundMark {
			if i == 0 || (precedingUnicodeBlock != unicode.Hiragana && precedingUnicodeBlock != unicode.Katakana) {
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

// getUnicodeBlock mengembalikan blok unicode untuk rune yang diberikan
func getUnicodeBlock(r rune) *unicode.RangeTable {
	if unicode.In(r, unicode.Hiragana) {
		return unicode.Hiragana
	}
	if unicode.In(r, unicode.Katakana) {
		return unicode.Katakana
	}
	if unicode.In(r, unicode.Han) {
		return unicode.Han
	}
	// Cek untuk CJK Symbols and Punctuation (U+3000-U+303F)
	if r >= 0x3000 && r <= 0x303F {
		// Untuk keperluan ini, kita anggap sebagai blok khusus
		// Karena Go tidak memiliki konstanta untuk ini, kita buat sendiri
		return &unicode.RangeTable{
			R16: []unicode.Range16{
				{Lo: 0x3000, Hi: 0x303F, Stride: 1},
			},
		}
	}
	return nil
}

// isJapaneseCodepoint memeriksa apakah rune adalah codepoint Jepang
func isJapaneseCodepoint(r rune, block *unicode.RangeTable) bool {
	// Cek blok unicode Jepang standar
	if unicode.In(r, unicode.Hiragana) || unicode.In(r, unicode.Katakana) || unicode.In(r, unicode.Han) {
		return true
	}

	// Cek CJK Symbols and Punctuation
	if r >= 0x3000 && r <= 0x303F {
		return true
	}

	return false
}
