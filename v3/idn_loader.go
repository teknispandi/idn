package v3

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// IdnLoader mengelola loading dan caching IDN tables dari folder /idn
type IdnLoader struct {
	idnPath string
	cache   map[string]*IdnTable
	mutex   sync.RWMutex
}

// NewIdnLoader membuat instance baru IdnLoader
func NewIdnLoader(idnPath string) *IdnLoader {
	return &IdnLoader{
		idnPath: idnPath,
		cache:   make(map[string]*IdnTable),
	}
}

// LoadTable memuat IDN table dari file dan cache di memory
func (loader *IdnLoader) LoadTable(language string) (*IdnTable, error) {
	loader.mutex.RLock()
	if table, exists := loader.cache[language]; exists {
		loader.mutex.RUnlock()
		return table, nil
	}
	loader.mutex.RUnlock()

	// Lock untuk menulis
	loader.mutex.Lock()
	defer loader.mutex.Unlock()

	// Double-check locking pattern
	if table, exists := loader.cache[language]; exists {
		return table, nil
	}

	// Cari file yang sesuai dengan bahasa
	filename, err := loader.findIdnFile(language)
	if err != nil {
		return nil, fmt.Errorf("failed to find IDN file for language %s: %w", language, err)
	}

	// Load data dari file
	data, err := loader.loadFromFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load IDN table from %s: %w", filename, err)
	}

	// Buat IDN table
	validator := GetLanguageValidator(language)
	table, err := NewIdnTable(language, data, validator)
	if err != nil {
		return nil, fmt.Errorf("failed to create IDN table for %s: %w", language, err)
	}

	// Cache table
	loader.cache[language] = table
	return table, nil
}

// LoadMultipleTables memuat beberapa IDN tables sekaligus
func (loader *IdnLoader) LoadMultipleTables(languages []string) (map[string]*IdnTable, error) {
	tables := make(map[string]*IdnTable)

	for _, language := range languages {
		table, err := loader.LoadTable(language)
		if err != nil {
			return nil, fmt.Errorf("failed to load table for language %s: %w", language, err)
		}
		tables[language] = table
	}

	return tables, nil
}

// GetCachedTable mengembalikan table yang sudah di-cache (jika ada)
func (loader *IdnLoader) GetCachedTable(language string) (*IdnTable, bool) {
	loader.mutex.RLock()
	defer loader.mutex.RUnlock()

	table, exists := loader.cache[language]
	return table, exists
}

// ListAvailableLanguages mengembalikan daftar bahasa yang tersedia
func (loader *IdnLoader) ListAvailableLanguages() ([]string, error) {
	files, err := filepath.Glob(filepath.Join(loader.idnPath, "*-IDN.txt"))
	if err != nil {
		return nil, err
	}

	var languages []string
	for _, file := range files {
		basename := filepath.Base(file)
		// Hapus suffix "-IDN.txt" untuk mendapatkan nama bahasa
		language := strings.TrimSuffix(basename, "-IDN.txt")
		languages = append(languages, strings.ToLower(language))
	}

	return languages, nil
}

// PreloadLanguages memuat beberapa bahasa ke dalam cache
func (loader *IdnLoader) PreloadLanguages(languages []string) error {
	for _, language := range languages {
		_, err := loader.LoadTable(language)
		if err != nil {
			return fmt.Errorf("failed to preload language %s: %w", language, err)
		}
	}
	return nil
}

// ClearCache membersihkan cache
func (loader *IdnLoader) ClearCache() {
	loader.mutex.Lock()
	defer loader.mutex.Unlock()

	loader.cache = make(map[string]*IdnTable)
}

// GetCacheStats mengembalikan statistik cache
func (loader *IdnLoader) GetCacheStats() map[string]interface{} {
	loader.mutex.RLock()
	defer loader.mutex.RUnlock()

	return map[string]interface{}{
		"cached_languages": len(loader.cache),
		"languages":        loader.getCachedLanguageNames(),
	}
}

// findIdnFile mencari file IDN yang sesuai dengan nama bahasa
func (loader *IdnLoader) findIdnFile(language string) (string, error) {
	// Kemungkinan format nama file
	patterns := []string{
		fmt.Sprintf("%s-IDN.txt", strings.Title(language)),
		fmt.Sprintf("%s-IDN.txt", strings.ToUpper(language)),
		fmt.Sprintf("%s-IDN.txt", language),
	}

	for _, pattern := range patterns {
		filename := filepath.Join(loader.idnPath, pattern)
		if _, err := os.Stat(filename); err == nil {
			return filename, nil
		}
	}

	return "", fmt.Errorf("IDN file not found for language: %s", language)
}

// loadFromFile memuat data dari file IDN
func (loader *IdnLoader) loadFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

// getCachedLanguageNames mengembalikan daftar nama bahasa yang di-cache
func (loader *IdnLoader) getCachedLanguageNames() []string {
	var names []string
	for name := range loader.cache {
		names = append(names, name)
	}
	return names
}
