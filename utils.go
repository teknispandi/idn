package idn

import (
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/net/idna"
)

// IdnUtils menyediakan utility functions untuk operasi IDN
type IdnUtils struct{}

// ToUnicode mengkonversi label IDN yang dikodekan ASCII (Punycode) ke representasi Unicode
// Fungsi ini setara dengan Idn.toUnicode() di Java
func (u *IdnUtils) ToUnicode(label string) (string, error) {
	// Menggunakan golang.org/x/net/idna untuk konversi Punycode
	result, err := idna.ToUnicode(label)
	if err != nil {
		return "", fmt.Errorf("failed to convert to Unicode: %w", err)
	}
	return result, nil
}

// ToASCII mengkonversi label Unicode ke representasi ASCII (Punycode)
// Fungsi ini setara dengan Idn.toASCII() di Java
func (u *IdnUtils) ToASCII(label string) (string, error) {
	result, err := idna.ToASCII(label)
	if err != nil {
		return "", fmt.Errorf("failed to convert to ASCII: %w", err)
	}
	return result, nil
}

// IsValidDomainLabel memeriksa apakah label adalah domain label yang valid
// sesuai dengan spesifikasi DNS dan IDN
func (u *IdnUtils) IsValidDomainLabel(label string) bool {
	if len(label) == 0 || len(label) > 63 {
		return false
	}

	// Label tidak boleh dimulai atau diakhiri dengan hyphen
	if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
		return false
	}

	// Semua karakter harus valid
	for _, r := range label {
		if !isValidDomainCharacter(r) {
			return false
		}
	}

	return true
}

// isValidDomainCharacter memeriksa apakah karakter valid untuk domain label
func isValidDomainCharacter(r rune) bool {
	// ASCII letters, digits, dan hyphen
	if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') || r == '-' {
		return true
	}

	// Non-ASCII characters (untuk IDN)
	if r >= 0x80 {
		return true
	}

	return false
}

// NormalizeDomainLabel menormalkan label domain untuk perbandingan konsisten
func (u *IdnUtils) NormalizeDomainLabel(label string) string {
	// Konversi ke lowercase (ASCII case folding)
	result := strings.ToLower(label)

	// Untuk karakter non-ASCII, kita bisa menambahkan normalisasi Unicode
	// jika diperlukan (NFC, NFD, dll.)

	return result
}

// SplitDomainName memisahkan nama domain menjadi label-label individual
func (u *IdnUtils) SplitDomainName(domain string) []string {
	if domain == "" {
		return []string{}
	}

	// Hapus trailing dot jika ada
	domain = strings.TrimSuffix(domain, ".")

	return strings.Split(domain, ".")
}

// ValidateDomainName memvalidasi seluruh nama domain
func (u *IdnUtils) ValidateDomainName(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain name cannot be empty")
	}

	labels := u.SplitDomainName(domain)

	// Domain name maksimal 253 karakter
	if len(domain) > 253 {
		return fmt.Errorf("domain name too long: %d characters (max 253)", len(domain))
	}

	// Minimal harus ada satu label
	if len(labels) == 0 {
		return fmt.Errorf("domain name must have at least one label")
	}

	// Validasi setiap label
	for i, label := range labels {
		if !u.IsValidDomainLabel(label) {
			return fmt.Errorf("invalid label at position %d: %s", i, label)
		}
	}

	return nil
}

// GetUnicodeBlocks mengembalikan daftar Unicode blocks yang terdapat dalam string
func (u *IdnUtils) GetUnicodeBlocks(s string) map[string]bool {
	blocks := make(map[string]bool)

	for _, r := range s {
		blockName := getUnicodeBlockName(r)
		if blockName != "" {
			blocks[blockName] = true
		}
	}

	return blocks
}

// getUnicodeBlockName mengembalikan nama Unicode block untuk rune tertentu
func getUnicodeBlockName(r rune) string {
	// Implementasi sederhana untuk beberapa block utama
	if unicode.In(r, unicode.Latin) {
		return "Latin"
	}
	if unicode.In(r, unicode.Hiragana) {
		return "Hiragana"
	}
	if unicode.In(r, unicode.Katakana) {
		return "Katakana"
	}
	if unicode.In(r, unicode.Han) {
		return "CJK_Unified_Ideographs"
	}
	if r >= 0x3000 && r <= 0x303F {
		return "CJK_Symbols_and_Punctuation"
	}
	if unicode.In(r, unicode.Arabic) {
		return "Arabic"
	}
	if unicode.In(r, unicode.Cyrillic) {
		return "Cyrillic"
	}

	return "Other"
}

// CountCodepoints menghitung jumlah codepoint dalam string
// Berbeda dari len() yang menghitung byte
func (u *IdnUtils) CountCodepoints(s string) int {
	return len([]rune(s))
}

// HasMixedScripts memeriksa apakah string mengandung campuran script yang berbeda
func (u *IdnUtils) HasMixedScripts(s string) bool {
	blocks := u.GetUnicodeBlocks(s)

	// Jika hanya ada satu block atau tidak ada block, tidak mixed
	if len(blocks) <= 1 {
		return false
	}

	// Latin + digits selalu diizinkan
	if len(blocks) == 2 && blocks["Latin"] && blocks["Other"] {
		return false
	}

	return true
}

// IsConfusable memeriksa apakah string berpotensi membingungkan
// (implementasi sederhana - dalam produksi perlu database confusables Unicode)
func (u *IdnUtils) IsConfusable(s string) bool {
	confusableChars := map[rune]bool{
		0x00EF: true, // LATIN SMALL LETTER I WITH DIAERESIS (confusable with i)
		0x043E: true, // CYRILLIC SMALL LETTER O (confusable with o)
		0x0430: true, // CYRILLIC SMALL LETTER A (confusable with a)
		// Tambah karakter confusable lainnya sesuai kebutuhan
	}

	for _, r := range s {
		if confusableChars[r] {
			return true
		}
	}

	return false
}

// DefaultIdnUtils adalah instance global dari IdnUtils untuk kemudahan penggunaan
var DefaultIdnUtils = &IdnUtils{}
