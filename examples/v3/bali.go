package main

import (
	"log"

	v3 "github.com/al0fayz/idn/v3"
)

func ExampleBali() {
	validator := &v3.BalineseLanguageValidator{}

	tests := []struct {
		name     string
		label    string
		expected bool
	}{
		{
			name:     "✅ Valid length Balinese label",
			label:    "ᬩᭂᬘᬶ", // 2 characters
			expected: true,
		},
		{name: "✅ Valid Bali 1", label: "ᬩᬓᬵ", expected: true},
		{name: "✅ Valid Bali 2", label: "ᬳᬶᬩᭂ", expected: true},
		{name: "✅ Valid Bali 3", label: "ᬢᬷᬓᬸ", expected: true},
		{name: "✅ Valid Bali 4", label: "ᬓᬸᬓ᭄ᬱ", expected: true},
		// Invalid cases
		{
			name:     "❌ Not Valid ASCII label",
			label:    "example",
			expected: false,
		},
		{
			name:     "❌ Not Valid ASCII label mix with Balinese",
			label:    "example-ᬩᭂᬘᬶ",
			expected: false,
		},
		{name: "❌ Invalid: dv at start", label: "ᬸᬩᬓ", expected: false},
		{name: "❌ Invalid: virama before dv", label: "ᬳ᭄ᬵ", expected: false},
		{name: "❌ Invalid: double virama", label: "ᬓᬓ᭄᭄ᬱ", expected: false},
		{name: "❌ Invalid: 3 consonants before virama", label: "ᬓᬓᬓ᭄ᬳ", expected: false},
		{name: "❌ Invalid: hyphen end", label: "ᬩᭂᬘᬶ-", expected: false},
		{name: "❌ Invalid: dv at start 2", label: "ᭂᬩᬓᬵ", expected: false},
	}
	for _, tt := range tests {
		got := validator.IsValidLabelForLanguage(tt.label)
		log.Printf("- %s: expected %v, got %v", tt.name, tt.expected, got)
	}
}
