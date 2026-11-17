package v4

import (
	"path/filepath"
	"strings"
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

func TestUnicodeVersionCheck(t *testing.T) {
	lgr := func(version, cp string) string {
		return `<lgr xmlns="urn:ietf:params:xml:ns:lgr-1.0"><meta><unicode-version>` + version +
			`</unicode-version></meta><data><char cp="` + cp + `"/></data></lgr>`
	}
	for _, c := range []struct {
		version, cp string
		wantErr     bool
	}{
		{"11.0.0", "0061", false},
		{SupportedUnicodeVersion(), "0061", false},
		{"99.0.0", "0061", false}, // versi lebih baru, tetapi code point-nya sudah dikenal
		{"99.0.0", "0378", true},  // U+0378 belum ditetapkan di Unicode runtime
		{"11.0.0", "0378", false}, // versi lama: tidak diperiksa di sini
		{"not-a-version", "0061", true},
	} {
		_, err := Parse(strings.NewReader(lgr(c.version, c.cp)))
		if (err != nil) != c.wantErr {
			t.Errorf("unicode-version %q cp %s: err = %v, want error=%v", c.version, c.cp, err, c.wantErr)
		}
	}
	if compareVersions("14.0.0", "15.0.0") >= 0 || compareVersions("15.1", "15.0.0") <= 0 {
		t.Error("compareVersions ordering is wrong")
	}
}
