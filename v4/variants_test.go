package v4

import (
	"errors"
	"sync"
	"testing"
	"time"

	idn "github.com/al0fayz/idn"
	v3 "github.com/al0fayz/idn/v3"
)

const refDir = "../lgrs/second-level-reference/"

func mustLoad(t *testing.T, name string) *LGR {
	t.Helper()
	l, err := LoadFile(refDir + name + ".xml")
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func variantMap(vs []v3.VariantLabel) map[string]v3.Disposition {
	m := make(map[string]v3.Disposition, len(vs))
	for _, v := range vs {
		m[v.ULabel] = v.Disposition
	}
	return m
}

// zh.xml: 国 (56FD) refleksif r-simp, varian 國 (570B) bertipe trad, 囯/圀 blocked.
func TestChineseVariantDispositions(t *testing.T) {
	zh := mustLoad(t, "zh")

	for _, label := range []string{"国", "國", "中国"} {
		if disp, err := zh.Disposition(label); disp != v3.DispositionValid {
			t.Errorf("Disposition(%s) = %s, %v; want valid", label, disp, err)
		}
	}

	variants, truncated := zh.VariantLabels("国", 0)
	got := variantMap(variants)
	if truncated || got["國"] != v3.DispositionAllocatable {
		t.Errorf("国 → 國 = %q (truncated=%v), want allocatable; all: %v", got["國"], truncated, got)
	}
	for label, disp := range got {
		if label != "國" && disp != v3.DispositionBlocked {
			t.Errorf("国 → %s = %s, want blocked", label, disp)
		}
	}

	// only-variants: 中 tetap (refleksif r-both) + 国→國 (trad) masih allocatable.
	if disp := variantMap(must(zh.VariantLabels("中国", 0)))["中國"]; disp != v3.DispositionAllocatable {
		t.Errorf("中国 → 中國 = %q, want allocatable", disp)
	}

	if a, b := zh.VariantIndex("国"), zh.VariantIndex("國"); a != b || a != "囯" {
		t.Errorf("VariantIndex 国/國 = %q/%q, want both 囯", a, b)
	}
}

// de.xml: ß ↔ "ss" (sekuens) dengan tipe fallback / r-original.
func TestGermanSharpSVariants(t *testing.T) {
	de := mustLoad(t, "de")

	if disp := variantMap(must(de.VariantLabels("straße", 0)))["strasse"]; disp != v3.DispositionAllocatable {
		t.Errorf("straße → strasse = %q, want allocatable", disp)
	}
	if disp := variantMap(must(de.VariantLabels("strasse", 0)))["straße"]; disp != v3.DispositionBlocked {
		t.Errorf("strasse → straße = %q, want blocked", disp)
	}
	if a, b := de.VariantIndex("straße"), de.VariantIndex("strasse"); a != b {
		t.Errorf("VariantIndex straße/strasse = %q/%q, want equal", a, b)
	}
}

// und-Arab.xml: U+0649 not-when="initial-or-medial-position", yaitu
// diikuti huruf yang tersambung ke kanan (jt:R atau jt:D).
func TestArabicAlefMaksuraJoiningContext(t *testing.T) {
	arab := mustLoad(t, "und-Arab")
	for label, want := range map[string]bool{
		"بى": true,  // akhir label
		"ىء": true,  // HAMZA bertipe U: tidak tersambung
		"ىب": false, // BEH bertipe D
		"ىا": false, // ALEF bertipe R
	} {
		err := arab.CheckLabel(label)
		if (err == nil) != want {
			t.Errorf("CheckLabel(%s) = %v, want valid=%v", label, err, want)
		}
		var ve *v3.ValidationError
		if !want && (!errors.As(err, &ve) || ve.Rule != "initial-or-medial-position" || ve.Pos != 0) {
			t.Errorf("CheckLabel(%s) error = %v, want rule initial-or-medial-position at 0", label, err)
		}
	}
}

func TestBalineseVariantsAndErrors(t *testing.T) {
	bali := mustLoad(t, "und-Bali")

	if got := variantMap(must(bali.VariantLabels("ᬓ", 0))); len(got) != 1 || got["ᬔ"] != v3.DispositionBlocked {
		t.Errorf("ᬓ variants = %v, want {ᬔ: blocked}", got)
	}

	err := bali.CheckLabel("ᬓ᭄ᬓ᭄ᬓ")
	var ve *v3.ValidationError
	if !errors.As(err, &ve) || !errors.Is(err, v3.ErrLanguageRule) || ve.Rule != "disallow-for-1B44" || ve.Pos != 3 || ve.Table != "und-bali" {
		t.Errorf("C᭄C᭄C error = %v, want disallow-for-1B44 at 3", err)
	}
	if err := bali.CheckLabel("ᬓa"); !errors.Is(err, v3.ErrCodepointNotInTable) {
		t.Errorf("mixed label error = %v, want ErrCodepointNotInTable", err)
	}
}

// LGR dipasang ke v3.IdnManager: satu jalur dengan tabel .txt.
func TestLGRInIdnManager(t *testing.T) {
	m := v3.NewIdnManagerFS(idn.Tables())
	for _, name := range []string{"und-Bali", "de"} {
		if err := m.RegisterTable(mustLoad(t, name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.RegisterTld("id", []string{"latin", "und-bali"}); err != nil {
		t.Fatal(err)
	}
	if err := m.RegisterTld("de", []string{"de"}); err != nil {
		t.Fatal(err)
	}

	res, err := m.CheckDomain("ᬓ.id")
	if err != nil {
		t.Fatal(err)
	}
	if _, isLGR := res.Table.(*LGR); !isLGR || res.TableName() != "und-bali" {
		t.Errorf("table = %T %q, want *v4.LGR und-bali", res.Table, res.TableName())
	}
	variant, err := m.CheckDomain("ᬔ.id")
	if err != nil {
		t.Fatal(err)
	}
	if res.VariantKey != variant.VariantKey {
		t.Errorf("VariantKey ᬓ/ᬔ = %q/%q, want equal (blocked variants)", res.VariantKey, variant.VariantKey)
	}
	if vs, _ := res.VariantLabels(0); len(vs) != 1 || vs[0].ALabel != variant.ALabel {
		t.Errorf("VariantLabels = %+v, want [%s]", vs, variant.ALabel)
	}

	// Label ASCII lolos tanpa tabel, tapi kunci variannya tetap dari LGR.
	ascii, err := m.CheckDomain("strasse.de")
	if err != nil {
		t.Fatal(err)
	}
	sharp, err := m.CheckDomain("straße.de")
	if err != nil {
		t.Fatal(err)
	}
	if ascii.VariantKey != sharp.VariantKey {
		t.Errorf("VariantKey strasse/straße = %q/%q, want equal", ascii.VariantKey, sharp.VariantKey)
	}
}

func must(vs []v3.VariantLabel, _ bool) []v3.VariantLabel { return vs }

// Jumlah varian tumbuh eksponensial; VariantLabels harus berhenti di limit
// dan melaporkan truncated, bukan menghabiskan memori.
func TestVariantLabelsAreBounded(t *testing.T) {
	zh := mustLoad(t, "zh")
	label := "国国国国国国国国国国国国国国国国国国国国" // 20 posisi, masing-masing 4 pilihan
	start := time.Now()
	variants, truncated := zh.VariantLabels(label, 50)
	if !truncated || len(variants) != 50 {
		t.Errorf("got %d variants (truncated=%v), want 50 truncated", len(variants), truncated)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("VariantLabels took %v", elapsed)
	}
}

func TestLGRConcurrentUse(t *testing.T) {
	bali := mustLoad(t, "und-Bali")
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				if err := bali.CheckLabel("ᬩᬮᬶ"); err != nil {
					t.Error(err)
					return
				}
				bali.VariantLabels("ᬓᬩ", 0)
				bali.VariantIndex("ᬓᬩ")
			}
		}()
	}
	wg.Wait()
}

func TestLGRVariantCoherence(t *testing.T) {
	m := v3.NewIdnManagerFS(idn.Tables())
	if err := m.RegisterTable(mustLoad(t, "und-Bali")); err != nil {
		t.Fatal(err)
	}
	if err := m.RegisterTld("id", []string{"latin", "und-bali"}); err != nil {
		t.Fatal(err)
	}
	if conflicts, err := m.VariantConflicts("id"); err != nil || len(conflicts) != 0 {
		t.Errorf("latin + und-bali LGR: %v, %v; want coherent", conflicts, err)
	}
}
