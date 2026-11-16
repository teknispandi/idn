package v4

import (
	"math/rand"
	"sort"
	"testing"

	v3 "github.com/al0fayz/idn/v3"
)

// v3.BalineseLanguageValidator (ditulis tangan) dan mesin LGR ini yang
// membaca und-Bali.xml adalah dua implementasi independen dari LGR yang
// sama; keduanya harus sepakat untuk setiap label. Ini menguji kelas
// from-tag, choice, anchor, look-behind/look-ahead dan aksi sekaligus.
func TestBalineseLGRMatchesHandwrittenValidator(t *testing.T) {
	lgr, err := LoadFile("../lgrs/second-level-reference/und-Bali.xml")
	if err != nil {
		t.Fatal(err)
	}
	var repertoire []rune
	for key := range lgr.elements {
		repertoire = append(repertoire, []rune(key)...)
	}
	sort.Slice(repertoire, func(i, j int) bool { return repertoire[i] < repertoire[j] })
	if len(repertoire) != 66 {
		t.Fatalf("repertoire has %d code points, want 66", len(repertoire))
	}

	handwritten := &v3.BalineseLanguageValidator{}
	mismatches, valid := 0, 0
	compare := func(label []rune) {
		want := handwritten.IsValidLabelForLanguage(string(label))
		if want {
			valid++
		}
		disp, _ := lgr.Disposition(string(label))
		if got := disp == v3.DispositionValid; got != want {
			mismatches++
			if mismatches <= 10 {
				t.Errorf("%q %U: handwritten=%v lgr=%s", string(label), label, want, disp)
			}
		}
	}

	checked := 0
	var walk func(prefix []rune, depth int)
	walk = func(prefix []rune, depth int) {
		if len(prefix) > 0 {
			compare(prefix)
			checked++
		}
		if depth == 0 {
			return
		}
		for _, r := range repertoire {
			walk(append(prefix, r), depth-1)
		}
	}
	walk(nil, 3)

	rng := rand.New(rand.NewSource(1))
	weighted := append([]rune{}, repertoire...)
	for r := rune(0x1B13); r <= 0x1B33; r++ { // lebih banyak konsonan → label sah lebih sering
		weighted = append(weighted, r, r)
	}
	weighted = append(weighted, 0x1B44, 0x1B44, 0x1B44)
	for i := 0; i < 100000; i++ {
		label := make([]rune, 4+rng.Intn(9))
		for j := range label {
			label[j] = weighted[rng.Intn(len(weighted))]
		}
		compare(label)
		checked++
	}
	if mismatches > 0 {
		t.Errorf("%d of %d labels disagree", mismatches, checked)
	}
	// Perbandingan hanya bermakna bila banyak label yang sah ikut diuji.
	if valid < 50000 {
		t.Errorf("only %d valid labels among %d; sample too weak", valid, checked)
	}
	t.Logf("%d labels compared, %d valid", checked, valid)
}
