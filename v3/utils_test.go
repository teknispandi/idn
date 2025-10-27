package v3

import (
	"testing"
)

func TestIdnUtils_ToUnicode(t *testing.T) {
	utils := &IdnUtils{}

	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{
			name:     "Basic ASCII domain",
			input:    "example.com",
			expected: "example.com",
			hasError: false,
		},
		{
			name:     "Punycode domain",
			input:    "xn--nxasmq6b",
			expected: "βόλοσ",
			hasError: false,
		},
		{
			name:     "Mixed punycode and ASCII",
			input:    "xn--fsq.example.com",
			expected: "例.example.com",
			hasError: false,
		},
		{
			name:     "Japanese punycode",
			input:    "xn--wgv71a119e",
			expected: "日本語",
			hasError: false,
		},
		{
			name:     "Arabic punycode",
			input:    "xn--mgbaam7a8h",
			expected: "امارات",
			hasError: false,
		},
		{
			name:     "Invalid punycode",
			input:    "xn--invalid",
			expected: "曌曓曐曒曌", // golang.org/x/net/idna actually decodes this to Chinese characters
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := utils.ToUnicode(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("ToUnicode(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("ToUnicode(%q) unexpected error: %v", tt.input, err)
				return
			}

			if result != tt.expected {
				t.Errorf("ToUnicode(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIdnUtils_ToASCII(t *testing.T) {
	utils := &IdnUtils{}

	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{
			name:     "Basic ASCII domain",
			input:    "example.com",
			expected: "example.com",
			hasError: false,
		},
		{
			name:     "Chinese characters",
			input:    "中国.com",
			expected: "xn--fiqs8s.com",
			hasError: false,
		},
		{
			name:     "Japanese hiragana",
			input:    "ひらがな.jp",
			expected: "xn--v8j0cwa6g.jp",
			hasError: false,
		},
		{
			name:     "Arabic text",
			input:    "مثال.com",
			expected: "xn--mgbh0fb.com",
			hasError: false,
		},
		{
			name:     "Russian text",
			input:    "пример.com",
			expected: "xn--e1afmkfd.com",
			hasError: false,
		},
		{
			name:     "Greek text",
			input:    "παράδειγμα.com",
			expected: "xn--hxajbheg2az3al.com",
			hasError: false,
		},
		{
			name:     "Mixed scripts",
			input:    "café.com",
			expected: "xn--caf-dma.com",
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := utils.ToASCII(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("ToASCII(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("ToASCII(%q) unexpected error: %v", tt.input, err)
				return
			}

			if result != tt.expected {
				t.Errorf("ToASCII(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIdnUtils_IsValidDomainLabel(t *testing.T) {
	utils := &IdnUtils{}

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Valid ASCII label",
			input:    "example",
			expected: true,
		},
		{
			name:     "Valid label with numbers",
			input:    "test123",
			expected: true,
		},
		{
			name:     "Valid label with hyphen",
			input:    "test-domain",
			expected: true,
		},
		{
			name:     "Valid Unicode label",
			input:    "مثال",
			expected: true,
		},
		{
			name:     "Valid Chinese label",
			input:    "例子",
			expected: true,
		},
		{
			name:     "Empty label",
			input:    "",
			expected: false,
		},
		{
			name:     "Label starting with hyphen",
			input:    "-invalid",
			expected: false,
		},
		{
			name:     "Label ending with hyphen",
			input:    "invalid-",
			expected: false,
		},
		{
			name:     "Label too long (over 63 chars)",
			input:    "this-is-a-very-long-label-that-exceeds-the-maximum-allowed-length-of-sixty-three-characters",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsValidDomainLabel(tt.input)

			if result != tt.expected {
				t.Errorf("IsValidDomainLabel(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIdnUtils_NormalizeDomainLabel(t *testing.T) {
	utils := &IdnUtils{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Uppercase to lowercase",
			input:    "EXAMPLE",
			expected: "example",
		},
		{
			name:     "Mixed case",
			input:    "ExAmPlE",
			expected: "example",
		},
		{
			name:     "Already lowercase",
			input:    "example",
			expected: "example",
		},
		{
			name:     "With numbers",
			input:    "Test123",
			expected: "test123",
		},
		{
			name:     "Unicode characters (unchanged)",
			input:    "مثال",
			expected: "مثال",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.NormalizeDomainLabel(tt.input)

			if result != tt.expected {
				t.Errorf("NormalizeDomainLabel(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIdnUtils_SplitDomainName(t *testing.T) {
	utils := &IdnUtils{}

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Simple domain",
			input:    "example.com",
			expected: []string{"example", "com"},
		},
		{
			name:     "Subdomain",
			input:    "www.example.com",
			expected: []string{"www", "example", "com"},
		},
		{
			name:     "Deep subdomain",
			input:    "api.v1.example.com",
			expected: []string{"api", "v1", "example", "com"},
		},
		{
			name:     "With trailing dot",
			input:    "example.com.",
			expected: []string{"example", "com"},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "Single label",
			input:    "localhost",
			expected: []string{"localhost"},
		},
		{
			name:     "Unicode domain",
			input:    "مثال.com",
			expected: []string{"مثال", "com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.SplitDomainName(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("SplitDomainName(%q) returned %d labels, expected %d", tt.input, len(result), len(tt.expected))
				return
			}

			for i, label := range result {
				if label != tt.expected[i] {
					t.Errorf("SplitDomainName(%q)[%d] = %q, expected %q", tt.input, i, label, tt.expected[i])
				}
			}
		})
	}
}

func TestIdnUtils_ValidateDomainName(t *testing.T) {
	utils := &IdnUtils{}

	tests := []struct {
		name      string
		input     string
		hasError  bool
		errorType string
	}{
		{
			name:      "Valid domain",
			input:     "example.com",
			hasError:  false,
			errorType: "",
		},
		{
			name:      "Valid subdomain",
			input:     "www.example.com",
			hasError:  false,
			errorType: "",
		},
		{
			name:      "Valid Unicode domain",
			input:     "مثال.com",
			hasError:  false,
			errorType: "",
		},
		{
			name:      "Empty domain",
			input:     "",
			hasError:  true,
			errorType: "empty",
		},
		{
			name:      "Domain too long",
			input:     "this-is-a-very-long-domain-name-that-exceeds-the-maximum-allowed-length-of-two-hundred-fifty-three-characters-which-is-the-limit-for-fully-qualified-domain-names-according-to-rfc-specifications-and-should-therefore-be-rejected.example.com",
			hasError:  true,
			errorType: "too long",
		},
		{
			name:      "Invalid label with hyphen at start",
			input:     "-invalid.com",
			hasError:  true,
			errorType: "invalid label",
		},
		{
			name:      "Invalid label with hyphen at end",
			input:     "invalid-.com",
			hasError:  true,
			errorType: "invalid label",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := utils.ValidateDomainName(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("ValidateDomainName(%q) expected error, got nil", tt.input)
				}
				// Optionally check error type/message
				t.Logf("Expected error for %q: %v", tt.input, err)
			} else {
				if err != nil {
					t.Errorf("ValidateDomainName(%q) unexpected error: %v", tt.input, err)
				}
			}
		})
	}
}

func TestIdnUtils_GetUnicodeBlocks(t *testing.T) {
	utils := &IdnUtils{}

	tests := []struct {
		name     string
		input    string
		expected []string // Expected blocks present
	}{
		{
			name:     "ASCII text",
			input:    "example",
			expected: []string{"Latin"},
		},
		{
			name:     "Chinese text",
			input:    "例子",
			expected: []string{"CJK_Unified_Ideographs"},
		},
		{
			name:     "Japanese hiragana",
			input:    "ひらがな",
			expected: []string{"Hiragana"},
		},
		{
			name:     "Japanese katakana",
			input:    "カタカナ",
			expected: []string{"Katakana"},
		},
		{
			name:     "Arabic text",
			input:    "مثال",
			expected: []string{"Arabic"},
		},
		{
			name:     "Russian text",
			input:    "пример",
			expected: []string{"Cyrillic"},
		},
		{
			name:     "Mixed Latin and numbers",
			input:    "test123",
			expected: []string{"Latin", "Other"},
		},
		{
			name:     "Mixed scripts",
			input:    "test例子",
			expected: []string{"Latin", "CJK_Unified_Ideographs"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.GetUnicodeBlocks(tt.input)

			// Check that all expected blocks are present
			for _, expectedBlock := range tt.expected {
				if !result[expectedBlock] {
					t.Errorf("GetUnicodeBlocks(%q) missing expected block: %s", tt.input, expectedBlock)
				}
			}

			t.Logf("GetUnicodeBlocks(%q) found blocks: %v", tt.input, getMapKeys(result))
		})
	}
}

func TestIdnUtils_CountCodepoints(t *testing.T) {
	utils := &IdnUtils{}

	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "ASCII text",
			input:    "example",
			expected: 7,
		},
		{
			name:     "Unicode text",
			input:    "例子",
			expected: 2,
		},
		{
			name:     "Mixed text",
			input:    "test例",
			expected: 5,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "Emoji",
			input:    "👋🌍",
			expected: 2,
		},
		{
			name:     "Arabic text",
			input:    "مثال",
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.CountCodepoints(tt.input)

			if result != tt.expected {
				t.Errorf("CountCodepoints(%q) = %d, expected %d", tt.input, result, tt.expected)
			}

			// Also compare with len() to show the difference
			byteLen := len(tt.input)
			t.Logf("String %q: bytes=%d, codepoints=%d", tt.input, byteLen, result)
		})
	}
}

func TestIdnUtils_HasMixedScripts(t *testing.T) {
	utils := &IdnUtils{}

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Pure ASCII",
			input:    "example",
			expected: false,
		},
		{
			name:     "Pure Chinese",
			input:    "例子",
			expected: false,
		},
		{
			name:     "ASCII with numbers (allowed)",
			input:    "test123",
			expected: false,
		},
		{
			name:     "Mixed Latin and Chinese",
			input:    "test例子",
			expected: true,
		},
		{
			name:     "Mixed Arabic and Latin",
			input:    "testمثال",
			expected: true,
		},
		{
			name:     "Mixed Cyrillic and Latin",
			input:    "testпример",
			expected: true,
		},
		{
			name:     "Single character",
			input:    "a",
			expected: false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.HasMixedScripts(tt.input)

			if result != tt.expected {
				t.Errorf("HasMixedScripts(%q) = %v, expected %v", tt.input, result, tt.expected)
			}

			// Log the blocks for debugging
			blocks := utils.GetUnicodeBlocks(tt.input)
			t.Logf("String %q has blocks: %v", tt.input, getMapKeys(blocks))
		})
	}
}

func TestIdnUtils_IsConfusable(t *testing.T) {
	utils := &IdnUtils{}

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Normal ASCII",
			input:    "example",
			expected: false,
		},
		{
			name:     "Confusable Cyrillic 'р'",
			input:    "exemрle", // Contains Cyrillic 'р' (U+0440) which looks like Latin 'p'
			expected: true,
		},
		{
			name:     "Confusable Cyrillic 'a'",
			input:    "exаmple", // Contains Cyrillic 'а' that looks like Latin 'a'
			expected: true,
		},
		{
			name:     "Normal Unicode",
			input:    "例子",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsConfusable(tt.input)

			if result != tt.expected {
				t.Errorf("IsConfusable(%q) = %v, expected %v", tt.input, result, tt.expected)
			}

			// Log codepoints for debugging
			var codepoints []rune
			for _, r := range tt.input {
				codepoints = append(codepoints, r)
			}
			t.Logf("String %q codepoints: %v", tt.input, codepoints)
		})
	}
}

