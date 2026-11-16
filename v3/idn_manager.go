package v3

import (
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"sync"
)

// TldIdnConfig konfigurasi IDN untuk TLD tertentu
type TldIdnConfig struct {
	// TldName adalah zona tempat nama didaftarkan: "id", "co.id", atau zona
	// IDN. Disimpan dalam bentuk A-label huruf kecil.
	TldName string `json:"tld_name"`

	// Languages adalah nama tabel IDN, terurut menurut prioritas: tabel
	// pertama yang menerima label adalah tabel yang dicatat.
	Languages []string `json:"languages"`

	// Enabled: jika false, semua validasi untuk TLD ini ditolak dengan ErrTldDisabled.
	Enabled bool `json:"enabled"`

	// AllowASCII: label LDH murni (a-z, 0-9, '-') boleh didaftarkan tanpa
	// harus diterima tabel IDN. Matikan untuk zona khusus IDN (mis. .рф).
	AllowASCII bool `json:"allow_ascii"`
}

// IdnManager mengelola IDN tables untuk multiple TLD.
// Aman dipakai bersamaan dari banyak goroutine.
type IdnManager struct {
	loader     *IdnLoader
	tldConfigs map[string]*TldIdnConfig
	// tldTables terurut sesuai Languages. Slice tidak pernah diubah di
	// tempat (copy-on-write), jadi boleh dipakai setelah lock dilepas.
	tldTables map[string][]*IdnTable
	mutex     sync.RWMutex
}

// NewIdnManager membuat instance baru IdnManager yang membaca tabel dari folder idnPath
func NewIdnManager(idnPath string) *IdnManager {
	return newIdnManager(NewIdnLoader(idnPath))
}

// NewIdnManagerFS membuat IdnManager yang membaca tabel dari fs.FS
// (mis. idn.Tables() untuk tabel yang disematkan di binary)
func NewIdnManagerFS(fsys fs.FS) *IdnManager {
	return newIdnManager(NewIdnLoaderFS(fsys))
}

func newIdnManager(loader *IdnLoader) *IdnManager {
	return &IdnManager{
		loader:     loader,
		tldConfigs: make(map[string]*TldIdnConfig),
		tldTables:  make(map[string][]*IdnTable),
	}
}

// RegisterTld mendaftarkan TLD dengan konfigurasi IDN. TLD langsung aktif
// dan label LDH murni diizinkan; gunakan RegisterTldConfig untuk mengatur
// keduanya.
func (manager *IdnManager) RegisterTld(tldName string, languages []string) error {
	return manager.RegisterTldConfig(TldIdnConfig{
		TldName:    tldName,
		Languages:  languages,
		Enabled:    true,
		AllowASCII: true,
	})
}

