package v3

import (
	"errors"
	"fmt"
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
	// DefaultVariantLimit). Hanya varian yang lolos IDNA2008 yang dihitung
	// terhadap limit. truncated bernilai true jika dipotong.
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

// maxVariantCombinations membatasi kombinasi yang dicoba IdnTable.VariantLabels,
// agar label yang sebagian besar variannya tidak valid tetap cepat selesai.
const maxVariantCombinations = 200_000

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
	tried := 0
	var walk func(pos int, preferred, changed bool)
	walk = func(pos int, preferred, changed bool) {
		if truncated {
			return
		}
		if pos == len(runes) {
			if !changed {
				return
			}
			if tried++; tried > maxVariantCombinations {
				truncated = true
				return
			}
			// Varian yang melanggar IDNA2008 (mis. tanda hubung di awal) tidak
			// dihitung terhadap limit.
			p, err := prepareLabel(string(current))
			if err != nil {
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
			out = append(out, VariantLabel{ULabel: p.ulabel, ALabel: p.alabel, Disposition: disp})
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

// --- Koherensi varian antar tabel dalam satu zona ---
//
// ICANN IDN Implementation Guidelines, Guideline 13 (diusulkan di v4.0,
// ditangguhkan/deferred di v4.1): semua tabel IDN dengan kebijakan varian
// dalam satu TLD harus memiliki code point varian yang seragam. Terlepas
// dari status pedoman itu, di modul ini syaratnya teknis: VariantKey dihitung oleh
// tabel yang menerima label, jadi bila dua tabel memberi index berbeda untuk
// elemen yang sama, bentrok varian antar nama bisa lolos.

// VariantSetTable adalah tabel yang mengekspos himpunan variannya untuk
// pemeriksaan koherensi. *IdnTable dan *v4.LGR mengimplementasikannya;
// tabel lain dilewati oleh VariantConflicts.
type VariantSetTable interface {
	LabelTable

	// VariantElements memetakan setiap elemen yang dikenal tabel (anggota
	// repertoire dan target varian; code point atau sekuens) ke index
	// himpunan variannya. Elemen tanpa varian ber-index dirinya sendiri.
	VariantElements() map[string]string
}

// VariantElements mengimplementasikan VariantSetTable
func (t *IdnTable) VariantElements() map[string]string {
	out := make(map[string]string, len(t.validCodepoints)+len(t.variantIndex))
	for cp := range t.validCodepoints {
		out[string(rune(cp))] = string(rune(cp))
	}
	for r, root := range t.variantIndex {
		out[string(r)] = string(root)
	}
	return out
}

// VariantConflict menyatakan dua tabel dalam satu zona yang tidak sepakat
// tentang himpunan varian sejumlah elemen bersama.
type VariantConflict struct {
	Zone     string
	TableA   string
	TableB   string
	Count    int      // jumlah elemen bersama yang index-nya berbeda
	Examples []string // hingga 5 contoh, mis. "U+5CA9 岩: chinese→U+55A6, japanese→U+5CA9"
}

func (c VariantConflict) String() string {
	return fmt.Sprintf("zone %s: tables %s and %s disagree on the variant sets of %d shared code points (e.g. %s)",
		c.Zone, c.TableA, c.TableB, c.Count, strings.Join(c.Examples, "; "))
}

// VariantConflicts membandingkan setiap pasangan tabel TLD yang setidaknya
// salah satunya mendefinisikan varian. Hasil kosong berarti koheren.
func (manager *IdnManager) VariantConflicts(tldName string) ([]VariantConflict, error) {
	config, tables, err := manager.lookup(tldName)
	if err != nil {
		return nil, err
	}

	type info struct {
		name     string
		elements map[string]string
		variants bool
	}
	var infos []info
	for _, table := range tables {
		vs, ok := table.(VariantSetTable)
		if !ok {
			continue
		}
		elements := vs.VariantElements()
		hasVariants := false
		for element, index := range elements {
			if element != index {
				hasVariants = true
				break
			}
		}
		infos = append(infos, info{table.GetName(), elements, hasVariants})
	}

	var conflicts []VariantConflict
	for i := 0; i < len(infos); i++ {
		for j := i + 1; j < len(infos); j++ {
			a, b := infos[i], infos[j]
			if !a.variants && !b.variants {
				continue
			}
			var differing []string
			for element, indexA := range a.elements {
				if indexB, ok := b.elements[element]; ok && indexA != indexB {
					differing = append(differing, element)
				}
			}
			if len(differing) == 0 {
				continue
			}
			sort.Slice(differing, func(x, y int) bool { return differing[x] < differing[y] })
			c := VariantConflict{Zone: config.TldName, TableA: a.name, TableB: b.name, Count: len(differing)}
			for _, element := range differing[:min(5, len(differing))] {
				c.Examples = append(c.Examples, fmt.Sprintf("%s %s: %s→%s, %s→%s",
					formatElement(element), element, a.name, formatElement(a.elements[element]), b.name, formatElement(b.elements[element])))
			}
			conflicts = append(conflicts, c)
		}
	}
	return conflicts, nil
}

// CheckVariantCoherence memeriksa VariantConflicts untuk setiap TLD aktif.
// Panggil saat registry mulai, bersama CheckRdeTableRefs.
func (manager *IdnManager) CheckVariantCoherence() error {
	var errs []error
	for _, tld := range manager.ListRegisteredTlds() {
		config, err := manager.GetTldConfig(tld)
		if err != nil || !config.Enabled {
			continue
		}
		conflicts, err := manager.VariantConflicts(tld)
		if err != nil {
			errs = append(errs, err)
		}
		for _, c := range conflicts {
			errs = append(errs, errors.New(c.String()))
		}
	}
	return errors.Join(errs...)
}

func formatElement(element string) string {
	parts := make([]string, 0, len(element))
	for _, r := range element {
		parts = append(parts, fmt.Sprintf("U+%04X", r))
	}
	return strings.Join(parts, " ")
}
