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
	lgr := func(version string) string {
		return `<lgr xmlns="urn:ietf:params:xml:ns:lgr-1.0"><meta><unicode-version>` + version +
			`</unicode-version></meta><data><char cp="0061"/></data></lgr>`
	}
	for version, wantErr := range map[string]bool{
		"11.0.0":                  false,
		SupportedUnicodeVersion(): false,
		"99.0.0":                  true,
		"not-a-version":           true,
	} {
		_, err := Parse(strings.NewReader(lgr(version)))
		if (err != nil) != wantErr {
			t.Errorf("unicode-version %q: err = %v, want error=%v", version, err, wantErr)
		}
	}
	if compareVersions("14.0.0", "15.0.0") >= 0 || compareVersions("15.1", "15.0.0") <= 0 {
		t.Error("compareVersions ordering is wrong")
	}
}
