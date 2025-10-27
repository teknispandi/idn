package main

import (
	"fmt"
	"log"

	v3 "github.com/al0fayz/idn/v3"
)

// ExampleTld adalah implementasi contoh dari TldInterface
type ExampleTld struct {
	name      string
	idnTables []v3.IdnTableType
}

func (t *ExampleTld) GetIdnTables() []v3.IdnTableType {
	return t.idnTables
}

func ExampleIdn() {
	fmt.Println("=== Contoh Penggunaan IDN Package ===")

	// 1. Inisialisasi validator
	validator := v3.NewIdnLabelValidator()

	// 2. Buat contoh TLD
	// TLD dengan tabel default (ExtendedLatin + Japanese)
	defaultTld := &ExampleTld{
		name:      "example",
		idnTables: []v3.IdnTableType{}, // kosong = gunakan default
	}

	// TLD khusus dengan hanya Extended Latin
	latinOnlyTld := &ExampleTld{
		name:      "latin",
		idnTables: []v3.IdnTableType{v3.ExtendedLatin},
	}

	// TLD khusus dengan hanya Japanese
	japaneseTld := &ExampleTld{
		name:      "jp",
		idnTables: []v3.IdnTableType{v3.Japanese},
	}

	// 3. Test berbagai label
	testLabels := []string{
		"example",        // ASCII sederhana
		"ex-ample",       // dengan hyphen
		"example123",     // dengan angka
		"café",           // dengan accent
		"résumé",         // multiple accents
		"münchen",        // German umlaut
		"こんにちは",          // Japanese hiragana
		"カタカナ",           // Japanese katakana
		"example@domain", // karakter tidak valid
		"",               // string kosong
	}

	fmt.Println("\n--- Testing Default TLD ---")
	testLabelsWithTld(validator, defaultTld, testLabels)

	fmt.Println("\n--- Testing Latin-Only TLD ---")
	testLabelsWithTld(validator, latinOnlyTld, testLabels)

	fmt.Println("\n--- Testing Japanese TLD ---")
	testLabelsWithTld(validator, japaneseTld, testLabels)

	// 4. Demonstrasi utilitas IDN
	fmt.Println("\n--- IDN Utilities Demo ---")
	demonstrateIdnUtils()

	// 5. Demonstrasi informasi tabel
	fmt.Println("\n--- Table Information ---")
	demonstrateTableInfo(validator)
}

func testLabelsWithTld(validator *v3.IdnLabelValidator, tld *ExampleTld, labels []string) {
	fmt.Printf("TLD: %s\n", tld.name)
	fmt.Printf("Supported tables: %v\n", validator.GetIdnTablesForTld(tld))

	for _, label := range labels {
		tableName, isValid := validator.FindValidIdnTableForTld(label, tld)
		status := "❌ INVALID"
		if isValid {
			status = fmt.Sprintf("✅ VALID (table: %s)", tableName)
		}
		fmt.Printf("  %-20s → %s\n", fmt.Sprintf("'%s'", label), status)
	}
}

func demonstrateIdnUtils() {
	utils := v3.DefaultIdnUtils

	testCases := []struct {
		label       string
		description string
	}{
		{"café", "French with accent"},
		{"münchen", "German umlaut"},
		{"xn--caf-dma", "Punycode for café"},
		{"こんにちは", "Japanese hiragana"},
		{"example-123", "Mixed alphanumeric"},
	}

	for _, tc := range testCases {
		fmt.Printf("\nTesting: %s (%s)\n", tc.label, tc.description)

		// Validasi basic
		fmt.Printf("  Valid domain label: %t\n", utils.IsValidDomainLabel(tc.label))

		// Hitung codepoints
		fmt.Printf("  Codepoint count: %d\n", utils.CountCodepoints(tc.label))

		// Unicode blocks
		blocks := utils.GetUnicodeBlocks(tc.label)
		fmt.Printf("  Unicode blocks: %v\n", getMapKeys(blocks))

		// Mixed scripts
		fmt.Printf("  Has mixed scripts: %t\n", utils.HasMixedScripts(tc.label))

		// Confusable
		fmt.Printf("  Is confusable: %t\n", utils.IsConfusable(tc.label))

		// Konversi ASCII/Unicode (jika memungkinkan)
		if ascii, err := utils.ToASCII(tc.label); err == nil && ascii != tc.label {
			fmt.Printf("  ASCII form: %s\n", ascii)
		}

		if unicode, err := utils.ToUnicode(tc.label); err == nil && unicode != tc.label {
			fmt.Printf("  Unicode form: %s\n", unicode)
		}
	}
}

func demonstrateTableInfo(validator *v3.IdnLabelValidator) {
	tables := []v3.IdnTableType{
		v3.ExtendedLatin,
		v3.UnconfusableLatin,
		v3.Japanese,
	}

	for _, tableType := range tables {
		info, err := validator.GetTableInfo(tableType)
		if err != nil {
			log.Printf("Error getting table info for %s: %v", tableType.String(), err)
			continue
		}

		fmt.Printf("\nTable: %s\n", info.Name)
		fmt.Printf("  Type: %s\n", info.Type.String())
		fmt.Printf("  URL: %s\n", info.URL.String())
		fmt.Printf("  Policy: %s\n", info.Policy.String())
	}
}

// Helper function untuk mendapatkan keys dari map[string]bool
func getMapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
