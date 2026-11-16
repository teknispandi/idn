package v4

import (
	"path/filepath"
	"testing"
)

// Semua LGR di lgrs/ harus bisa dimuat: elemen, properti dan referensi
// yang tidak didukung ditolak saat dimuat, jadi ini menguji cakupan parser.
func TestLoadAllLGRs(t *testing.T) {
	files, err := filepath.Glob("../lgrs/*/*.xml")
	if err != nil || len(files) == 0 {
		t.Fatalf("no LGR files: %v", err)
	}
	for _, f := range files {
		l, err := LoadFile(f)
		if err != nil {
			t.Errorf("%s: %v", f, err)
			continue
		}
		if l.GetName() == "" || len(l.elements) == 0 {
			t.Errorf("%s: empty LGR", f)
		}
	}
	t.Logf("loaded %d LGR files", len(files))
}
