package v3

import (
	"path/filepath"
	"testing"
)

func TestIdnLoaderBasic(t *testing.T) {
	idnPath := filepath.Join("..", "idn")
	loader := NewIdnLoader(idnPath)

	t.Run("LoadSingleTable", func(t *testing.T) {
		table, err := loader.LoadTable("arabic")
		if err != nil {
			t.Skipf("Arabic IDN table not found: %v", err)
		}

		if table.GetName() != "arabic" {
			t.Errorf("Expected name 'arabic', got '%s'", table.GetName())
		}

		// Test cache
		table2, err := loader.LoadTable("arabic")
		if err != nil {
			t.Fatalf("Failed to load from cache: %v", err)
		}

		if table != table2 {
			t.Error("Expected same instance from cache")
		}
	})

	t.Run("ListLanguages", func(t *testing.T) {
		languages, err := loader.ListAvailableLanguages()
		if err != nil {
			t.Fatalf("Failed to list languages: %v", err)
		}

		if len(languages) == 0 {
			t.Skip("No IDN files found")
		}

		t.Logf("Available languages: %d", len(languages))
	})
}

func TestIdnManagerBasic(t *testing.T) {
	idnPath := filepath.Join("..", "idn")
	manager := NewIdnManager(idnPath)

	t.Run("RegisterAndValidate", func(t *testing.T) {
		err := manager.RegisterTld("test", []string{"latin"})
		if err != nil {
			t.Skipf("Could not register TLD: %v", err)
		}

		// Test validation
		isValid, languages, err := manager.ValidateLabel("test", "example")
		if err != nil {
			t.Fatalf("Validation failed: %v", err)
		}

		t.Logf("Label 'example' valid: %v, languages: %v", isValid, languages)
	})

	t.Run("MultipleLanguages", func(t *testing.T) {
		languages := []string{"latin", "arabic"}
		err := manager.RegisterTld("multi", languages)
		if err != nil {
			t.Skipf("Could not register multi-language TLD: %v", err)
		}

		config, err := manager.GetTldConfig("multi")
		if err != nil {
			t.Fatalf("Could not get TLD config: %v", err)
		}

		if len(config.Languages) != len(languages) {
			t.Errorf("Expected %d languages, got %d", len(languages), len(config.Languages))
		}
	})
}

func TestCachePerformance(t *testing.T) {
	idnPath := filepath.Join("..", "idn")
	loader := NewIdnLoader(idnPath)

	// Load a table first
	_, err := loader.LoadTable("latin")
	if err != nil {
		t.Skipf("Latin table not available: %v", err)
	}

	// Test cache stats
	stats := loader.GetCacheStats()
	cached, ok := stats["cached_languages"].(int)
	if !ok || cached != 1 {
		t.Errorf("Expected 1 cached language, got %v", cached)
	}
}
