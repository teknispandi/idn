package v3

import (
	"reflect"
	"testing"
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
