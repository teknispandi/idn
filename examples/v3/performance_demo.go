package main

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	v3 "github.com/al0fayz/idn/v3"
)

func PerformanceDemo() {
	fmt.Println("=== IDN Performance Test ===")

	idnPath := filepath.Join("..", "..", "idn")
	loader := v3.NewIdnLoader(idnPath)

	// 1. Test loading performance dengan cache
	fmt.Println("\n--- Cache Performance Test ---")

	// First load (from file)
	start := time.Now()
	arabicTable1, err := loader.LoadTable("arabic")
	firstLoadTime := time.Since(start)
	if err != nil {
		log.Fatalf("Error loading Arabic table: %v", err)
	}

	// Second load (from cache)
	start = time.Now()
	arabicTable2, err := loader.LoadTable("arabic")
	cachedLoadTime := time.Since(start)
	if err != nil {
		log.Fatalf("Error loading Arabic table from cache: %v", err)
	}

	fmt.Printf("First load (from file): %v\n", firstLoadTime)
	fmt.Printf("Second load (from cache): %v\n", cachedLoadTime)
	fmt.Printf("Cache speedup: %.2fx\n", float64(firstLoadTime)/float64(cachedLoadTime))

	// Verify sama instance
	fmt.Printf("Same instance from cache: %v\n", arabicTable1 == arabicTable2)

	// 2. Test multiple language loading
	fmt.Println("\n--- Multiple Language Loading ---")
	languages := []string{"arabic", "chinese", "japanese", "korean", "latin"}

	start = time.Now()
	tables, err := loader.LoadMultipleTables(languages)
	multiLoadTime := time.Since(start)
	if err != nil {
		log.Printf("Error loading multiple tables: %v", err)
	} else {
		fmt.Printf("Loaded %d languages in: %v\n", len(tables), multiLoadTime)
		for lang, table := range tables {
			fmt.Printf("  - %s: %s\n", lang, table.GetName())
		}
	}

	// 3. Test preloading
	fmt.Println("\n--- Preloading Test ---")
	manager := v3.NewIdnManager(idnPath)

	// Without preloading
	start = time.Now()
	err = manager.RegisterTld("test1", languages)
	withoutPreloadTime := time.Since(start)
	if err != nil {
		log.Printf("Error registering without preload: %v", err)
	}

	// With preloading
	start = time.Now()
	err = manager.RegisterTldWithPreload("test2", languages)
	withPreloadTime := time.Since(start)
	if err != nil {
		log.Printf("Error registering with preload: %v", err)
	}

	fmt.Printf("Register without preload: %v\n", withoutPreloadTime)
	fmt.Printf("Register with preload: %v\n", withPreloadTime)

	// 4. Validation performance test
	fmt.Println("\n--- Validation Performance Test ---")

	testLabels := []string{
		"example",
		"test-domain",
		"café",
		"мир",
		"例え",
		"مثال",
		"παράδειγμα",
	}

	iterations := 1000

	start = time.Now()
	for i := 0; i < iterations; i++ {
		for _, label := range testLabels {
			_, _, _ = manager.ValidateLabel("test1", label)
		}
	}
	validationTime := time.Since(start)

	totalValidations := iterations * len(testLabels)
	avgTimePerValidation := time.Duration(int64(validationTime) / int64(totalValidations))

	fmt.Printf("Total validations: %d\n", totalValidations)
	fmt.Printf("Total time: %v\n", validationTime)
	fmt.Printf("Average time per validation: %v\n", avgTimePerValidation)
	fmt.Printf("Validations per second: %.0f\n", float64(totalValidations)/validationTime.Seconds())

	// 5. Memory usage statistics
	fmt.Println("\n--- Memory Usage Statistics ---")
	stats := manager.GetStats()
	fmt.Printf("Manager stats: %+v\n", stats)

	// 6. Cache hit rate simulation
	fmt.Println("\n--- Cache Hit Rate Test ---")

	// Mix of cached and uncached languages
	testLanguages := []string{"arabic", "chinese", "new-lang", "latin", "japanese", "unknown-lang"}

	var hits, misses int
	for _, lang := range testLanguages {
		start := time.Now()
		_, err := loader.LoadTable(lang)
		loadTime := time.Since(start)

		if err != nil {
			fmt.Printf("❌ %s: ERROR (%v)\n", lang, err)
			misses++
		} else if loadTime < time.Microsecond*100 { // Arbitrary threshold for cache hit
			fmt.Printf("✓ %s: CACHE HIT (%v)\n", lang, loadTime)
			hits++
		} else {
			fmt.Printf("○ %s: LOADED (%v)\n", lang, loadTime)
		}
	}

	fmt.Printf("\nCache hit rate: %.2f%% (%d hits, %d misses)\n",
		float64(hits)/float64(hits+misses)*100, hits, misses)

	fmt.Println("\n=== Performance test completed ===")
}
