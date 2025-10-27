// Package idn provides IDN (Internationalized Domain Names) validation functionality.
// This package validates whether a given IDN label can be provisioned for a particular TLD.
//
// Copyright 2017 The Nomulus Authors. All Rights Reserved.
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

	// validCodepoints berisi set range dari codepoint yang valid dalam tabel ini
	validCodepoints map[int]bool

	// languageValidator memvalidasi aturan bahasa yang terkait dengan tabel IDN ini
	languageValidator LanguageValidator
}

// NewIdnTable membuat instance baru IdnTable dari data file teks
// Parameter:
// - language: nama bahasa/tabel
// - data: slice string yang berisi baris-baris dari file tabel IDN
// - validator: validator bahasa opsional untuk validasi khusus bahasa
func NewIdnTable(language string, data []string, validator LanguageValidator) (*IdnTable, error) {
	table := &IdnTable{
		name:              language,
		validCodepoints:   make(map[int]bool),
		languageValidator: validator,
	}

	linePattern := regexp.MustCompile(`^U\+([0-9a-fA-F]{4,6})`)
	const urlLinePrefix = "# URL: "
	const policyLinePrefix = "# Policy: "

	for _, line := range data {
		line = strings.TrimSpace(line)

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
	}

	// Set default URLs jika tidak ditemukan dalam file
	if table.url == nil {
		defaultURL := fmt.Sprintf("https://www.iana.org/domains/idn-tables/%s", language)
		table.url, _ = url.Parse(defaultURL)
	}
	if table.policy == nil {
		defaultPolicy := fmt.Sprintf("https://www.iana.org/domains/idn-tables/%s-policy", language)
		table.policy, _ = url.Parse(defaultPolicy)
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

// GetURL mengembalikan URL publik tabel IDN
func (t *IdnTable) GetURL() *url.URL {
	return t.url
}

// GetPolicy mengembalikan URL kebijakan tabel IDN
func (t *IdnTable) GetPolicy() *url.URL {
	return t.policy
}

// IsValidLabel mengembalikan true jika label yang diberikan valid untuk tabel IDN ini.
// Label dianggap valid jika semua codepoint-nya ada dalam tabel IDN.
func (t *IdnTable) IsValidLabel(label string) bool {
	// Iterasi melalui setiap codepoint dalam label
	for _, r := range label {
		if !t.validCodepoints[int(r)] {
			return false
		}
	}

	// Jika ada validator bahasa, gunakan untuk validasi tambahan
	if t.languageValidator != nil {
		return t.languageValidator.IsValidLabelForLanguage(label)
	}

	return true
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
