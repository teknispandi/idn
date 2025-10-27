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

	// 6. IDN Utils Demo
	fmt.Println("\n--- IDN Utils Examples ---")
	utils := &v3.IdnUtils{}

	// Punycode conversion
	fmt.Println("\nPunycode Conversion:")
	unicodeDomains := []string{"例子.com", "مثال.com", "пример.com"}
	for _, domain := range unicodeDomains {
		ascii, err := utils.ToASCII(domain)
		if err != nil {
			fmt.Printf("❌ %s: conversion failed - %v\n", domain, err)
		} else {
			fmt.Printf("✅ %s -> %s\n", domain, ascii)
		}
	}

	// Domain validation
	fmt.Println("\nDomain Validation:")
	testDomains := []string{"example.com", "例子.com", "-invalid.com", "test_underscore.com"}
	for _, domain := range testDomains {
		err := utils.ValidateDomainName(domain)
		if err != nil {
			fmt.Printf("❌ %s: %v\n", domain, err)
		} else {
			fmt.Printf("✅ %s: Valid\n", domain)
		}
	}

	// Unicode analysis
	fmt.Println("\nUnicode Analysis:")
	testStrings := []string{"example", "例子", "test例", "مثال"}
	for _, str := range testStrings {
		codepoints := utils.CountCodepoints(str)
		mixed := utils.HasMixedScripts(str)
		blocks := utils.GetUnicodeBlocks(str)

		var blockNames []string
		for block := range blocks {
			if blocks[block] {
				blockNames = append(blockNames, block)
			}
		}

		fmt.Printf("'%s': %d codepoints, mixed: %v, blocks: %v\n",
			str, codepoints, mixed, blockNames)
	}

	// Global utils
	fmt.Println("\nGlobal Utils:")
	normalized := v3.DefaultIdnUtils.NormalizeDomainLabel("EXAMPLE")
	fmt.Printf("Normalize 'EXAMPLE' -> '%s'\n", normalized)

	fmt.Println("\n=== Simple example completed ===")
}
