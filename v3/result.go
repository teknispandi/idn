package v3

// LabelResult adalah hasil validasi satu label yang lolos.
type LabelResult struct {
	Input  string // label apa adanya dari pemanggil
	ULabel string // bentuk Unicode — untuk ditampilkan ke pengguna
	ALabel string // bentuk ASCII (punycode) — untuk disimpan registry dan dipakai di EPP/DNS
	IsIDN  bool   // true jika label mengandung code point non-ASCII

	// Table adalah tabel IDN yang menerima label; registry mencatatnya untuk
	// deposit RDE (idnTableId). Nil untuk label LDH yang diizinkan tanpa
	// tabel (TldIdnConfig.AllowASCII).
	Table *IdnTable
}

// TableName mengembalikan nama tabel IDN yang menerima label, atau "" jika tidak ada
func (r *LabelResult) TableName() string {
	if r.Table == nil {
		return ""
	}
	return r.Table.GetName()
}

// DomainResult adalah hasil validasi nama domain yang dapat didaftarkan.
type DomainResult struct {
	LabelResult        // label yang didaftarkan (label paling kiri)
	Zone        string // zona terdaftar dalam bentuk A-label, mis. "co.id"
	ASCIIName   string // nama lengkap bentuk A-label, mis. "xn--caf-dma.co.id"
	UnicodeName string // nama lengkap bentuk Unicode, mis. "café.co.id"
}

// matchTables mencari tabel pertama (sesuai urutan konfigurasi) yang
// menerima label. Urutan menentukan tabel yang dicatat, jadi hasilnya
// deterministik.
//
// Jika only tidak kosong, hanya tabel bernama only yang dicoba (mis. tabel
// yang diminta registrar lewat ekstensi EPP IDN). Label LDH lolos tanpa
// tabel jika allowASCII dan only kosong.
//
// Jika tidak ada yang menerima, kesalahan dari tabel yang "paling dekat"
// dikembalikan — tabel yang menolak paling belakang dalam label — karena
// itulah alasan yang paling berguna bagi pengguna.
func matchTables(p preparedLabel, tables []*IdnTable, allowASCII bool, only string) (*LabelResult, error) {
	res := &LabelResult{Input: p.input, ULabel: p.ulabel, ALabel: p.alabel, IsIDN: p.isIDN}
	if !p.isIDN && allowASCII && only == "" {
		return res, nil
	}

	var best *ValidationError
	tried := false
	for _, table := range tables {
		if only != "" && table.name != only {
			continue
		}
		tried = true
		err := table.CheckLabel(p.ulabel)
		if err == nil {
			res.Table = table
			return res, nil
		}
		ve, ok := err.(*ValidationError)
		if !ok {
			ve = &ValidationError{Kind: ErrLanguageRule, Label: p.ulabel, Pos: -1, Table: table.name, Err: err}
		}
		if best == nil || closeness(ve) > closeness(best) {
			best = ve
		}
	}

	if !tried {
		e := newLabelError(ErrNoIdnTable, p.ulabel)
		if only != "" {
			e.Table = only
			e.Rule = "table is not configured for this TLD"
		}
		return nil, e
	}
	return nil, best
}

// closeness: seberapa jauh label lolos sebelum ditolak. Pelanggaran aturan
// bahasa tanpa posisi berarti semua code point sudah ada di tabel.
func closeness(e *ValidationError) int {
	if e.Pos < 0 && e.Kind == ErrLanguageRule {
		return 1 << 30
	}
	return e.Pos
}
