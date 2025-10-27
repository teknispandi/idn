package v3

import (
	"testing"
)

func TestIdnUtils_BasicFunctionality(t *testing.T) {
	utils := &IdnUtils{}

	t.Run("ToASCII_Basic", func(t *testing.T) {
		result, err := utils.ToASCII("example.com")
		if err != nil {
			t.Errorf("ToASCII failed: %v", err)
		}
		if result != "example.com" {
			t.Errorf("Expected 'example.com', got '%s'", result)
		}
	})

	t.Run("ToASCII_Unicode", func(t *testing.T) {
		result, err := utils.ToASCII("例子.com")
		if err != nil {
			t.Errorf("ToASCII failed: %v", err)
		}
		expected := "xn--fsqu00a.com"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("IsValidDomainLabel", func(t *testing.T) {
		validCases := []string{"example", "test123", "test-domain", "例子"}
		invalidCases := []string{"", "-invalid", "invalid-"}

		for _, valid := range validCases {
			if !utils.IsValidDomainLabel(valid) {
				t.Errorf("Expected '%s' to be valid", valid)
			}
		}

		for _, invalid := range invalidCases {
			if utils.IsValidDomainLabel(invalid) {
				t.Errorf("Expected '%s' to be invalid", invalid)
			}
		}
	})

	t.Run("SplitDomainName", func(t *testing.T) {
		result := utils.SplitDomainName("www.example.com")
		expected := []string{"www", "example", "com"}

		if len(result) != len(expected) {
			t.Errorf("Expected %d labels, got %d", len(expected), len(result))
		}

		for i, label := range result {
			if label != expected[i] {
				t.Errorf("Expected label[%d] = '%s', got '%s'", i, expected[i], label)
			}
		}
	})

	t.Run("CountCodepoints", func(t *testing.T) {
		testCases := map[string]int{
			"abc":   3,
			"例子":    2,
			"test例": 5,
		}

		for input, expected := range testCases {
			result := utils.CountCodepoints(input)
			if result != expected {
				t.Errorf("CountCodepoints('%s') = %d, expected %d", input, result, expected)
			}
		}
	})

	t.Run("HasMixedScripts", func(t *testing.T) {
		nonMixed := []string{"example", "例子", "test123"}
		mixed := []string{"test例", "exampleمثال"}

		for _, str := range nonMixed {
			if utils.HasMixedScripts(str) {
				t.Errorf("Expected '%s' to not have mixed scripts", str)
			}
		}

		for _, str := range mixed {
			if !utils.HasMixedScripts(str) {
				t.Errorf("Expected '%s' to have mixed scripts", str)
			}
		}
	})
}
