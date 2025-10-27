package main

import (
	"fmt"
	"log"
	"time"

	v3 "github.com/al0fayz/idn/v3"
)

func UtilsExample() {
	fmt.Println("=== IDN Utils Examples ===")

	utils := &v3.IdnUtils{}

	// 1. Punycode Conversion Examples
	fmt.Println("\n--- Punycode Conversion ---")
	unicodeDomains := []string{
		"例子.com",         // Chinese
		"مثال.com",       // Arabic
		"пример.com",     // Russian
		"παράδειγμα.com", // Greek
		"テスト.jp",         // Japanese
		"한국.kr",          // Korean
	}

	for _, domain := range unicodeDomains {
		ascii, err := utils.ToASCII(domain)
		if err != nil {
			log.Printf("❌ ToASCII failed for %s: %v", domain, err)
			continue
		}

		// Convert back to verify
		unicode, err := utils.ToUnicode(ascii)
		if err != nil {
			log.Printf("❌ ToUnicode failed for %s: %v", ascii, err)
			continue
		}

		fmt.Printf("Unicode: %-20s -> ASCII: %-25s -> Unicode: %s\n",
			domain, ascii, unicode)

		if unicode != domain {
			fmt.Printf("⚠️  Round-trip mismatch for %s\n", domain)
		}
	}

	// 2. Domain Validation Examples
	fmt.Println("\n--- Domain Validation ---")
	testDomains := []string{
		"example.com",
		"sub.example.com",
		"test-123.org",
		"例子.com",
		"مثال.test",
		"",             // empty
		"-invalid.com", // starts with hyphen
		"invalid-.com", // ends with hyphen
		"this-is-a-very-long-domain-name-that-exceeds-the-maximum-allowed-length-of-sixty-three-characters.com",
	}

	for _, domain := range testDomains {
		err := utils.ValidateDomainName(domain)
		if err != nil {
			fmt.Printf("❌ Domain '%s': %v\n", domain, err)
		} else {
			fmt.Printf("✅ Domain '%s': Valid\n", domain)
		}

		// Show domain splitting
		if domain != "" {
			labels := utils.SplitDomainName(domain)
			fmt.Printf("   Labels: %v\n", labels)
		}
	}

	// 3. Label Validation Examples
	fmt.Println("\n--- Label Validation ---")
	testLabels := []string{
		"example",
		"test-123",
		"مثال",
		"例子",
		"",
		"-invalid",
		"invalid-",
		"test_underscore", // underscores not allowed in domain labels
	}

	for _, label := range testLabels {
		isValid := utils.IsValidDomainLabel(label)
		normalized := utils.NormalizeDomainLabel(label)

		fmt.Printf("Label: %-15s Valid: %-5v Normalized: %s\n",
			fmt.Sprintf("'%s'", label), isValid, normalized)
	}

	// 4. Unicode Analysis Examples
	fmt.Println("\n--- Unicode Analysis ---")
	testStrings := []string{
		"example",  // Latin
		"例子",       // Chinese
		"مثال",     // Arabic
		"пример",   // Cyrillic
		"test例",    // Mixed Latin + Chinese
		"testمثال", // Mixed Latin + Arabic
		"test123",  // Latin + digits
		"👋🌍",       // Emoji
	}

	for _, str := range testStrings {
		codepoints := utils.CountCodepoints(str)
		bytes := len(str)
		blocks := utils.GetUnicodeBlocks(str)
		mixed := utils.HasMixedScripts(str)
		confusable := utils.IsConfusable(str)

		fmt.Printf("String: %-10s\n", fmt.Sprintf("'%s'", str))
		fmt.Printf("  Codepoints: %d, Bytes: %d\n", codepoints, bytes)
		fmt.Printf("  Unicode blocks: %v\n", getBlockNames(blocks))
		fmt.Printf("  Mixed scripts: %v, Confusable: %v\n", mixed, confusable)
		fmt.Println()
	}

	// 5. Performance Demo
	fmt.Println("\n--- Performance Demo ---")
	performanceDemo(utils)

	// 6. Global Utils Demo
	fmt.Println("\n--- Global Utils Demo ---")
	// Using the global DefaultIdnUtils instance
	result, err := v3.DefaultIdnUtils.ToASCII("例子.com")
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Global utils ToASCII('例子.com') = %s\n", result)
	}

	normalized := v3.DefaultIdnUtils.NormalizeDomainLabel("EXAMPLE")
	fmt.Printf("Global utils NormalizeDomainLabel('EXAMPLE') = %s\n", normalized)

	fmt.Println("\n=== IDN Utils Examples Completed ===")
}

func performanceDemo(utils *v3.IdnUtils) {
	testStrings := []string{
		"example.com",
		"例子.com",
		"مثال.com",
		"пример.com",
	}

	// Test validation performance
	iterations := 10000
	start := time.Now()

	for i := 0; i < iterations; i++ {
		for _, str := range testStrings {
			utils.IsValidDomainLabel(str)
		}
	}

	duration := time.Since(start)
	totalTests := iterations * len(testStrings)
	avgTime := duration / time.Duration(totalTests)

	fmt.Printf("Validation Performance:\n")
	fmt.Printf("  Total validations: %d\n", totalTests)
	fmt.Printf("  Total time: %v\n", duration)
	fmt.Printf("  Average per validation: %v\n", avgTime)
	fmt.Printf("  Validations per second: %.0f\n", float64(totalTests)/duration.Seconds())

	// Test conversion performance
	iterations = 1000
	start = time.Now()

	for i := 0; i < iterations; i++ {
		for _, str := range testStrings {
			ascii, _ := utils.ToASCII(str)
			utils.ToUnicode(ascii)
		}
	}

	duration = time.Since(start)
	totalConversions := iterations * len(testStrings)
	avgTime = duration / time.Duration(totalConversions)

	fmt.Printf("\nConversion Performance:\n")
	fmt.Printf("  Total round-trip conversions: %d\n", totalConversions)
	fmt.Printf("  Total time: %v\n", duration)
	fmt.Printf("  Average per conversion: %v\n", avgTime)
	fmt.Printf("  Conversions per second: %.0f\n", float64(totalConversions)/duration.Seconds())
}

func getBlockNames(blocks map[string]bool) []string {
	var names []string
	for block := range blocks {
		if blocks[block] {
			names = append(names, block)
		}
	}
	return names
}
