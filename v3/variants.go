package v3

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// LabelTable adalah kebijakan label yang bisa dipasang ke TLD: tabel IDN
// format .txt (*IdnTable) atau LGR XML RFC 7940 (package v4). Pasang tabel
// non-.txt dengan IdnManager.RegisterTable.
type LabelTable interface {
	// GetName mengembalikan nama tabel (idnTableId untuk RDE)
	GetName() string

	// CheckLabel memeriksa U-label yang sudah lolos IDNA2008. Penolakan
	// berupa *ValidationError.
	CheckLabel(label string) error
}

// VariantTable adalah LabelTable yang mendefinisikan varian.
type VariantTable interface {
	LabelTable

	// VariantIndex mengembalikan index label (RFC 7940 §8.5): setiap code
	// point diganti anggota terkecil himpunan variannya. Dua label saling
	// varian jika dan hanya jika index label-nya sama, sehingga registry
	// cukup menyimpan satu kunci per nama dan mengecek keunikannya per zona.
	VariantIndex(label string) string

	// VariantLabels menghasilkan label varian (tanpa label aslinya) beserta
	// disposisinya, paling banyak limit buah (limit <= 0 berarti
	// DefaultVariantLimit). truncated bernilai true jika dipotong.
	VariantLabels(label string, limit int) (variants []VariantLabel, truncated bool)
}

// Disposition adalah status sebuah label menurut RFC 7940 §7.3.
type Disposition string

const (
	DispositionValid       Disposition = "valid"       // boleh didaftarkan
	DispositionInvalid     Disposition = "invalid"     // tidak boleh ada
	DispositionBlocked     Disposition = "blocked"     // tidak boleh didaftarkan siapa pun
	DispositionAllocatable Disposition = "allocatable" // boleh dialokasikan ke pendaftar label asal
	DispositionActivated   Disposition = "activated"   // dialokasikan dan diaktifkan bersama label asal
)

// DefaultVariantLimit adalah batas jumlah label varian bila limit <= 0.
// Jumlah varian tumbuh eksponensial terhadap panjang label; untuk mendeteksi
// bentrok gunakan VariantKey, bukan daftar lengkap varian.
const DefaultVariantLimit = 1000

// VariantLabel adalah satu label varian.
type VariantLabel struct {
	ULabel      string
	ALabel      string // diisi oleh LabelResult.VariantLabels
	Disposition Disposition
	Types       []string // tipe varian yang dipakai (LGR); nil untuk tabel .txt
}

// --- Varian tabel .txt format RFC 3743 (mis. idn/Chinese-IDN.txt) ---
//
// Baris: "U+XXXX(ref);<varian preferred>(ref),...;<varian karakter>(ref),..."
// Varian preferred menjadi label yang boleh dialokasikan ke pendaftar yang
// sama (preferred variant label, RFC 3743 §3.2.3); varian lain diblokir.

var variantColumnPattern = regexp.MustCompile(`U\+([0-9a-fA-F]{4,6})`)

// parseVariantColumns membaca kolom 2 dan 3 sebuah baris RFC 3743.
func parseVariantColumns(line string) (preferred, variants []rune) {
	columns := strings.Split(line, ";")
	for i, column := range columns {
		if i == 0 || i > 2 {
			continue
		}
		for _, m := range variantColumnPattern.FindAllStringSubmatch(column, -1) {
			cp, err := strconv.ParseInt(m[1], 16, 32)
			if err != nil {
				continue
			}
			if i == 1 {
				preferred = append(preferred, rune(cp))
			}
			variants = append(variants, rune(cp))
		}
	}
	return preferred, variants
}

// buildVariantSets membentuk himpunan varian (penutupan simetris dan
// transitif) dengan union-find, lalu menetapkan anggota terkecil sebagai index.
func (t *IdnTable) buildVariantSets(edges map[rune][]rune) {
	parent := make(map[rune]rune)
	var find func(r rune) rune
	find = func(r rune) rune {
		p, ok := parent[r]
		if !ok || p == r {
			parent[r] = r
			return r
		}
		root := find(p)
		parent[r] = root
		return root
	}
	for src, targets := range edges {
		for _, dst := range targets {
			a, b := find(src), find(dst)
			if a != b {
				if b < a {
					a, b = b, a
				}
				parent[b] = a // akar selalu anggota terkecil
			}
		}
	}

	t.variantIndex = make(map[rune]rune)
	t.variantSets = make(map[rune][]rune)
	for r := range parent {
		root := find(r)
		t.variantIndex[r] = root
		t.variantSets[root] = append(t.variantSets[root], r)
	}
	// Himpunan satu anggota (pemetaan identitas saja) bukan varian.
	for root, members := range t.variantSets {
		if len(members) < 2 {
			delete(t.variantSets, root)
			delete(t.variantIndex, root)
			continue
		}
		sort.Slice(members, func(i, j int) bool { return members[i] < members[j] })
	}
}

// HasVariants melaporkan apakah tabel mendefinisikan varian
func (t *IdnTable) HasVariants() bool {
	return len(t.variantSets) > 0
}

// VariantIndex mengimplementasikan VariantTable
func (t *IdnTable) VariantIndex(label string) string {
	if !t.HasVariants() {
		return label
	}
	runes := []rune(label)
	for i, r := range runes {
		if root, ok := t.variantIndex[r]; ok {
			runes[i] = root
		}
	}
	return string(runes)
}

// variantsOf mengembalikan anggota himpunan varian r (termasuk r sendiri)
func (t *IdnTable) variantsOf(r rune) []rune {
	if root, ok := t.variantIndex[r]; ok {
		return t.variantSets[root]
	}
	return []rune{r}
}

// VariantLabels mengimplementasikan VariantTable
func (t *IdnTable) VariantLabels(label string, limit int) ([]VariantLabel, bool) {
	if !t.HasVariants() {
		return nil, false
	}
	if limit <= 0 {
		limit = DefaultVariantLimit
	}

	runes := []rune(label)
	options := make([][]rune, len(runes))
	for i, r := range runes {
		options[i] = t.variantsOf(r)
	}

	var out []VariantLabel
	current := make([]rune, len(runes))
	truncated := false
	var walk func(pos int, preferred, changed bool)
	walk = func(pos int, preferred, changed bool) {
		if truncated {
			return
		}
		if pos == len(runes) {
			if !changed {
				return
			}
			if len(out) == limit {
				truncated = true
				return
			}
			disp := DispositionBlocked
			if preferred {
				disp = DispositionAllocatable
			}
			out = append(out, VariantLabel{ULabel: string(current), Disposition: disp})
			return
		}
		for _, candidate := range options[pos] {
			current[pos] = candidate
			walk(pos+1, preferred && t.isPreferred(runes[pos], candidate), changed || candidate != runes[pos])
		}
	}
	walk(0, true, false)
	return out, truncated
}

// isPreferred: candidate adalah varian preferred dari r. Code point tanpa
// varian preferred dianggap preferred terhadap dirinya sendiri.
func (t *IdnTable) isPreferred(r, candidate rune) bool {
	preferred := t.preferred[r]
	if len(preferred) == 0 {
		return candidate == r
	}
	for _, p := range preferred {
		if p == candidate {
			return true
		}
	}
	return false
}
