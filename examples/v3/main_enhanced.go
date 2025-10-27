package main

import (
	"fmt"
	"log"
	"path/filepath"

	v3 "github.com/al0fayz/idn/v3"
)

func ExampleEnhanced() {
	fmt.Println("=== IDN Manager Example ===")

	// 1. Inisialisasi IDN Manager
	// Pastikan path menuju folder /idn benar
	idnPath := filepath.Join("..", "..", "idn")
	manager := v3.NewIdnManager(idnPath)

	// 2. Register TLD dengan multiple languages
	fmt.Println("\n--- Registering TLDs ---")

	// TLD untuk bahasa Arab dan Hebrew
	err := manager.RegisterTld("arab", []string{"arabic", "hebrew"})
	if err != nil {
		log.Printf("Error registering arab TLD: %v", err)
	} else {
		fmt.Println("✓ Registered TLD 'arab' with Arabic and Hebrew support")
	}

	// TLD untuk bahasa Asia
	err = manager.RegisterTld("asia", []string{"chinese", "japanese", "korean"})
	if err != nil {
		log.Printf("Error registering asia TLD: %v", err)
	} else {
		fmt.Println("✓ Registered TLD 'asia' with Chinese, Japanese, and Korean support")
	}

	// TLD untuk bahasa India
	err = manager.RegisterTld("india", []string{"devanagari", "bengali", "tamil"})
	if err != nil {
		log.Printf("Error registering india TLD: %v", err)
	} else {
		fmt.Println("✓ Registered TLD 'india' with Devanagari, Bengali, and Tamil support")
	}

	// 3. List available languages
	fmt.Println("\n--- Available Languages ---")
	loader := v3.NewIdnLoader(idnPath)
	languages, err := loader.ListAvailableLanguages()
	if err != nil {
		log.Printf("Error listing languages: %v", err)
	} else {
		fmt.Printf("Available languages: %v\n", languages)
	}

	// 4. Preload beberapa bahasa untuk performa yang lebih baik
	fmt.Println("\n--- Preloading Languages ---")
	err = manager.RegisterTldWithPreload("global", []string{"latin", "cyrillic", "greek"})
	if err != nil {
		log.Printf("Error preloading global TLD: %v", err)
	} else {
		fmt.Println("✓ Preloaded and registered TLD 'global'")
	}

	// 5. Test validasi label
	fmt.Println("\n--- Testing Label Validation ---")

	testCases := []struct {
		tld   string
		label string
	}{
		{"arab", "example"},      // Latin untuk TLD Arab
		{"arab", "مثال"},         // Arabic text
		{"asia", "例え"},           // Japanese text
		{"asia", "例子"},           // Chinese text
		{"global", "example"},    // Latin
		{"global", "пример"},     // Cyrillic
		{"global", "παράδειγμα"}, // Greek
	}

	for _, tc := range testCases {
		isValid, validLanguages, err := manager.ValidateLabel(tc.tld, tc.label)
		if err != nil {
			fmt.Printf("❌ Error validating '%s' for TLD '%s': %v\n", tc.label, tc.tld, err)
			continue
		}

		if isValid {
			fmt.Printf("✓ Label '%s' is valid for TLD '%s' (languages: %v)\n",
				tc.label, tc.tld, validLanguages)
		} else {
			fmt.Printf("❌ Label '%s' is not valid for TLD '%s'\n", tc.label, tc.tld)
		}
	}

	// 6. Add/Remove languages dynamically
	fmt.Println("\n--- Dynamic Language Management ---")

	// Add language to existing TLD
	err = manager.AddLanguageToTld("global", "armenian")
	if err != nil {
		log.Printf("Error adding Armenian to global TLD: %v", err)
	} else {
		fmt.Println("✓ Added Armenian language to global TLD")
	}

	// Test dengan bahasa yang baru ditambahkan
	isValid, err := manager.ValidateLabelForLanguage("global", "armenian", "օրինակ")
	if err != nil {
		fmt.Printf("❌ Error validating Armenian text: %v\n", err)
	} else if isValid {
		fmt.Println("✓ Armenian text validated successfully")
	} else {
		fmt.Println("❌ Armenian text validation failed")
	}

	// 7. Get statistics
	fmt.Println("\n--- Manager Statistics ---")
	stats := manager.GetStats()
	fmt.Printf("Registered TLDs: %d\n", stats["registered_tlds"])

	if cacheStats, ok := stats["cache_stats"].(map[string]interface{}); ok {
		fmt.Printf("Cached languages: %d\n", cacheStats["cached_languages"])
		if languages, ok := cacheStats["languages"].([]string); ok {
			fmt.Printf("Cached language list: %v\n", languages)
		}
	}

	// 8. List all registered TLDs
	fmt.Println("\n--- Registered TLDs ---")
	tlds := manager.ListRegisteredTlds()
	for _, tld := range tlds {
		config, err := manager.GetTldConfig(tld)
		if err != nil {
			continue
		}
		fmt.Printf("TLD: %s, Languages: %v, Enabled: %v\n",
			config.TldName, config.Languages, config.Enabled)
	}

	// 9. Demonstrate enable/disable functionality
	fmt.Println("\n--- Enable/Disable TLD ---")
	err = manager.DisableTld("asia")
	if err != nil {
		log.Printf("Error disabling asia TLD: %v", err)
	} else {
		fmt.Println("✓ Disabled TLD 'asia'")
	}

	err = manager.EnableTld("asia")
	if err != nil {
		log.Printf("Error enabling asia TLD: %v", err)
	} else {
		fmt.Println("✓ Re-enabled TLD 'asia'")
	}

	fmt.Println("\n=== Example completed ===")
}
