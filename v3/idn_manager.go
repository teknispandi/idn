package v3

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
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

	// SingleScript: tolak label IDN yang mencampur script Unicode (ICANN
	// Guideline 15), kecuali kombinasi CJK yang lazim. Default mati karena
	// pengecualian Guideline 15 diatur oleh tabel: tabel Jepang, Chinese dan
	// Korea sengaja memuat a–z, jadi "日本abc" sah menurut tabelnya. Nyalakan
	// untuk zona yang ingin lebih ketat daripada tabelnya.
	SingleScript bool `json:"single_script"`

	// TableRefs: URL tabel di IANA Repository of IDN Practices dan URL
	// kebijakannya untuk setiap tabel, untuk deposit RDE. Kunci adalah nama
	// tabel. Jika kosong, header "# URL:"/"# Policy:" tabel dipakai — header
	// tabel Google di idn/ menunjuk ke registrasi Google, jadi registry lain
	// harus mengisi TableRefs dengan URL miliknya sendiri.
	TableRefs map[string]TableRef `json:"table_refs,omitempty"`
}

// TableRef adalah referensi publik sebuah tabel IDN untuk deposit RDE
// (elemen idnTableRef, RFC 9022).
type TableRef struct {
	ID     string `json:"id"`     // idnTableId, sama dengan nama tabel
	URL    string `json:"url"`    // URL tabel di IANA Repository of IDN Practices
	Policy string `json:"policy"` // URL kebijakan registrasi IDN registry
}

// IdnManager mengelola IDN tables untuk multiple TLD.
// Aman dipakai bersamaan dari banyak goroutine.
type IdnManager struct {
	loader     *IdnLoader
	tldConfigs map[string]*TldIdnConfig
	// tldTables terurut sesuai Languages. Slice tidak pernah diubah di
	// tempat (copy-on-write), jadi boleh dipakai setelah lock dilepas.
	tldTables map[string][]LabelTable
	// customTables berisi tabel non-.txt (mis. LGR v4) dari RegisterTable;
	// diutamakan di atas file loader dengan nama yang sama.
	customTables map[string]LabelTable
	mutex        sync.RWMutex
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
		loader:       loader,
		tldConfigs:   make(map[string]*TldIdnConfig),
		tldTables:    make(map[string][]LabelTable),
		customTables: make(map[string]LabelTable),
	}
}

// RegisterTable menyediakan tabel non-.txt (mis. *v4.LGR) dengan namanya,
// agar bisa dipakai di Languages pada konfigurasi TLD. Daftarkan tabel
// sebelum TLD yang memakainya.
func (manager *IdnManager) RegisterTable(table LabelTable) error {
	name := normalizeLanguageName(table.GetName())
	if name == "" {
		return fmt.Errorf("table has no name")
	}

	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	manager.customTables[name] = table
	return nil
}

// resolveTable mencari tabel dari RegisterTable lalu dari loader.
func (manager *IdnManager) resolveTable(language string) (LabelTable, error) {
	manager.mutex.RLock()
	table, ok := manager.customTables[language]
	manager.mutex.RUnlock()
	if ok {
		return table, nil
	}
	return manager.loader.LoadTable(language)
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
	tables := make([]LabelTable, 0, len(languages))
	for _, language := range languages {
		table, err := manager.resolveTable(language)
		if err != nil {
			return fmt.Errorf("failed to load tables for TLD %s: %w", zone, err)
		}
		tables = append(tables, table)
	}

	refs := make(map[string]TableRef, len(config.TableRefs))
	for name, ref := range config.TableRefs {
		refs[normalizeLanguageName(name)] = ref
	}

	config.TldName = zone
	config.Languages = languages
	config.TableRefs = refs

	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	manager.tldConfigs[zone] = &config
	manager.tldTables[zone] = tables

	return nil
}

// RegisterTldWithPreload mendaftarkan TLD dan preload semua bahasa
func (manager *IdnManager) RegisterTldWithPreload(tldName string, languages []string) error {
	// Preload languages ke cache; tabel dari RegisterTable sudah di memori
	for _, language := range languages {
		if _, err := manager.resolveTable(normalizeLanguageName(language)); err != nil {
			return fmt.Errorf("failed to preload languages: %w", err)
		}
	}

	return manager.RegisterTld(tldName, languages)
}

