package v3

import (
	"reflect"
	"testing"
	"time"
)

// MockTld adalah implementasi mock dari TldInterface untuk testing
type MockTld struct {
	idnTables []IdnTableType
}

func (m *MockTld) GetIdnTables() []IdnTableType {
	return m.idnTables
}

func TestIdnLabelValidator_FindValidIdnTableForTld(t *testing.T) {
	validator := NewIdnLabelValidator()

	tests := []struct {
		name          string
		label         string
		tld           TldInterface
		expectedTable string
		expectedValid bool
	}{
		{
			name:          "Valid ASCII label with default tables",
			label:         "example",
			tld:           &MockTld{idnTables: []IdnTableType{}}, // empty = use defaults
			expectedTable: "extended_latin",
			expectedValid: true,
		},
		{
			name:          "Valid label with hyphen",
			label:         "ex-ample",
			tld:           &MockTld{idnTables: []IdnTableType{}},
			expectedTable: "extended_latin",
			expectedValid: true,
		},
		{
			name:          "Valid label with numbers",
			label:         "example123",
			tld:           &MockTld{idnTables: []IdnTableType{}},
			expectedTable: "extended_latin",
			expectedValid: true,
		},
		{
			name:          "TLD with specific IDN table",
			label:         "example",
			tld:           &MockTld{idnTables: []IdnTableType{ExtendedLatin}},
			expectedTable: "extended_latin",
			expectedValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTable, gotValid := validator.FindValidIdnTableForTld(tt.label, tt.tld)

			if gotValid != tt.expectedValid {
				t.Errorf("FindValidIdnTableForTld() valid = %v, expected %v", gotValid, tt.expectedValid)
			}

			if gotTable != tt.expectedTable {
				t.Errorf("FindValidIdnTableForTld() table = %v, expected %v", gotTable, tt.expectedTable)
			}
		})
	}
}

