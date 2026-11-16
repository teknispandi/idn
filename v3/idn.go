// Package idn provides IDN (Internationalized Domain Names) validation functionality.
// This package validates whether a given IDN label can be provisioned for a particular TLD.
//
// Copyright 2025 The Akhmadrofai@gmail.com Authors. All Rights Reserved.
// Licensed under the Apache License, Version 2.0
package v3

import (
	"bufio"
	"embed"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

//go:embed *.txt
var tableFiles embed.FS

// IdnTable represents an IDN table for a particular TLD
type IdnTable struct {
	// name adalah nama bahasa untuk tabel ini (sesuai dengan nama file)
	name string

	// url adalah URL publik dari tabel IDN ini, yang diperlukan oleh RDE
	url *url.URL

	// policy adalah URL publik kebijakan untuk tabel IDN ini, yang diperlukan oleh RDE
	policy *url.URL

	// script, version dan effectiveDate diambil dari header "# Script:",
	// "# Version:" dan "# Effective Date:" (kosong jika tidak ada)
	script        string
	version       string
	effectiveDate string

	// validCodepoints berisi set range dari codepoint yang valid dalam tabel ini
	validCodepoints map[int]bool

	// languageValidator memvalidasi aturan bahasa yang terkait dengan tabel IDN ini
	languageValidator LanguageValidator

	// Varian dari kolom 2 dan 3 format RFC 3743 (lihat variants.go); kosong
	// untuk tabel tanpa varian.
	preferred    map[rune][]rune
	variantIndex map[rune]rune   // code point → anggota terkecil himpunannya
	variantSets  map[rune][]rune // anggota terkecil → semua anggota, terurut
}

// NewIdnTable membuat instance baru IdnTable dari data file teks
// Parameter:
//   - language: nama bahasa/tabel
//   - data: slice string yang berisi baris-baris dari file tabel IDN
//   - validator: validator bahasa opsional untuk validasi khusus bahasa. Jika
//     nil, validator dipilih dari header "# Script:" (lihat scriptValidators).
//
// URL dan Policy hanya diisi dari header "# URL:" / "# Policy:". Jika tidak
// ada, GetURL/GetPolicy mengembalikan nil — tambahkan header itu ke file
// tabel sebelum dipakai untuk deposit RDE, jangan mengarang URL.
func NewIdnTable(language string, data []string, validator LanguageValidator) (*IdnTable, error) {
	table := &IdnTable{
		name:              language,
		validCodepoints:   make(map[int]bool),
		languageValidator: validator,
	}

	linePattern := regexp.MustCompile(`^U\+([0-9a-fA-F]{4,6})`)
	const urlLinePrefix = "# URL: "
	const policyLinePrefix = "# Policy: "
	headers := map[string]*string{
		"# Script:":         &table.script,
		"# Version:":        &table.version,
		"# Effective Date:": &table.effectiveDate,
	}

	variantEdges := make(map[rune][]rune)

	for _, line := range data {
		line = strings.TrimSpace(line)

		for prefix, field := range headers {
			if *field == "" && strings.HasPrefix(line, prefix) {
				*field = strings.TrimSpace(strings.TrimPrefix(line, prefix))
			}
		}

		// Tangani baris komentar khusus untuk URL dan Policy
		if strings.HasPrefix(line, urlLinePrefix) {
			var err error
			table.url, err = url.Parse(strings.TrimPrefix(line, urlLinePrefix))
			if err != nil {
				return nil, fmt.Errorf("invalid URL in table %s: %w", language, err)
			}
			continue
		}

		if strings.HasPrefix(line, policyLinePrefix) {
			var err error
			table.policy, err = url.Parse(strings.TrimPrefix(line, policyLinePrefix))
			if err != nil {
				return nil, fmt.Errorf("invalid policy URL in table %s: %w", language, err)
			}
			continue
		}

		// Lewati baris kosong dan komentar
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse codepoint dari baris
		codepoint, err := readCodepoint(line, linePattern)
		if err != nil {
			return nil, fmt.Errorf("error parsing line '%s' in table %s: %w", line, language, err)
		}

		table.validCodepoints[codepoint] = true

		if strings.Contains(line, ";") {
			preferred, variants := parseVariantColumns(line)
			if len(preferred) > 0 {
				if table.preferred == nil {
					table.preferred = make(map[rune][]rune)
				}
				table.preferred[rune(codepoint)] = preferred
			}
			variantEdges[rune(codepoint)] = append(variantEdges[rune(codepoint)], variants...)
		}
	}

	if len(table.validCodepoints) == 0 {
		return nil, fmt.Errorf("IDN table %s has no code points", language)
	}

	if len(variantEdges) > 0 {
		table.buildVariantSets(variantEdges)
	}

	if table.languageValidator == nil {
		table.languageValidator = validatorForScript(table.script)
	}

	return table, nil
}

// readCodepoint membaca codepoint dari satu baris. Mendukung berbagai format:
// - U+XXXX (format standar)
// - U+XXXX(0);U+XXXX(0); (format Chinese)
// - U+XXXX # COMMENT (dengan komentar)
func readCodepoint(line string, pattern *regexp.Regexp) (int, error) {
	// Handle format Chinese dengan semicolon
	if strings.Contains(line, ";") {
		parts := strings.Split(line, ";")
		if len(parts) > 0 {
			line = strings.TrimSpace(parts[0])
		}
	}

	// Remove parentheses jika ada (format Chinese)
	if strings.Contains(line, "(") {
		idx := strings.Index(line, "(")
		line = line[:idx]
	}

	matches := pattern.FindStringSubmatch(line)
	if len(matches) < 2 {
		return 0, fmt.Errorf("can't parse line: %s", line)
	}

	hexString := matches[1]
	codepoint, err := strconv.ParseInt(hexString, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid hex value %s: %w", hexString, err)
	}

	return int(codepoint), nil
}

// GetName mengembalikan nama tabel IDN
func (t *IdnTable) GetName() string {
	return t.name
}

// GetURL mengembalikan URL publik tabel IDN, atau nil jika file tabel tidak
// memiliki header "# URL:"
func (t *IdnTable) GetURL() *url.URL {
	return t.url
}

// GetPolicy mengembalikan URL kebijakan tabel IDN, atau nil jika file tabel
// tidak memiliki header "# Policy:"
func (t *IdnTable) GetPolicy() *url.URL {
	return t.policy
}

// GetScript mengembalikan tag script dari header tabel (mis. "Latn", "und-Bali")
func (t *IdnTable) GetScript() string {
	return t.script
}

// GetVersion mengembalikan versi tabel dari header
func (t *IdnTable) GetVersion() string {
	return t.version
}

// GetEffectiveDate mengembalikan tanggal berlaku tabel dari header
func (t *IdnTable) GetEffectiveDate() string {
	return t.effectiveDate
}

// IsValidLabel mengembalikan true jika label yang diberikan valid untuk tabel IDN ini.
// Label dianggap valid jika tidak kosong, semua codepoint-nya ada dalam tabel IDN,
// dan lolos validator bahasa tabel (jika ada).
//
// Ini hanya pemeriksaan tabel: label harus sudah berupa U-label. Untuk
// validasi lengkap (A-label, IDNA2008, tanda hubung, panjang) gunakan
// IdnManager.CheckLabel / CheckDomain atau IdnLabelValidator.
func (t *IdnTable) IsValidLabel(label string) bool {
	return t.CheckLabel(label) == nil
}

// CheckLabel seperti IsValidLabel, tetapi mengembalikan *ValidationError
// yang menyebut code point atau aturan bahasa yang membuat label ditolak.
func (t *IdnTable) CheckLabel(label string) error {
	if label == "" {
		return newLabelError(ErrEmptyLabel, label)
	}

	// Iterasi melalui setiap codepoint dalam label
	for i, r := range []rune(label) {
		if !t.validCodepoints[int(r)] {
			e := newRuneError(ErrCodepointNotInTable, label, i, r)
			e.Table = t.name
			return e
		}
	}

	// Jika ada validator bahasa, gunakan untuk validasi tambahan
	if t.languageValidator != nil {
		return checkLanguageRules(t.languageValidator, t.name, label)
	}

	return nil
}

// loadTableFromFile memuat tabel IDN dari file yang disematkan
func loadTableFromFile(filename string) ([]string, error) {
	content, err := tableFiles.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}
