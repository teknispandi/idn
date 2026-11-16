package v3

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
	res, err := v.CheckLabelForTld(label, tld)
	if err != nil {
		return "", false
	}
	return res.TableName(), true
}

// CheckLabelForTld memvalidasi label secara lengkap untuk TLD: struktur
// label, IDNA2008 (A-label didekode dulu, seperti Idn.toUnicode di Nomulus),
// lalu tabel IDN TLD. Seperti Nomulus, label ASCII pun harus diterima salah
// satu tabel. Jika ditolak, error berupa *ValidationError.
func (v *IdnLabelValidator) CheckLabelForTld(label string, tld TldInterface) (*LabelResult, error) {
	p, err := prepareLabel(label)
	if err != nil {
		return nil, err
	}

	var tables []LabelTable
	for _, tableType := range v.getIdnTablesForTld(tld) {
		if table := v.registry.GetTable(tableType); table != nil {
			tables = append(tables, table)
		}
	}
	return matchTables(p, tables, false, "")
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

// ValidateLabel memvalidasi label (U-label atau A-label) terhadap tabel IDN tertentu
func (v *IdnLabelValidator) ValidateLabel(label string, tableType IdnTableType) bool {
	table := v.registry.GetTable(tableType)
	if table == nil {
		return false
	}
	p, err := prepareLabel(label)
	if err != nil {
		return false
	}
	return table.IsValidLabel(p.ulabel)
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
		ti.Name, ti.Type.String(), urlString(ti.URL), urlString(ti.Policy))
}

func urlString(u *url.URL) string {
	if u == nil {
		return "-"
	}
	return u.String()
}