func TestIdnLabelValidator_GetIdnTablesForTld(t *testing.T) {
	validator := NewIdnLabelValidator()

	tests := []struct {
		name     string
		tld      TldInterface
		expected []string
	}{
		{
			name:     "TLD with no configured tables should use defaults",
			tld:      &MockTld{idnTables: []IdnTableType{}},
			expected: []string{"extended_latin", "ja"},
		},
		{
			name:     "TLD with specific table",
			tld:      &MockTld{idnTables: []IdnTableType{ExtendedLatin}},
			expected: []string{"extended_latin"},
		},
		{
			name:     "TLD with multiple tables",
			tld:      &MockTld{idnTables: []IdnTableType{ExtendedLatin, Japanese}},
			expected: []string{"extended_latin", "ja"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validator.GetIdnTablesForTld(tt.tld)

			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("GetIdnTablesForTld() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestJapaneseLanguageValidator_IsValidLabelForLanguage(t *testing.T) {
	validator := &JapaneseLanguageValidator{}

	tests := []struct {
		name     string
		label    string
		expected bool
	}{
		{
			name:     "Valid ASCII label",
			label:    "example",
			expected: true,
		},
		{
			name:     "Valid length Japanese label",
			label:    "あいうえお", // 5 characters
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validator.IsValidLabelForLanguage(tt.label)
			if got != tt.expected {
				t.Errorf("IsValidLabelForLanguage(%q) = %v, expected %v", tt.label, got, tt.expected)
			}
		})
	}
}

func TestParseIdnTableType(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    IdnTableType
		expectError bool
	}{
		{
			name:        "Valid extended_latin",
			input:       "extended_latin",
			expected:    ExtendedLatin,
			expectError: false,
		},
		{
			name:        "Valid ja",
			input:       "ja",
			expected:    Japanese,
			expectError: false,
		},
		{
			name:        "Invalid table type",
			input:       "invalid",
			expected:    0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseIdnTableType(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("ParseIdnTableType(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseIdnTableType(%q) unexpected error: %v", tt.input, err)
				return
			}

			if got != tt.expected {
				t.Errorf("ParseIdnTableType(%q) = %v, expected %v", tt.input, got, tt.expected)
			}
		})
	}
}

// Test examples for IDN Utils integration
func TestIdnUtils_Integration(t *testing.T) {
	utils := &IdnUtils{}

	t.Run("PunycodeRoundTrip", func(t *testing.T) {
		testCases := []struct {
			name     string
			original string
		}{
			{"Chinese", "例子"},
			{"Arabic", "مثال"},
			{"Japanese", "テスト"},
			{"Russian", "пример"},
			{"Greek", "παράδειγμα"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Convert to ASCII (punycode)
				ascii, err := utils.ToASCII(tc.original)
				if err != nil {
					t.Fatalf("ToASCII failed: %v", err)
				}

				// Convert back to Unicode
				unicode, err := utils.ToUnicode(ascii)
				if err != nil {
					t.Fatalf("ToUnicode failed: %v", err)
				}

				// Should be identical to original
				if unicode != tc.original {
					t.Errorf("Round trip failed: %q -> %q -> %q", tc.original, ascii, unicode)
				}

				t.Logf("Round trip success: %q <-> %q", tc.original, ascii)
			})
		}
	})

	t.Run("DomainValidationWithIDN", func(t *testing.T) {
		validDomains := []string{
			"example.com",
			"例子.com",
			"مثال.test",
			"test-123.org",
			"sub.domain.com",
		}

		for _, domain := range validDomains {
			t.Run(domain, func(t *testing.T) {
				err := utils.ValidateDomainName(domain)
				if err != nil {
					t.Errorf("Valid domain %q rejected: %v", domain, err)
				}

				// Test splitting
				labels := utils.SplitDomainName(domain)
				t.Logf("Domain %q split into: %v", domain, labels)

				// Test each label
				for i, label := range labels {
					if !utils.IsValidDomainLabel(label) {
						t.Errorf("Label %d (%q) in domain %q is invalid", i, label, domain)
					}
				}
			})
		}
	})

	t.Run("ScriptMixingDetection", func(t *testing.T) {
		testCases := []struct {
			input  string
			mixed  bool
			blocks []string
		}{
			{"example", false, []string{"Latin"}},
			{"例子", false, []string{"CJK_Unified_Ideographs"}},
			{"test123", false, []string{"Latin", "Other"}}, // Latin + digits allowed
			{"test例", true, []string{"Latin", "CJK_Unified_Ideographs"}},
			{"مثالtest", true, []string{"Arabic", "Latin"}},
		}

		for _, tc := range testCases {
			t.Run(tc.input, func(t *testing.T) {
				mixed := utils.HasMixedScripts(tc.input)
				if mixed != tc.mixed {
					t.Errorf("HasMixedScripts(%q) = %v, expected %v", tc.input, mixed, tc.mixed)
				}

				blocks := utils.GetUnicodeBlocks(tc.input)
				var blockNames []string
				for k := range blocks {
					if blocks[k] {
						blockNames = append(blockNames, k)
					}
				}
				t.Logf("Input %q: mixed=%v, blocks=%v", tc.input, mixed, blockNames)
			})
		}
	})

	t.Run("CodepointCounting", func(t *testing.T) {
		testCases := []struct {
			input      string
			codepoints int
			bytes      int
		}{
			{"abc", 3, 3},
			{"例子", 2, 6},
			{"مثال", 4, 8},
			{"test例", 5, 7},
			{"👋🌍", 2, 8},
		}

		for _, tc := range testCases {
			t.Run(tc.input, func(t *testing.T) {
				codepoints := utils.CountCodepoints(tc.input)
				bytes := len(tc.input)

				if codepoints != tc.codepoints {
					t.Errorf("CountCodepoints(%q) = %d, expected %d", tc.input, codepoints, tc.codepoints)
				}

				if bytes != tc.bytes {
					t.Errorf("Byte length of %q = %d, expected %d", tc.input, bytes, tc.bytes)
				}

				t.Logf("String %q: %d codepoints, %d bytes", tc.input, codepoints, bytes)
			})
		}
	})
}

// Performance comparison test
func TestIdnUtils_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	utils := &IdnUtils{}

	testStrings := []string{
		"example.com",
		"例子.com",
		"مثال.com",
		"пример.com",
		"test-domain-123.org",
	}

	t.Run("ValidationPerformance", func(t *testing.T) {
		iterations := 10000

		for _, s := range testStrings {
			t.Run(s, func(t *testing.T) {
				start := time.Now()

				for i := 0; i < iterations; i++ {
					_ = utils.IsValidDomainLabel(s)
				}

				duration := time.Since(start)
				avgTime := duration / time.Duration(iterations)

				t.Logf("Validated %q %d times in %v (avg: %v per validation)",
					s, iterations, duration, avgTime)
			})
		}
	})

	t.Run("ConversionPerformance", func(t *testing.T) {
		iterations := 1000

		for _, s := range testStrings {
			t.Run(s, func(t *testing.T) {
				start := time.Now()

				for i := 0; i < iterations; i++ {
					ascii, _ := utils.ToASCII(s)
					_, _ = utils.ToUnicode(ascii)
				}

				duration := time.Since(start)
				avgTime := duration / time.Duration(iterations)

				t.Logf("Round-trip converted %q %d times in %v (avg: %v per conversion)",
					s, iterations, duration, avgTime)
			})
		}
	})
}
