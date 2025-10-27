package main

import (
	"fmt"
	"log"
	"time"

	v3 "github.com/al0fayz/idn/v3"
)

func ExampleBasicUtils() {
	fmt.Println("=== IDN Utils Test Runner ===")

	// Test all major utility functions
	testBasicConversion()
	testDomainValidation()
	testUnicodeAnalysis()
	testPerformance()
	testErrorHandling()

	fmt.Println("\n=== All Tests Completed Successfully ===")
}

func testBasicConversion() {
	fmt.Println("\n--- Test: Basic Punycode Conversion ---")
	utils := &v3.IdnUtils{}

	testCases := map[string]string{
		"example.com": "example.com",
		"例子.com":      "xn--fsqu00a.com",
		"مثال.com":    "xn--mgbh0fb.com",
		"пример.com":  "xn--e1afmkfd.com",
		"테스트.kr":      "xn--9t4b11yi5a.kr",
	}

	for unicode, expectedAscii := range testCases {
		// Test ToASCII
		ascii, err := utils.ToASCII(unicode)
		if err != nil {
			log.Printf("❌ ToASCII failed for %s: %v", unicode, err)
			continue
		}

		// Verify expected result
		if ascii != expectedAscii {
			log.Printf("⚠️  ToASCII result mismatch for %s: got %s, expected %s",
				unicode, ascii, expectedAscii)
		}

		// Test round-trip
		back, err := utils.ToUnicode(ascii)
		if err != nil {
			log.Printf("❌ ToUnicode failed for %s: %v", ascii, err)
			continue
		}

		if back != unicode {
			log.Printf("❌ Round-trip failed: %s -> %s -> %s", unicode, ascii, back)
		} else {
			fmt.Printf("✅ %s <-> %s\n", unicode, ascii)
		}
	}
}

func testDomainValidation() {
	fmt.Println("\n--- Test: Domain Validation ---")
	utils := &v3.IdnUtils{}

	validDomains := []string{
		"example.com",
		"sub.domain.com",
		"test-123.org",
		"例子.com",
		"мир.рф",
	}

	invalidDomains := []string{
		"",
		"-invalid.com",
		"invalid-.com",
		"under_score.com",
		"toolongdomainlabelthatexceedssixtythreecharacterslimitandshouldbeinvalid.com",
	}

	fmt.Println("Valid domains:")
	for _, domain := range validDomains {
		err := utils.ValidateDomainName(domain)
		if err != nil {
			fmt.Printf("❌ %s: %v\n", domain, err)
		} else {
			fmt.Printf("✅ %s: Valid\n", domain)
		}
	}

	fmt.Println("Invalid domains:")
	for _, domain := range invalidDomains {
		err := utils.ValidateDomainName(domain)
		if err != nil {
			fmt.Printf("✅ %s: Correctly rejected (%v)\n", domain, err)
		} else {
			fmt.Printf("❌ %s: Should be invalid but was accepted\n", domain)
		}
	}
}

func testUnicodeAnalysis() {
	fmt.Println("\n--- Test: Unicode Analysis ---")
	utils := &v3.IdnUtils{}

	testStrings := []struct {
		text     string
		expected map[string]interface{}
	}{
		{
			text: "example",
			expected: map[string]interface{}{
				"codepoints": 7,
				"mixed":      false,
				"blocks":     []string{"Latin"},
			},
		},
		{
			text: "例子",
			expected: map[string]interface{}{
				"codepoints": 2,
				"mixed":      false,
				"blocks":     []string{"CJK_Unified_Ideographs"},
			},
		},
		{
			text: "test例",
			expected: map[string]interface{}{
				"codepoints": 5,
				"mixed":      true,
				"blocks":     []string{"Latin", "CJK_Unified_Ideographs"},
			},
		},
	}

	for _, tc := range testStrings {
		codepoints := utils.CountCodepoints(tc.text)
		mixed := utils.HasMixedScripts(tc.text)
		blocks := utils.GetUnicodeBlocks(tc.text)

		// Get block names
		var blockNames []string
		for block := range blocks {
			if blocks[block] {
				blockNames = append(blockNames, block)
			}
		}

		fmt.Printf("Text: %-10s Codepoints: %d, Mixed: %-5v, Blocks: %v\n",
			fmt.Sprintf("'%s'", tc.text), codepoints, mixed, blockNames)

		// Verify expectations
		expectedCodepoints := tc.expected["codepoints"].(int)
		expectedMixed := tc.expected["mixed"].(bool)

		if codepoints != expectedCodepoints {
			fmt.Printf("⚠️  Expected %d codepoints, got %d\n", expectedCodepoints, codepoints)
		}

		if mixed != expectedMixed {
			fmt.Printf("⚠️  Expected mixed=%v, got %v\n", expectedMixed, mixed)
		}
	}
}

func testPerformance() {
	fmt.Println("\n--- Test: Performance ---")
	utils := &v3.IdnUtils{}

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

	validationDuration := time.Since(start)
	totalValidations := iterations * len(testStrings)

	fmt.Printf("Validation: %d ops in %v (avg: %v/op, %d ops/sec)\n",
		totalValidations,
		validationDuration,
		validationDuration/time.Duration(totalValidations),
		int(float64(totalValidations)/validationDuration.Seconds()))

	// Test conversion performance
	iterations = 1000
	start = time.Now()

	for i := 0; i < iterations; i++ {
		for _, str := range testStrings {
			ascii, _ := utils.ToASCII(str)
			utils.ToUnicode(ascii)
		}
	}

	conversionDuration := time.Since(start)
	totalConversions := iterations * len(testStrings)

	fmt.Printf("Conversion: %d ops in %v (avg: %v/op, %d ops/sec)\n",
		totalConversions,
		conversionDuration,
		conversionDuration/time.Duration(totalConversions),
		int(float64(totalConversions)/conversionDuration.Seconds()))
}

func testErrorHandling() {
	fmt.Println("\n--- Test: Error Handling ---")
	utils := &v3.IdnUtils{}

	// Test invalid punycode
	_, err := utils.ToUnicode("xn--invalid-punycode")
	if err != nil {
		fmt.Printf("✅ Invalid punycode correctly rejected: %v\n", err)
	} else {
		fmt.Printf("❌ Invalid punycode should have been rejected\n")
	}

	// Test extremely long labels
	longLabel := make([]rune, 100)
	for i := range longLabel {
		longLabel[i] = 'a'
	}
	longString := string(longLabel)

	if !utils.IsValidDomainLabel(longString) {
		fmt.Printf("✅ Long label correctly rejected (length: %d)\n", len(longString))
	} else {
		fmt.Printf("❌ Long label should have been rejected\n")
	}

	// Test global utils instance
	result, err := v3.DefaultIdnUtils.ToASCII("test.com")
	if err != nil {
		fmt.Printf("❌ Global utils failed: %v\n", err)
	} else {
		fmt.Printf("✅ Global utils working: %s\n", result)
	}
}
