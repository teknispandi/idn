package v3

import (
	"fmt"
	"strings"
	"unicode"
)

// IdnUtils menyediakan utility functions untuk operasi IDN
type IdnUtils struct{}

// ToUnicode mengkonversi nama domain (atau label) yang dikodekan ASCII (Punycode) ke representasi Unicode
// Fungsi ini setara dengan Idn.toUnicode() di Java. Setiap label divalidasi
// menurut IDNA2008; A-label yang tidak kanonik atau tidak valid ditolak.
func (u *IdnUtils) ToUnicode(label string) (string, error) {
	result, err := convertDomain(label, func(p preparedLabel) string { return p.ulabel })
	if err != nil {
		return "", fmt.Errorf("failed to convert to Unicode: %w", err)
	}
	return result, nil
}

// ToASCII mengkonversi nama domain (atau label) Unicode ke representasi ASCII (Punycode)
// Fungsi ini setara dengan Idn.toASCII() di Java. Setiap label divalidasi
// menurut IDNA2008 tanpa pemetaan: huruf besar non-ASCII ditolak (gunakan
// NormalizeDomainName dulu untuk input pengguna).
func (u *IdnUtils) ToASCII(label string) (string, error) {
	result, err := convertDomain(label, func(p preparedLabel) string { return p.alabel })
	if err != nil {
		return "", fmt.Errorf("failed to convert to ASCII: %w", err)
	}
	return result, nil
}

// convertDomain memvalidasi semua label dan menyusunnya kembali dengan
// bentuk yang dipilih pick. Titik di akhir (FQDN) dipertahankan.
func convertDomain(name string, pick func(preparedLabel) string) (string, error) {
	labels, err := prepareDomain(name)
	if err != nil {
		return "", err
	}
	parts := make([]string, len(labels))
	for i, p := range labels {
		parts[i] = pick(p)
	}
	result := strings.Join(parts, ".")
	if strings.HasSuffix(name, ".") {
		result += "."
	}
	return result, nil
}

// IsValidDomainLabel memeriksa apakah label adalah domain label yang valid
// sesuai dengan spesifikasi DNS dan IDNA2008: panjang A-label maksimal 63
// oktet, aturan tanda hubung, A-label kanonik, dan code point PVALID.
// Tabel IDN TLD tidak diperiksa di sini; gunakan IdnManager.CheckLabel.
func (u *IdnUtils) IsValidDomainLabel(label string) bool {
	_, err := prepareLabel(label)
	return err == nil
}

// NormalizeDomainLabel menormalkan label domain untuk perbandingan konsisten:
// pemetaan UTS46 (huruf kecil, lebar penuh → normal, NFC).
func (u *IdnUtils) NormalizeDomainLabel(label string) string {
	result, _ := mapProfile.ToUnicode(label)
	return result
}

// NormalizeDomainName menyiapkan input pengguna di sisi registrar sebelum
// divalidasi: spasi di tepi dibuang, pemetaan UTS46 (huruf kecil, karakter
// lebar penuh, NFC, titik ideografik "。" menjadi "."), A-label didekode,
// dan titik di akhir dibuang. Hasilnya berbentuk Unicode untuk ditampilkan
// ke pengguna; validasi tetap oleh IdnManager.CheckDomain, yang memberi
// alasan penolakan yang tepat.
func (u *IdnUtils) NormalizeDomainName(name string) (string, error) {
	result, _ := mapProfile.ToUnicode(strings.TrimSpace(name))
	result = strings.TrimSuffix(result, ".")
	if result == "" {
		return "", newLabelError(ErrEmptyLabel, name)
	}
	return result, nil
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

// ValidateDomainName memvalidasi seluruh nama domain menurut IDNA2008:
// setiap label (lihat IsValidDomainLabel), panjang total maksimal 253 oktet
// dalam bentuk A-label, dan aturan bidi RFC 5893 untuk nama yang memuat
// label RTL. Error berupa *ValidationError.
func (u *IdnUtils) ValidateDomainName(domain string) error {
	_, err := prepareDomain(domain)
	return err
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

// HasMixedScripts memeriksa apakah string mengandung campuran script yang berbeda.
// Script Common (digit, tanda hubung) dan Inherited (tanda diakritik) tidak
// dihitung, dan kombinasi yang lazim dalam satu bahasa tidak dianggap campuran:
// Han+Hiragana+Katakana (Jepang), Han+Hangul (Korea), Han+Bopomofo.
func (u *IdnUtils) HasMixedScripts(s string) bool {
	scripts := make(map[string]bool)
	for _, r := range s {
		if name := scriptOf(r); name != "" && name != "Common" && name != "Inherited" {
			scripts[name] = true
		}
	}

	// Jika hanya ada satu script atau tidak ada script, tidak mixed
	if len(scripts) <= 1 {
		return false
	}

	for _, allowed := range [][]string{{"Han", "Hiragana", "Katakana"}, {"Han", "Hangul"}, {"Han", "Bopomofo"}} {
		covered := 0
		for _, name := range allowed {
			if scripts[name] {
				covered++
			}
		}
		if covered == len(scripts) {
			return false
		}
	}

	return true
}

// scriptOf mengembalikan nama script Unicode untuk rune, atau "" jika tidak dikenal
func scriptOf(r rune) string {
	for name, table := range unicode.Scripts {
		if unicode.Is(table, r) {
			return name
		}
	}
	return ""
}

// IsConfusable memeriksa apakah string berpotensi membingungkan
// (implementasi sederhana - dalam produksi perlu database confusables Unicode).
// Hanya heuristik untuk tampilan; bukan pengganti kebijakan varian/tabel IDN
// dan tidak dipakai dalam validasi registry.
func (u *IdnUtils) IsConfusable(s string) bool {
	confusableChars := map[rune]bool{
		0x00EF: true, // LATIN SMALL LETTER I WITH DIAERESIS (confusable with i)
		0x043E: true, // CYRILLIC SMALL LETTER O (confusable with o)
		0x0430: true, // CYRILLIC SMALL LETTER A (confusable with a)
		0x0440: true, // CYRILLIC SMALL LETTER ER (confusable with p)
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
