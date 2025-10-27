package v3

import (
	"fmt"
	"sync"
)

// TldIdnConfig konfigurasi IDN untuk TLD tertentu
type TldIdnConfig struct {
	TldName   string   `json:"tld_name"`
	Languages []string `json:"languages"`
	Enabled   bool     `json:"enabled"`
}

// IdnManager mengelola IDN tables untuk multiple TLD
type IdnManager struct {
	loader     *IdnLoader
	tldConfigs map[string]*TldIdnConfig
	tldTables  map[string]map[string]*IdnTable
	mutex      sync.RWMutex
}

// NewIdnManager membuat instance baru IdnManager
func NewIdnManager(idnPath string) *IdnManager {
	return &IdnManager{
		loader:     NewIdnLoader(idnPath),
		tldConfigs: make(map[string]*TldIdnConfig),
		tldTables:  make(map[string]map[string]*IdnTable),
	}
}

// RegisterTld mendaftarkan TLD dengan konfigurasi IDN
func (manager *IdnManager) RegisterTld(tldName string, languages []string) error {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	config := &TldIdnConfig{
		TldName:   tldName,
		Languages: languages,
		Enabled:   true,
	}

	// Load tables untuk bahasa yang diperlukan
	tables, err := manager.loader.LoadMultipleTables(languages)
	if err != nil {
		return fmt.Errorf("failed to load tables for TLD %s: %w", tldName, err)
	}

	manager.tldConfigs[tldName] = config
	manager.tldTables[tldName] = tables

	return nil
}

// RegisterTldWithPreload mendaftarkan TLD dan preload semua bahasa
func (manager *IdnManager) RegisterTldWithPreload(tldName string, languages []string) error {
	// Preload languages ke cache
	err := manager.loader.PreloadLanguages(languages)
	if err != nil {
		return fmt.Errorf("failed to preload languages: %w", err)
	}

	return manager.RegisterTld(tldName, languages)
}

// GetTldTables mengembalikan semua IDN tables untuk TLD tertentu
func (manager *IdnManager) GetTldTables(tldName string) (map[string]*IdnTable, error) {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()

	tables, exists := manager.tldTables[tldName]
	if !exists {
		return nil, fmt.Errorf("TLD %s not registered", tldName)
	}

	// Return copy untuk keamanan
	result := make(map[string]*IdnTable)
	for k, v := range tables {
		result[k] = v
	}

	return result, nil
}

// ValidateLabel memvalidasi label untuk TLD tertentu
func (manager *IdnManager) ValidateLabel(tldName, label string) (bool, []string, error) {
	tables, err := manager.GetTldTables(tldName)
	if err != nil {
		return false, nil, err
	}

	var validLanguages []string
	isValid := false

	for language, table := range tables {
		if table.IsValidLabel(label) {
			validLanguages = append(validLanguages, language)
			isValid = true
		}
	}

	return isValid, validLanguages, nil
}

// ValidateLabelForLanguage memvalidasi label untuk bahasa tertentu dalam TLD
func (manager *IdnManager) ValidateLabelForLanguage(tldName, language, label string) (bool, error) {
	tables, err := manager.GetTldTables(tldName)
	if err != nil {
		return false, err
	}

	table, exists := tables[language]
	if !exists {
		return false, fmt.Errorf("language %s not supported for TLD %s", language, tldName)
	}

	return table.IsValidLabel(label), nil
}

// GetTldConfig mengembalikan konfigurasi untuk TLD
func (manager *IdnManager) GetTldConfig(tldName string) (*TldIdnConfig, error) {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()

	config, exists := manager.tldConfigs[tldName]
	if !exists {
		return nil, fmt.Errorf("TLD %s not registered", tldName)
	}

	// Return copy untuk keamanan
	return &TldIdnConfig{
		TldName:   config.TldName,
		Languages: append([]string(nil), config.Languages...),
		Enabled:   config.Enabled,
	}, nil
}

// ListRegisteredTlds mengembalikan daftar TLD yang terdaftar
func (manager *IdnManager) ListRegisteredTlds() []string {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()

	var tlds []string
	for tld := range manager.tldConfigs {
		tlds = append(tlds, tld)
	}

	return tlds
}

// EnableTld mengaktifkan IDN untuk TLD
func (manager *IdnManager) EnableTld(tldName string) error {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	config, exists := manager.tldConfigs[tldName]
	if !exists {
		return fmt.Errorf("TLD %s not registered", tldName)
	}

	config.Enabled = true
	return nil
}

// DisableTld menonaktifkan IDN untuk TLD
func (manager *IdnManager) DisableTld(tldName string) error {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	config, exists := manager.tldConfigs[tldName]
	if !exists {
		return fmt.Errorf("TLD %s not registered", tldName)
	}

	config.Enabled = false
	return nil
}

// AddLanguageToTld menambahkan bahasa ke TLD yang sudah ada
func (manager *IdnManager) AddLanguageToTld(tldName, language string) error {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	config, exists := manager.tldConfigs[tldName]
	if !exists {
		return fmt.Errorf("TLD %s not registered", tldName)
	}

	// Check jika bahasa sudah ada
	for _, lang := range config.Languages {
		if lang == language {
			return nil // sudah ada
		}
	}

	// Load table untuk bahasa baru
	table, err := manager.loader.LoadTable(language)
	if err != nil {
		return fmt.Errorf("failed to load table for language %s: %w", language, err)
	}

	// Update config dan tables
	config.Languages = append(config.Languages, language)
	manager.tldTables[tldName][language] = table

	return nil
}

// RemoveLanguageFromTld menghapus bahasa dari TLD
func (manager *IdnManager) RemoveLanguageFromTld(tldName, language string) error {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	config, exists := manager.tldConfigs[tldName]
	if !exists {
		return fmt.Errorf("TLD %s not registered", tldName)
	}

	// Hapus dari config
	var newLanguages []string
	found := false
	for _, lang := range config.Languages {
		if lang != language {
			newLanguages = append(newLanguages, lang)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("language %s not found for TLD %s", language, tldName)
	}

	config.Languages = newLanguages

	// Hapus dari tables
	delete(manager.tldTables[tldName], language)

	return nil
}

// GetStats mengembalikan statistik manager
func (manager *IdnManager) GetStats() map[string]interface{} {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()

	stats := map[string]interface{}{
		"registered_tlds": len(manager.tldConfigs),
		"cache_stats":     manager.loader.GetCacheStats(),
		"tld_details":     make(map[string]interface{}),
	}

	tldDetails := make(map[string]interface{})
	for tld, config := range manager.tldConfigs {
		tldDetails[tld] = map[string]interface{}{
			"languages": config.Languages,
			"enabled":   config.Enabled,
			"tables":    len(manager.tldTables[tld]),
		}
	}
	stats["tld_details"] = tldDetails

	return stats
}

// UnregisterTld menghapus TLD dari manager
func (manager *IdnManager) UnregisterTld(tldName string) error {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	_, exists := manager.tldConfigs[tldName]
	if !exists {
		return fmt.Errorf("TLD %s not registered", tldName)
	}

	delete(manager.tldConfigs, tldName)
	delete(manager.tldTables, tldName)

	return nil
}
