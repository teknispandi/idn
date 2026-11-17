package v4

import (
	"math/rand"
	"sort"
	"testing"

	v3 "github.com/al0fayz/idn/v3"
)

// Validator yang ditulis tangan di v3 dan mesin LGR ini yang membaca file
// XML-nya adalah dua implementasi independen dari LGR yang sama; keduanya
// harus sepakat untuk setiap label. Ini menguji kelas from-tag, choice,
// anchor, look-behind/look-ahead, sekuens dan aksi sekaligus.

func TestBalineseLGRMatchesHandwrittenValidator(t *testing.T) {
	var weighted []rune
	for r := rune(0x1B13); r <= 0x1B33; r++ { // lebih banyak konsonan → label sah lebih sering
		weighted = append(weighted, r, r)
	}
	weighted = append(weighted, 0x1B44, 0x1B44, 0x1B44) // adeg-adeg
	crossCheck(t, "und-Bali", &v3.BalineseLanguageValidator{}, 66, weighted, 50000)
}

func TestJavaneseLGRMatchesHandwrittenValidator(t *testing.T) {
	var weighted []rune
	for r := rune(0xA98F); r <= 0xA9B2; r++ { // konsonan (code point di luar repertoire dibuang crossCheck)
		weighted = append(weighted, r, r)
	}
	weighted = append(weighted,
		0xA9C0, 0xA9C0, 0xA9C0, // pangkon
		0xA9B3, 0xA9B3, // cecak telu (aksara rekan)
		0xA98F, 0xA992, 0xA997, 0xA9A2, 0xA9A5, 0xA9AE, // basis aksara rekan
	)
	crossCheck(t, "und-Java", &v3.JavaneseLanguageValidator{}, 53, weighted, 30000)
}

// crossCheck membandingkan validator dengan LGR untuk semua label sepanjang
// 1–3 code point dari repertoire, ditambah 100.000 label acak sepanjang
// 4–12. weighted menambah bobot code point tertentu agar label sah cukup
// sering muncul; minValid memastikan sampelnya tidak didominasi label tidak sah.
func crossCheck(t *testing.T, name string, handwritten v3.LanguageValidator, wantCodePoints int, weighted []rune, minValid int) {
	t.Helper()
	lgr := mustLoad(t, name)

	inRepertoire := make(map[rune]bool)
	for key := range lgr.elements {
		for _, r := range key {
			inRepertoire[r] = true
		}
	}
	repertoire := make([]rune, 0, len(inRepertoire))
	for r := range inRepertoire {
		repertoire = append(repertoire, r)
	}
	sort.Slice(repertoire, func(i, j int) bool { return repertoire[i] < repertoire[j] })
	if len(repertoire) != wantCodePoints {
		t.Fatalf("%s repertoire has %d code points, want %d", name, len(repertoire), wantCodePoints)
	}

	mismatches, valid, checked := 0, 0, 0
	compare := func(label []rune) {
		checked++
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

	var walk func(prefix []rune, depth int)
	walk = func(prefix []rune, depth int) {
		if len(prefix) > 0 {
			compare(prefix)
		}
		if depth == 0 {
			return
		}
		for _, r := range repertoire {
			walk(append(prefix, r), depth-1)
		}
	}
	walk(nil, 3)

	pool := append([]rune{}, repertoire...)
	for _, r := range weighted {
		if inRepertoire[r] {
			pool = append(pool, r)
		}
	}
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 100000; i++ {
		label := make([]rune, 4+rng.Intn(9))
		for j := range label {
			label[j] = pool[rng.Intn(len(pool))]
		}
		compare(label)
	}

	if mismatches > 0 {
		t.Errorf("%s: %d of %d labels disagree", name, mismatches, checked)
	}
	if valid < minValid {
		t.Errorf("%s: only %d valid labels among %d; sample too weak", name, valid, checked)
	}
	t.Logf("%s: %d labels compared, %d valid", name, checked, valid)
}
