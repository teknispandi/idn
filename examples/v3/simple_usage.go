package main

import (
	"fmt"
	"log"
	"path/filepath"

	v3 "github.com/al0fayz/idn/v3"
)

func SimpleUsage() {
	fmt.Println("=== Simple IDN Usage Example ===")

	// 1. Setup IDN path
	idnPath := filepath.Join("..", "..", "idn")

	// 2. Cara sederhana: gunakan IDN Loader langsung
	fmt.Println("\n--- Using IDN Loader Directly ---")
	loader := v3.NewIdnLoader(idnPath)

	// Load single table
	arabicTable, err := loader.LoadTable("arabic")
	if err != nil {
		log.Printf("Error loading Arabic table: %v", err)
	} else {
		fmt.Println("✓ Arabic table loaded successfully")

		// Test beberapa label
		testLabels := []string{
			"example", // Latin
			"مثال",    // Arabic
			"test123", // Latin dengan angka
		}

		for _, label := range testLabels {
			isValid := arabicTable.IsValidLabel(label)
			fmt.Printf("Label '%s': %v\n", label, isValid)
		}
	}

	// 3. Cara advanced: gunakan IDN Manager
	fmt.Println("\n--- Using IDN Manager ---")
	manager := v3.NewIdnManager(idnPath)

	// Register TLD dengan single language
	err = manager.RegisterTld("example", []string{"latin"})
	if err != nil {
		log.Printf("Error registering TLD: %v", err)
		return
	}

	// Test validation
	testLabels := []string{
		"example",
		"test-domain",
		"café",
		"münchen",
	}

	for _, label := range testLabels {
		isValid, languages, err := manager.ValidateLabel("example", label)
		if err != nil {
			fmt.Printf("Error validating '%s': %v\n", label, err)
			continue
		}

		if isValid {
			fmt.Printf("✓ '%s' is valid (supported by: %v)\n", label, languages)
		} else {
			fmt.Printf("❌ '%s' is not valid\n", label)
		}
	}

	// 4. Cache demo
	fmt.Println("\n--- Cache Statistics ---")
	stats := loader.GetCacheStats()
	fmt.Printf("Cache stats: %+v\n", stats)

	// 5. List available languages
	languages, err := loader.ListAvailableLanguages()
	if err != nil {
		log.Printf("Error listing languages: %v", err)
	} else {
		fmt.Printf("Available languages: %v\n", languages)
	}

	fmt.Println("\n=== Simple example completed ===")
}
