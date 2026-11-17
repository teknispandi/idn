package v3

import (
	"strings"
	"testing"

	idn "github.com/al0fayz/idn"
)

// Data dari idn/Chinese-IDN.txt:
//
//	U+5CA9 岩: preferred 岩, kelas {喦 壧 岩 嵒 巌 巖 碞 礹}, index U+55A6 喦
//	U+5DD6 巖: preferred 岩
//	U+77F3 石: tanpa varian
func newVariantManager(t *testing.T) *IdnManager {
	t.Helper()
	m := NewIdnManagerFS(idn.Tables())
	if err := m.RegisterTld("zh", []string{"chinese"}); err != nil {
		t.Fatal(err)
	}
	if err := m.RegisterTld("id", []string{"latin", "und-bali"}); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestVariantKeyDetectsVariantCollision(t *testing.T) {
	m := newVariantManager(t)

	simplified, err := m.CheckDomain("岩石.zh")
	if err != nil {
		t.Fatal(err)
	}
	traditional, err := m.CheckDomain("巖石.zh")
	if err != nil {
		t.Fatal(err)
	}
	other, err := m.CheckDomain("石岩.zh")
	if err != nil {
		t.Fatal(err)
	}

	want := mustPunycode(t, "喦石")
	if simplified.VariantKey != want || traditional.VariantKey != want {
		t.Errorf("VariantKey = %q / %q, want both %q", simplified.VariantKey, traditional.VariantKey, want)
	}
	if other.VariantKey == want {
		t.Errorf("石岩 must not collide with 岩石")
	}
	if simplified.ALabel == traditional.ALabel {
		t.Errorf("variants must keep distinct A-labels")
	}
}

func TestVariantLabelsDispositions(t *testing.T) {
	m := newVariantManager(t)

	res, err := m.CheckDomain("巖石.zh")
	if err != nil {
		t.Fatal(err)
	}
	variants, truncated := res.VariantLabels(0)
	if truncated || len(variants) != 7 {
		t.Fatalf("got %d variants (truncated=%v), want 7", len(variants), truncated)
	}
	var allocatable []string
	for _, v := range variants {
		if v.ULabel == "巖石" || v.ALabel == "" || !strings.HasPrefix(v.ALabel, "xn--") {
			t.Errorf("bad variant %+v", v)
		}
		if v.Disposition == DispositionAllocatable {
			allocatable = append(allocatable, v.ULabel)
		} else if v.Disposition != DispositionBlocked {
			t.Errorf("unexpected disposition %+v", v)
		}
	}
	if len(allocatable) != 1 || allocatable[0] != "岩石" {
		t.Errorf("allocatable = %v, want [岩石] (preferred variant of 巖)", allocatable)
	}

	// 岩 adalah bentuk preferred-nya sendiri: semua variannya diblokir.
	res, _ = m.CheckDomain("岩石.zh")
	variants, _ = res.VariantLabels(0)
	for _, v := range variants {
		if v.Disposition != DispositionBlocked {
			t.Errorf("%s: disposition %s, want blocked", v.ULabel, v.Disposition)
		}
	}

	variants, truncated = res.VariantLabels(3)
	if !truncated || len(variants) != 3 {
		t.Errorf("limit 3: got %d (truncated=%v)", len(variants), truncated)
	}
}

func TestNoVariantsForPlainTables(t *testing.T) {
	m := newVariantManager(t)
	res, err := m.CheckDomain("café.id")
	if err != nil {
		t.Fatal(err)
	}
	if res.VariantKey != res.ALabel {
		t.Errorf("VariantKey = %q, want ALabel %q", res.VariantKey, res.ALabel)
	}
	if variants, _ := res.VariantLabels(0); variants != nil {
		t.Errorf("VariantLabels = %v, want nil", variants)
	}

	ascii, err := m.CheckDomain("example.id")
	if err != nil || ascii.VariantKey != "example" {
		t.Errorf("ASCII VariantKey = %q, %v", ascii.VariantKey, err)
	}
}

func TestRdeTableRefs(t *testing.T) {
	m := newVariantManager(t)

	// latin punya header # URL:/# Policy:, und-bali tidak.
	if _, err := m.RdeTableRefs("id"); err == nil || !strings.Contains(err.Error(), "und-bali") {
		t.Errorf("RdeTableRefs(id) error = %v, want missing und-bali", err)
	}
	if err := m.CheckRdeTableRefs(); err == nil {
		t.Error("CheckRdeTableRefs() = nil, want error for id and zh")
	}

	err := m.RegisterTldConfig(TldIdnConfig{
		TldName: "id", Languages: []string{"latin", "und-bali"}, Enabled: true, AllowASCII: true,
		TableRefs: map[string]TableRef{
			"Latin":    {URL: "https://www.iana.org/domains/idn-tables/tables/id_latn_1.0.txt", Policy: "https://pandi.id/idn"},
			"und-Bali": {URL: "https://www.iana.org/domains/idn-tables/tables/id_bali_1.0.txt", Policy: "https://pandi.id/idn"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	refs, err := m.RdeTableRefs("id")
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 || refs[0].ID != "latin" || refs[1].ID != "und-bali" || refs[0].URL != "https://www.iana.org/domains/idn-tables/tables/id_latn_1.0.txt" {
		t.Errorf("refs = %+v", refs)
	}
}

func mustPunycode(t *testing.T, s string) string {
	t.Helper()
	p, err := prepareLabel(s)
	if err != nil {
		// index label belum tentu label valid; encode tanpa validasi
		a, err := DefaultIdnUtils.ToASCII(s)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	return p.alabel
}

// Guideline 13 ICANN (ditangguhkan di v4.1): tabel dengan kebijakan varian dalam satu zona harus
// seragam. Chinese dan Traditional-Chinese berbagi himpunan varian yang
// sama; Japanese memuat aksara Han yang sama tanpa varian, jadi bentrok
// antara 岩 dan 巖 bisa lolos bila kedua tabel dipakai di satu zona.
func TestVariantCoherence(t *testing.T) {
	m := NewIdnManagerFS(idn.Tables())
	for zone, languages := range map[string][]string{
		"hans-hant": {"chinese", "traditional-chinese"},
		"zh-ja":     {"chinese", "japanese"},
		"id":        {"latin", "und-bali"},
	} {
		if err := m.RegisterTld(zone, languages); err != nil {
			t.Fatal(err)
		}
	}

	for _, zone := range []string{"hans-hant", "id"} {
		if conflicts, err := m.VariantConflicts(zone); err != nil || len(conflicts) != 0 {
			t.Errorf("VariantConflicts(%s) = %v, %v; want none", zone, conflicts, err)
		}
	}

	conflicts, err := m.VariantConflicts("zh-ja")
	if err != nil || len(conflicts) != 1 {
		t.Fatalf("VariantConflicts(zh-ja) = %v, %v; want 1 conflict", conflicts, err)
	}
	c := conflicts[0]
	if c.TableA != "chinese" || c.TableB != "japanese" || c.Count == 0 || len(c.Examples) == 0 {
		t.Errorf("conflict = %+v", c)
	}

	err = m.CheckVariantCoherence()
	if err == nil || !strings.Contains(err.Error(), "zone zh-ja") || strings.Contains(err.Error(), "hans-hant") {
		t.Errorf("CheckVariantCoherence() = %v, want only zh-ja", err)
	}
	t.Log(c)
}

// Varian yang melanggar IDNA2008 tidak boleh memakan jatah limit: dulu
// VariantLabels(1) bisa mengembalikan daftar kosong walau ada varian sah.
func TestVariantLabelsLimitCountsOnlyValid(t *testing.T) {
	table, err := NewIdnTable("hyphen-variants", []string{
		"U+002D(0);U+002D(0);U+0078(0)", // '-' ↔ 'x'
		"U+0078(0);U+0078(0);U+002D(0)",
		"U+0061(0);U+0061(0);U+0062(0)", // 'a' ↔ 'b'
		"U+0062(0);U+0062(0);U+0061(0)",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Varian "xa": "-a", "-b" (diawali tanda hubung: tidak valid), "xb".
	variants, truncated := table.VariantLabels("xa", 1)
	if len(variants) != 1 || variants[0].ULabel != "xb" || truncated {
		t.Errorf("VariantLabels(xa, 1) = %+v, truncated=%v; want [xb], false", variants, truncated)
	}
}
