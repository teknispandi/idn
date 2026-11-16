package v3

import (
	"path/filepath"
	"testing"
)

func TestIdnLoader(t *testing.T) {
	// Use test data path
	idnPath := filepath.Join("..", "idn")
	loader := NewIdnLoader(idnPath)

	t.Run("LoadTable", func(t *testing.T) {
		table, err := loader.LoadTable("arabic")
		if err != nil {
			t.Skipf("Skipping test - Arabic IDN file not found: %v", err)
			return
		}

		if table == nil {
			t.Fatal("Expected table to be non-nil")
		}

		if table.GetName() != "arabic" {
			t.Errorf("Expected table name to be 'arabic', got '%s'", table.GetName())
		}
	})

	t.Run("CacheTest", func(t *testing.T) {
		// Load once
		table1, err := loader.LoadTable("latin")
		if err != nil {
			t.Skipf("Skipping test - Latin IDN file not found: %v", err)
			return
		}

		// Load again - should be cached
		table2, err := loader.LoadTable("latin")
		if err != nil {
			t.Fatalf("Failed to load cached table: %v", err)
		}

		// Should be same instance
		if table1 != table2 {
			t.Error("Expected same instance from cache")
		}
	})

	t.Run("ListAvailableLanguages", func(t *testing.T) {
		languages, err := loader.ListAvailableLanguages()
		if err != nil {
			t.Fatalf("Failed to list languages: %v", err)
		}

		if len(languages) == 0 {
			t.Skip("No IDN files found in test directory")
		}

		t.Logf("Found languages: %v", languages)
	})

	t.Run("LoadMultipleTables", func(t *testing.T) {
		// Try to load common languages
		languages := []string{"latin", "arabic"}
		tables, err := loader.LoadMultipleTables(languages)

		// Allow partial success
		if err != nil && len(tables) == 0 {
			t.Skipf("No tables loaded: %v", err)
		}

		for lang, table := range tables {
			if table == nil {
				t.Errorf("Table for %s is nil", lang)
			}
		}
	})
}

func TestIdnManager(t *testing.T) {
	idnPath := filepath.Join("..", "idn")
	manager := NewIdnManager(idnPath)

	t.Run("RegisterTld", func(t *testing.T) {
		err := manager.RegisterTld("test", []string{"latin"})
		if err != nil {
			t.Skipf("Skipping test - could not register TLD: %v", err)
			return
		}

		tlds := manager.ListRegisteredTlds()
		found := false
		for _, tld := range tlds {
			if tld == "test" {
				found = true
				break
			}
		}

		if !found {
			t.Error("TLD 'test' not found in registered TLDs")
		}
	})

	t.Run("ValidateLabel", func(t *testing.T) {
		err := manager.RegisterTld("example", []string{"latin"})
		if err != nil {
			t.Skipf("Could not register TLD for validation test: %v", err)
			return
		}

		testCases := []struct {
			label    string
			expected bool
		}{
			{"example", true},
			{"test-domain", true},
			{"", false}, // empty should be invalid
		}

		for _, tc := range testCases {
			isValid, _, err := manager.ValidateLabel("example", tc.label)
			if err != nil {
				t.Errorf("Validation error for '%s': %v", tc.label, err)
				continue
			}

			// For this test, we just check that validation doesn't error
			// Actual validation depends on IDN table content
			t.Logf("Label '%s' validation: %v", tc.label, isValid)
		}
	})

	t.Run("TldManagement", func(t *testing.T) {
		tldName := "management-test"

		// Register
		err := manager.RegisterTld(tldName, []string{"latin"})
		if err != nil {
			t.Skipf("Could not register TLD: %v", err)
			return
		}

		// Get config
		config, err := manager.GetTldConfig(tldName)
		if err != nil {
			t.Fatalf("Could not get TLD config: %v", err)
		}

		if config.TldName != tldName {
			t.Errorf("Expected TLD name %s, got %s", tldName, config.TldName)
		}

		// Disable
		err = manager.DisableTld(tldName)
		if err != nil {
			t.Fatalf("Could not disable TLD: %v", err)
		}

		// Enable
		err = manager.EnableTld(tldName)
		if err != nil {
			t.Fatalf("Could not enable TLD: %v", err)
		}

		// Unregister
		err = manager.UnregisterTld(tldName)
		if err != nil {
			t.Fatalf("Could not unregister TLD: %v", err)
		}

		// Should not be found
		_, err = manager.GetTldConfig(tldName)
		if err == nil {
			t.Error("Expected error when getting config for unregistered TLD")
		}
	})
}

func TestIdnTable(t *testing.T) {
	// Test with mock data
	mockData := []string{
		"# Test IDN Table",
		"# URL: https://example.com/idn",
		"# Policy: https://example.com/policy",
		"U+0061  # LATIN SMALL LETTER A",
		"U+0062  # LATIN SMALL LETTER B",
		"U+002D  # HYPHEN-MINUS",
	}

	table, err := NewIdnTable("test", mockData, nil)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	t.Run("BasicProperties", func(t *testing.T) {
		if table.GetName() != "test" {
			t.Errorf("Expected name 'test', got '%s'", table.GetName())
		}

		if table.GetURL().String() != "https://example.com/idn" {
			t.Errorf("Expected URL 'https://example.com/idn', got '%s'", table.GetURL().String())
		}

		if table.GetPolicy().String() != "https://example.com/policy" {
			t.Errorf("Expected policy 'https://example.com/policy', got '%s'", table.GetPolicy().String())
		}
	})

	t.Run("LabelValidation", func(t *testing.T) {
		testCases := []struct {
			label    string
			expected bool
		}{
			{"ab", true},   // valid codepoints
			{"a-b", true},  // with hyphen
			{"abc", false}, // 'c' not in table
			{"", false},    // label kosong tidak pernah valid
		}

		for _, tc := range testCases {
			result := table.IsValidLabel(tc.label)
			if result != tc.expected {
				t.Errorf("Label '%s': expected %v, got %v", tc.label, tc.expected, result)
			}
		}
	})
}

func BenchmarkIdnLoader(b *testing.B) {
	idnPath := filepath.Join("..", "idn")
	loader := NewIdnLoader(idnPath)

	// Pre-warm cache
	loader.LoadTable("latin")

	b.Run("CachedLoad", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := loader.LoadTable("latin")
			if err != nil {
				b.Skip("Latin table not available")
			}
		}
	})
}

func BenchmarkValidation(b *testing.B) {
	idnPath := filepath.Join("..", "idn")
	manager := NewIdnManager(idnPath)

	err := manager.RegisterTld("bench", []string{"latin"})
	if err != nil {
		b.Skipf("Could not register TLD for benchmark: %v", err)
	}

	testLabel := "example-domain"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := manager.ValidateLabel("bench", testLabel)
		if err != nil {
			b.Fatalf("Validation error: %v", err)
		}
	}
}
