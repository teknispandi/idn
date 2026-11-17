package v3

import (
	"fmt"
	"strings"
)

// IdnTableType merepresentasikan jenis tabel IDN yang tersedia
type IdnTableType int

const (
	// ExtendedLatin adalah Extended Latin, sebagaimana digunakan pada peluncuran TLD yang ada sebelum 2023.
	// Pada tahun 2023 tabel ini tidak lagi sesuai dengan kebijakan IDN ICANN untuk peluncuran baru,
	// sehingga dipertahankan semata-mata untuk kompatibilitas legacy dengan TLD yang sudah diluncurkan.
	ExtendedLatin IdnTableType = iota

	// UnconfusableLatin adalah Extended Latin, tetapi dengan karakter yang membingungkan dihapus.
	// Ini kompatibel dengan persyaratan ICANN pada tahun 2023, dan digunakan untuk TLD Dads dan Grads
	// dan semua peluncuran TLD berikutnya.
	UnconfusableLatin

	// Japanese adalah bahasa Jepang, sebagaimana digunakan pada peluncuran TLD yang ada sebelum 2023.
	// Pada tahun 2023 tabel ini tidak lagi sesuai dengan kebijakan IDN ICANN untuk peluncuran baru,
	// sehingga dipertahankan semata-mata untuk kompatibilitas legacy dengan TLD yang sudah diluncurkan.
	Japanese

	// AugmentedLatin adalah Latin dari LGR ICANN 2025 dengan karakter yang
	// membingungkan dihapus (google_latn_3.0, AUGMENTED_LATIN di Nomulus).
	// Ditambahkan di akhir agar nilai numerik jenis yang sudah ada tidak berubah.
	AugmentedLatin
)

// IdnTableRegistry mengelola registry dari semua tabel IDN yang tersedia
type IdnTableRegistry struct {
	tables map[IdnTableType]*IdnTable
}

// NewIdnTableRegistry membuat instance baru dari registry tabel IDN
func NewIdnTableRegistry() *IdnTableRegistry {
	registry := &IdnTableRegistry{
		tables: make(map[IdnTableType]*IdnTable),
	}

	// Muat semua tabel IDN
	registry.loadTables()

	return registry
}

// loadTables memuat semua tabel IDN dari file yang disematkan
func (r *IdnTableRegistry) loadTables() {
	tableConfigs := map[IdnTableType]struct {
		filename string
		language string
	}{
		ExtendedLatin:     {"extended_latin.txt", "extended_latin"},
		UnconfusableLatin: {"unconfusable_latin.txt", "unconfusable_latin"},
		Japanese:          {"japanese.txt", "ja"},
		AugmentedLatin:    {"augmented_latin.txt", "augmented_latin"},
	}

	for tableType, config := range tableConfigs {
		table, err := r.loadTable(config.language, config.filename)
		if err != nil {
			// Dalam implementasi produksi, ini harus ditangani dengan lebih baik
			panic(fmt.Sprintf("Failed to load IDN table %s: %v", config.filename, err))
		}
		r.tables[tableType] = table
	}
}

// loadTable memuat satu tabel IDN dari file
func (r *IdnTableRegistry) loadTable(tableName, filename string) (*IdnTable, error) {
	data, err := loadTableFromFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read table file %s: %w", filename, err)
	}

	validator := GetLanguageValidator(tableName)
	return NewIdnTable(tableName, data, validator)
}

// GetTable mengembalikan tabel IDN untuk jenis yang diberikan
func (r *IdnTableRegistry) GetTable(tableType IdnTableType) *IdnTable {
	return r.tables[tableType]
}

// GetAllTables mengembalikan semua tabel IDN yang tersedia
func (r *IdnTableRegistry) GetAllTables() map[IdnTableType]*IdnTable {
	result := make(map[IdnTableType]*IdnTable)
	for k, v := range r.tables {
		result[k] = v
	}
	return result
}

// String mengembalikan representasi string dari jenis tabel IDN
func (t IdnTableType) String() string {
	switch t {
	case ExtendedLatin:
		return "extended_latin"
	case UnconfusableLatin:
		return "unconfusable_latin"
	case Japanese:
		return "ja"
	case AugmentedLatin:
		return "augmented_latin"
	default:
		return "unknown"
	}
}

// ParseIdnTableType mengurai string menjadi IdnTableType
func ParseIdnTableType(s string) (IdnTableType, error) {
	switch strings.ToLower(s) {
	case "extended_latin":
		return ExtendedLatin, nil
	case "unconfusable_latin":
		return UnconfusableLatin, nil
	case "ja", "japanese":
		return Japanese, nil
	case "augmented_latin":
		return AugmentedLatin, nil
	default:
		return 0, fmt.Errorf("unknown IDN table type: %s", s)
	}
}