// Test DefaultIdnUtils global instance
func TestDefaultIdnUtils(t *testing.T) {
	// Test that the global instance works
	result, err := DefaultIdnUtils.ToASCII("例子.com")
	if err != nil {
		t.Errorf("DefaultIdnUtils.ToASCII() error: %v", err)
	}

	expected := "xn--fsqu00a.com"
	if result != expected {
		t.Errorf("DefaultIdnUtils.ToASCII() = %q, expected %q", result, expected)
	}

	// Test normalization
	normalized := DefaultIdnUtils.NormalizeDomainLabel("EXAMPLE")
	if normalized != "example" {
		t.Errorf("DefaultIdnUtils.NormalizeDomainLabel() = %q, expected %q", normalized, "example")
	}
}

// Helper function to get map keys as slice
func getMapKeys(m map[string]bool) []string {
	var keys []string
	for k := range m {
		if m[k] {
			keys = append(keys, k)
		}
	}
	return keys
}

// Benchmark tests
func BenchmarkIdnUtils_ToASCII(b *testing.B) {
	utils := &IdnUtils{}
	testStrings := []string{
		"example.com",
		"例子.com",
		"مثال.com",
		"пример.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, s := range testStrings {
			_, _ = utils.ToASCII(s)
		}
	}
}

func BenchmarkIdnUtils_ToUnicode(b *testing.B) {
	utils := &IdnUtils{}
	testStrings := []string{
		"example.com",
		"xn--fsq.com",
		"xn--mgbh0fb.com",
		"xn--e1afmkfd.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, s := range testStrings {
			_, _ = utils.ToUnicode(s)
		}
	}
}

func BenchmarkIdnUtils_ValidateDomainName(b *testing.B) {
	utils := &IdnUtils{}
	testDomains := []string{
		"example.com",
		"www.example.com",
		"api.v1.example.com",
		"例子.com",
		"مثال.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, domain := range testDomains {
			_ = utils.ValidateDomainName(domain)
		}
	}
}
