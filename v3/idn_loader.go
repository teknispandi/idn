package v3

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
)

// IdnLoader mengelola loading dan caching IDN tables dari folder /idn
type IdnLoader struct {
	fsys  fs.FS
	cache map[string]*IdnTable
	mutex sync.RWMutex
}

// NewIdnLoader membuat instance baru IdnLoader yang membaca tabel dari folder idnPath
func NewIdnLoader(idnPath string) *IdnLoader {
	return NewIdnLoaderFS(os.DirFS(idnPath))
}

// NewIdnLoaderFS membuat IdnLoader yang membaca tabel dari fs.FS, misalnya
// tabel yang disematkan di binary (idn.Tables() dari package root modul ini).
func NewIdnLoaderFS(fsys fs.FS) *IdnLoader {
	return &IdnLoader{
		fsys:  fsys,
		cache: make(map[string]*IdnTable),
	}
}

// LoadTable memuat IDN table dari file dan cache di memory.
//
// Nama bahasa tidak peka huruf besar/kecil dan dicocokkan dengan nama file
// tanpa akhiran "-IDN.txt" atau ".txt": "latin" → Latin-IDN.txt,
// "traditional-chinese" → Traditional-Chinese-IDN.txt, "und-bali" → und-Bali.txt.
// Nama tabel yang dihasilkan adalah bentuk huruf kecilnya.
func (loader *IdnLoader) LoadTable(language string) (*IdnTable, error) {
	language = normalizeLanguageName(language)

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

// ListAvailableLanguages mengembalikan daftar bahasa yang tersedia, terurut
func (loader *IdnLoader) ListAvailableLanguages() ([]string, error) {
	files, err := loader.tableFiles()
	if err != nil {
		return nil, err
	}

	languages := make([]string, 0, len(files))
	for language := range files {
		languages = append(languages, language)
	}
	sort.Strings(languages)

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

// tableFiles memetakan nama bahasa (huruf kecil) ke nama file tabel di
// root fsys. "-IDN.txt" diutamakan bila ada dua file untuk bahasa yang sama.
func (loader *IdnLoader) tableFiles() (map[string]string, error) {
	entries, err := fs.ReadDir(loader.fsys, ".")
	if err != nil {
		return nil, err
	}

	files := make(map[string]string)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || path.Ext(name) != ".txt" {
			continue
		}
		stem := strings.TrimSuffix(name, ".txt")
		hasSuffix := strings.HasSuffix(strings.ToLower(stem), "-idn")
		if hasSuffix {
			stem = stem[:len(stem)-len("-idn")]
		}
		language := normalizeLanguageName(stem)
		if _, exists := files[language]; !exists || hasSuffix {
			files[language] = name
		}
	}
	return files, nil
}

// findIdnFile mencari file IDN yang sesuai dengan nama bahasa
func (loader *IdnLoader) findIdnFile(language string) (string, error) {
	files, err := loader.tableFiles()
	if err != nil {
		return "", err
	}
	if filename, ok := files[language]; ok {
		return filename, nil
	}

	return "", fmt.Errorf("IDN file not found for language: %s", language)
}

// normalizeLanguageName menyeragamkan nama bahasa: huruf kecil, spasi dan
// garis bawah menjadi tanda hubung ("Traditional Chinese" → "traditional-chinese").
func normalizeLanguageName(language string) string {
	language = strings.ToLower(strings.TrimSpace(language))
	return strings.NewReplacer(" ", "-", "_", "-").Replace(language)
}

// loadFromFile memuat data dari file IDN
func (loader *IdnLoader) loadFromFile(filename string) ([]string, error) {
	file, err := loader.fsys.Open(filename)
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