// lookup mengembalikan salinan konfigurasi dan tabel TLD.
func (manager *IdnManager) lookup(tldName string) (TldIdnConfig, []LabelTable, error) {
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
func (manager *IdnManager) enabledTld(tldName string) (TldIdnConfig, []LabelTable, error) {
	config, tables, err := manager.lookup(tldName)
	if err == nil && !config.Enabled {
		err = newLabelError(ErrTldDisabled, config.TldName)
	}
	return config, tables, err
}

// GetTldTables mengembalikan semua IDN tables format .txt untuk TLD tertentu.
// Tabel dari RegisterTable (mis. LGR) tidak termasuk; daftar lengkap nama
// tabel ada di GetTldConfig().Languages.
func (manager *IdnManager) GetTldTables(tldName string) (map[string]*IdnTable, error) {
	_, tables, err := manager.lookup(tldName)
	if err != nil {
		return nil, err
	}

	// Return copy untuk keamanan
	result := make(map[string]*IdnTable, len(tables))
	for _, table := range tables {
		if idnTable, ok := table.(*IdnTable); ok {
			result[idnTable.name] = idnTable
		}
	}

	return result, nil
}

// RdeTableRefs mengembalikan referensi semua tabel TLD (sesuai urutan
// konfigurasi) untuk deposit RDE. Error jika ada tabel tanpa URL atau
// Policy — deposit tidak boleh berisi referensi kosong atau karangan.
func (manager *IdnManager) RdeTableRefs(tldName string) ([]TableRef, error) {
	config, tables, err := manager.lookup(tldName)
	if err != nil {
		return nil, err
	}

	refs := make([]TableRef, 0, len(tables))
	var missing []string
	for _, table := range tables {
		name := table.GetName()
		ref, ok := config.TableRefs[name]
		if !ok {
			if headers, ok := table.(interface {
				GetURL() *url.URL
				GetPolicy() *url.URL
			}); ok {
				ref = TableRef{URL: urlOrEmpty(headers.GetURL()), Policy: urlOrEmpty(headers.GetPolicy())}
			}
		}
		ref.ID = name
		if ref.URL == "" || ref.Policy == "" {
			missing = append(missing, name)
			continue
		}
		refs = append(refs, ref)
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("TLD %s: no RDE URL/policy for tables %s; set TldIdnConfig.TableRefs", config.TldName, strings.Join(missing, ", "))
	}
	return refs, nil
}

// CheckRdeTableRefs memastikan setiap TLD aktif punya referensi RDE lengkap.
// Panggil saat registry mulai agar kekurangan ketahuan sebelum deposit.
func (manager *IdnManager) CheckRdeTableRefs() error {
	var errs []error
	for _, tld := range manager.ListRegisteredTlds() {
		config, err := manager.GetTldConfig(tld)
		if err != nil || !config.Enabled {
			continue
		}
		if _, err := manager.RdeTableRefs(tld); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func urlOrEmpty(u *url.URL) string {
	if u == nil {
		return ""
	}
	return u.String()
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
	if err != nil || checkScriptPolicy(config, p) != nil {
		return false, nil, nil
	}

	var validLanguages []string
	for _, table := range tables {
		if table.CheckLabel(p.ulabel) == nil {
			validLanguages = append(validLanguages, table.GetName())
		}
	}

	isValid := len(validLanguages) > 0 || (!p.isIDN && config.AllowASCII)
	return isValid, validLanguages, nil
}

// ValidateLabelForLanguage memvalidasi label untuk bahasa tertentu dalam TLD
func (manager *IdnManager) ValidateLabelForLanguage(tldName, language, label string) (bool, error) {
	config, tables, err := manager.enabledTld(tldName)
	if err != nil {
		return false, err
	}

	language = normalizeLanguageName(language)
	for _, table := range tables {
		if table.GetName() == language {
			p, err := prepareLabel(label)
			if err != nil || checkScriptPolicy(config, p) != nil {
				return false, nil
			}
			return table.CheckLabel(p.ulabel) == nil, nil
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
	if err := checkScriptPolicy(config, p); err != nil {
		return nil, err
	}
	return matchTables(p, tables, config.AllowASCII, "")
}

// checkScriptPolicy menerapkan TldIdnConfig.SingleScript.
func checkScriptPolicy(config TldIdnConfig, p preparedLabel) error {
	if !config.SingleScript || !p.isIDN {
		return nil
	}
	runes := []rune(p.ulabel)
	if i := mixedScriptAt(runes); i >= 0 {
		e := newRuneError(ErrMixedScript, p.ulabel, i, runes[i])
		e.Rule = "ICANN IDN Guideline 15"
		return e
	}
	return nil
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
	if err := checkScriptPolicy(config, labels[0]); err != nil {
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
	refs := make(map[string]TableRef, len(config.TableRefs))
	for name, ref := range config.TableRefs {
		refs[name] = ref
	}
	config.TableRefs = refs
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
	table, err := manager.resolveTable(language)
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
	manager.tldTables[zone] = append(append([]LabelTable(nil), manager.tldTables[zone]...), table)

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
	var newTables []LabelTable
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