// RegisterTldConfig mendaftarkan (atau mengganti) konfigurasi IDN sebuah TLD.
func (manager *IdnManager) RegisterTldConfig(config TldIdnConfig) error {
	zone, err := normalizeZone(config.TldName)
	if err != nil {
		return fmt.Errorf("invalid TLD name %q: %w", config.TldName, err)
	}

	var languages []string
	seen := make(map[string]bool)
	for _, language := range config.Languages {
		language = normalizeLanguageName(language)
		if !seen[language] {
			seen[language] = true
			languages = append(languages, language)
		}
	}

	// Load tables untuk bahasa yang diperlukan (loader punya lock sendiri)
	tables := make([]*IdnTable, 0, len(languages))
	for _, language := range languages {
		table, err := manager.loader.LoadTable(language)
		if err != nil {
			return fmt.Errorf("failed to load tables for TLD %s: %w", zone, err)
		}
		tables = append(tables, table)
	}

	config.TldName = zone
	config.Languages = languages

	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	manager.tldConfigs[zone] = &config
	manager.tldTables[zone] = tables

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

// lookup mengembalikan salinan konfigurasi dan tabel TLD.
func (manager *IdnManager) lookup(tldName string) (TldIdnConfig, []*IdnTable, error) {
	zone, err := normalizeZone(tldName)
	if err != nil {
		return TldIdnConfig{}, nil, &ValidationError{Kind: ErrUnknownTld, Label: tldName, Pos: -1, Err: err}
	}

	manager.mutex.RLock()
	defer manager.mutex.RUnlock()

	config, exists := manager.tldConfigs[zone]
	if !exists {
		return TldIdnConfig{}, nil, newLabelError(ErrUnknownTld, zone)
	}
	return *config, manager.tldTables[zone], nil
}

// enabledTld seperti lookup, tetapi menolak TLD yang dinonaktifkan.
func (manager *IdnManager) enabledTld(tldName string) (TldIdnConfig, []*IdnTable, error) {
	config, tables, err := manager.lookup(tldName)
	if err == nil && !config.Enabled {
		err = newLabelError(ErrTldDisabled, config.TldName)
	}
	return config, tables, err
}

// GetTldTables mengembalikan semua IDN tables untuk TLD tertentu
func (manager *IdnManager) GetTldTables(tldName string) (map[string]*IdnTable, error) {
	_, tables, err := manager.lookup(tldName)
	if err != nil {
		return nil, err
	}

	// Return copy untuk keamanan
	result := make(map[string]*IdnTable, len(tables))
	for _, table := range tables {
		result[table.name] = table
	}

	return result, nil
}

// ValidateLabel memvalidasi label untuk TLD tertentu. Label boleh U-label
// atau A-label. Mengembalikan semua tabel yang menerima label, sesuai urutan
// konfigurasi. Error hanya untuk masalah TLD (tidak terdaftar/nonaktif);
// alasan label ditolak tersedia lewat CheckLabel.
func (manager *IdnManager) ValidateLabel(tldName, label string) (bool, []string, error) {
	config, tables, err := manager.enabledTld(tldName)
	if err != nil {
		return false, nil, err
	}

	p, err := prepareLabel(label)
	if err != nil {
		return false, nil, nil
	}

	var validLanguages []string
	for _, table := range tables {
		if table.IsValidLabel(p.ulabel) {
			validLanguages = append(validLanguages, table.name)
		}
	}

	isValid := len(validLanguages) > 0 || (!p.isIDN && config.AllowASCII)
	return isValid, validLanguages, nil
}

// ValidateLabelForLanguage memvalidasi label untuk bahasa tertentu dalam TLD
func (manager *IdnManager) ValidateLabelForLanguage(tldName, language, label string) (bool, error) {
	_, tables, err := manager.enabledTld(tldName)
	if err != nil {
		return false, err
	}

	language = normalizeLanguageName(language)
	for _, table := range tables {
		if table.name == language {
			p, err := prepareLabel(label)
			if err != nil {
				return false, nil
			}
			return table.IsValidLabel(p.ulabel), nil
		}
	}

	return false, fmt.Errorf("language %s not supported for TLD %s", language, tldName)
}

// CheckLabel menjalankan validasi lengkap satu label untuk TLD: struktur
// label, IDNA2008, lalu tabel IDN TLD. Label boleh U-label atau A-label.
// Jika ditolak, error berupa *ValidationError (cek dengan errors.Is).
func (manager *IdnManager) CheckLabel(tldName, label string) (*LabelResult, error) {
	config, tables, err := manager.enabledTld(tldName)
	if err != nil {
		return nil, err
	}

	p, err := prepareLabel(label)
	if err != nil {
		return nil, err
	}
	return matchTables(p, tables, config.AllowASCII, "")
}

// CheckDomain memvalidasi nama domain yang akan didaftarkan, mis.
// "café.co.id" atau "xn--caf-dma.co.id". Zona dicari dengan kecocokan
// akhiran terpanjang di antara TLD terdaftar, dan nama harus tepat satu
// label di bawah zona itu.
//
// Registry menyimpan DomainResult.ASCIIName dan mencatat TableName() untuk
// RDE. Registrar sebaiknya memanggil IdnUtils.NormalizeDomainName lebih
// dulu untuk input dari pengguna.
func (manager *IdnManager) CheckDomain(domain string) (*DomainResult, error) {
	return manager.checkDomain(domain, "")
}

// CheckDomainForTable seperti CheckDomain, tetapi label hanya boleh
// diterima oleh tabel bernama table — untuk registrar yang menyebut tabel
// IDN secara eksplisit (ekstensi EPP IDN).
func (manager *IdnManager) CheckDomainForTable(domain, table string) (*DomainResult, error) {
	return manager.checkDomain(domain, normalizeLanguageName(table))
}

func (manager *IdnManager) checkDomain(domain, only string) (*DomainResult, error) {
	labels, err := prepareDomain(asciiLower(strings.TrimSpace(domain)))
	if err != nil {
		return nil, err
	}
	alabels := make([]string, len(labels))
	ulabels := make([]string, len(labels))
	for i, p := range labels {
		alabels[i] = p.alabel
		ulabels[i] = p.ulabel
	}
	asciiName := strings.Join(alabels, ".")

	// Akhiran terpanjang yang terdaftar sebagai zona.
	zoneIdx := -1
	manager.mutex.RLock()
	_, isZone := manager.tldConfigs[asciiName]
	for i := 1; i < len(labels) && zoneIdx < 0; i++ {
		if _, ok := manager.tldConfigs[strings.Join(alabels[i:], ".")]; ok {
			zoneIdx = i
		}
	}
	manager.mutex.RUnlock()

	switch {
	case isZone:
		e := newLabelError(ErrNotRegistrable, asciiName)
		e.Rule = "name is itself a registered zone"
		return nil, e
	case zoneIdx < 0:
		return nil, newLabelError(ErrUnknownTld, asciiName)
	case zoneIdx > 1:
		e := newLabelError(ErrNotRegistrable, asciiName)
		e.Rule = "zone " + strings.Join(alabels[zoneIdx:], ".")
		return nil, e
	}

	zone := strings.Join(alabels[1:], ".")
	config, tables, err := manager.enabledTld(zone)
	if err != nil {
		return nil, err
	}
	res, err := matchTables(labels[0], tables, config.AllowASCII, only)
	if err != nil {
		return nil, err
	}

	return &DomainResult{
		LabelResult: *res,
		Zone:        zone,
		ASCIIName:   asciiName,
		UnicodeName: strings.Join(ulabels, "."),
	}, nil
}

// GetTldConfig mengembalikan konfigurasi untuk TLD
func (manager *IdnManager) GetTldConfig(tldName string) (*TldIdnConfig, error) {
	config, _, err := manager.lookup(tldName)
	if err != nil {
		return nil, err
	}

	// Return copy untuk keamanan
	config.Languages = append([]string(nil), config.Languages...)
	return &config, nil
}

// ListRegisteredTlds mengembalikan daftar TLD yang terdaftar, terurut
func (manager *IdnManager) ListRegisteredTlds() []string {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()

	tlds := make([]string, 0, len(manager.tldConfigs))
	for tld := range manager.tldConfigs {
		tlds = append(tlds, tld)
	}
	sort.Strings(tlds)

	return tlds
}

// EnableTld mengaktifkan IDN untuk TLD
func (manager *IdnManager) EnableTld(tldName string) error {
	return manager.setEnabled(tldName, true)
}

// DisableTld menonaktifkan IDN untuk TLD; validasi untuk TLD ini akan
// ditolak dengan ErrTldDisabled sampai EnableTld dipanggil
func (manager *IdnManager) DisableTld(tldName string) error {
	return manager.setEnabled(tldName, false)
}

func (manager *IdnManager) setEnabled(tldName string, enabled bool) error {
	zone, _ := normalizeZone(tldName)

	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	config, exists := manager.tldConfigs[zone]
	if !exists {
		return fmt.Errorf("TLD %s not registered", tldName)
	}

	config.Enabled = enabled
	return nil
}

// AddLanguageToTld menambahkan bahasa ke TLD yang sudah ada (prioritas terakhir)
func (manager *IdnManager) AddLanguageToTld(tldName, language string) error {
	zone, _ := normalizeZone(tldName)
	language = normalizeLanguageName(language)

	// Load table untuk bahasa baru (di luar lock manager)
	table, err := manager.loader.LoadTable(language)
	if err != nil {
		return fmt.Errorf("failed to load table for language %s: %w", language, err)
	}

	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	config, exists := manager.tldConfigs[zone]
	if !exists {
		return fmt.Errorf("TLD %s not registered", tldName)
	}

	// Check jika bahasa sudah ada
	for _, lang := range config.Languages {
		if lang == language {
			return nil // sudah ada
		}
	}

	// Update config dan tables (copy-on-write)
	config.Languages = append(append([]string(nil), config.Languages...), language)
	manager.tldTables[zone] = append(append([]*IdnTable(nil), manager.tldTables[zone]...), table)

	return nil
}

// RemoveLanguageFromTld menghapus bahasa dari TLD
func (manager *IdnManager) RemoveLanguageFromTld(tldName, language string) error {
	zone, _ := normalizeZone(tldName)
	language = normalizeLanguageName(language)

	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	config, exists := manager.tldConfigs[zone]
	if !exists {
		return fmt.Errorf("TLD %s not registered", tldName)
	}

	// Hapus dari config dan tables (copy-on-write)
	var newLanguages []string
	var newTables []*IdnTable
	for i, lang := range config.Languages {
		if lang != language {
			newLanguages = append(newLanguages, lang)
			newTables = append(newTables, manager.tldTables[zone][i])
		}
	}

	if len(newLanguages) == len(config.Languages) {
		return fmt.Errorf("language %s not found for TLD %s", language, tldName)
	}

	config.Languages = newLanguages
	manager.tldTables[zone] = newTables

	return nil
}

// GetStats mengembalikan statistik manager
func (manager *IdnManager) GetStats() map[string]interface{} {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()

	tldDetails := make(map[string]interface{})
	for tld, config := range manager.tldConfigs {
		tldDetails[tld] = map[string]interface{}{
			"languages":   config.Languages,
			"enabled":     config.Enabled,
			"allow_ascii": config.AllowASCII,
			"tables":      len(manager.tldTables[tld]),
		}
	}

	return map[string]interface{}{
		"registered_tlds": len(manager.tldConfigs),
		"cache_stats":     manager.loader.GetCacheStats(),
		"tld_details":     tldDetails,
	}
}

// UnregisterTld menghapus TLD dari manager
func (manager *IdnManager) UnregisterTld(tldName string) error {
	zone, _ := normalizeZone(tldName)

	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	_, exists := manager.tldConfigs[zone]
	if !exists {
		return fmt.Errorf("TLD %s not registered", tldName)
	}

	delete(manager.tldConfigs, zone)
	delete(manager.tldTables, zone)

	return nil
}

// normalizeZone mengubah nama zona ke bentuk A-label huruf kecil tanpa titik
// di akhir, sehingga "ID", "id." dan "id" merujuk zona yang sama.
func normalizeZone(name string) (string, error) {
	labels, err := prepareDomain(strings.TrimSpace(name))
	if err != nil {
		return "", err
	}
	alabels := make([]string, len(labels))
	for i, p := range labels {
		alabels[i] = p.alabel
	}
	return strings.Join(alabels, "."), nil
}
