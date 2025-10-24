package idn

import (
	"fmt"
	"net/url"
)

// TldInterface mendefinisikan interface untuk objek TLD
// Ini adalah abstraksi untuk menghindari ketergantungan langsung pada implementasi TLD
type TldInterface interface {
	// GetIdnTables mengembalikan set tabel IDN yang dikonfigurasi untuk TLD ini
	GetIdnTables() []IdnTableType
}

// IdnLabelValidator memvalidasi apakah label IDN tertentu dapat diprovisi untuk TLD tertentu
type IdnLabelValidator struct {
	registry *IdnTableRegistry
}

// NewIdnLabelValidator membuat instance baru dari validator label IDN
func NewIdnLabelValidator() *IdnLabelValidator {
	return &IdnLabelValidator{
		registry: NewIdnTableRegistry(),
	}
}

// defaultIdnTables adalah tabel IDN default yang akan digunakan oleh sebagian besar TLD
var defaultIdnTables = []IdnTableType{ExtendedLatin, Japanese}

// FindValidIdnTableForTld mengembalikan nama tabel IDN pertama yang cocok jika label domain valid untuk TLD yang diberikan.
//
// Label valid jika dianggap valid oleh setidaknya satu tabel IDN yang dikonfigurasi untuk TLD tersebut.
// Jika tidak ada kecocokan yang ditemukan, string kosong dan false dikembalikan.
//
// Parameter:
// - label: label domain yang akan divalidasi
// - tld: objek TLD yang mengimplementasikan TldInterface
//
// Return:
// - string: nama tabel IDN yang valid (jika ditemukan)
// - bool: true jika label valid, false jika tidak
func (v *IdnLabelValidator) FindValidIdnTableForTld(label string, tld TldInterface) (string, bool) {
	// Konversi label ke Unicode jika diperlukan
	// Dalam implementasi ini, kita asumsikan label sudah dalam format Unicode
	unicodeString := label

	// Dapatkan tabel IDN untuk TLD
	idnTables := v.getIdnTablesForTld(tld)

	// Cek setiap tabel IDN
	for _, tableType := range idnTables {
		table := v.registry.GetTable(tableType)
		if table != nil && table.IsValidLabel(unicodeString) {
			return table.GetName(), true
		}
	}

	return "", false
}

// getIdnTablesForTld mengembalikan tabel IDN yang didukung oleh TLD.
// Jika TLD tidak memiliki tabel IDN yang dikonfigurasi, maka tabel default akan digunakan.
func (v *IdnLabelValidator) getIdnTablesForTld(tld TldInterface) []IdnTableType {
	idnTablesForTld := tld.GetIdnTables()
	if len(idnTablesForTld) == 0 {
		return defaultIdnTables
	}
	return idnTablesForTld
}

// GetIdnTablesForTld mengembalikan nama-nama tabel IDN yang didukung oleh TLD
func (v *IdnLabelValidator) GetIdnTablesForTld(tld TldInterface) []string {
	tableTypes := v.getIdnTablesForTld(tld)
	var tableNames []string

	for _, tableType := range tableTypes {
		table := v.registry.GetTable(tableType)
		if table != nil {
			tableNames = append(tableNames, table.GetName())
		}
	}

	return tableNames
}

// ValidateLabel memvalidasi label terhadap tabel IDN tertentu
func (v *IdnLabelValidator) ValidateLabel(label string, tableType IdnTableType) bool {
	table := v.registry.GetTable(tableType)
	if table == nil {
		return false
	}
	return table.IsValidLabel(label)
}

// GetTableInfo mengembalikan informasi tentang tabel IDN tertentu
func (v *IdnLabelValidator) GetTableInfo(tableType IdnTableType) (*TableInfo, error) {
	table := v.registry.GetTable(tableType)
	if table == nil {
		return nil, fmt.Errorf("table type %s not found", tableType.String())
	}

	return &TableInfo{
		Name:   table.GetName(),
		URL:    table.GetURL(),
		Policy: table.GetPolicy(),
		Type:   tableType,
	}, nil
}

// TableInfo berisi informasi metadata tentang tabel IDN
type TableInfo struct {
	Name   string       // Nama tabel
	URL    *url.URL     // URL referensi tabel
	Policy *url.URL     // URL kebijakan tabel
	Type   IdnTableType // Jenis tabel
}

// String mengembalikan representasi string dari informasi tabel
func (ti *TableInfo) String() string {
	return fmt.Sprintf("Table: %s, Type: %s, URL: %s, Policy: %s",
		ti.Name, ti.Type.String(), ti.URL.String(), ti.Policy.String())
}
