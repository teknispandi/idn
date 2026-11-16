package v4

import "testing"

// Jalankan: go test ./v4 -run '^$' -bench . -benchmem
// Hasil dicatat di v4/README.md.

func BenchmarkLoad(b *testing.B) {
	for _, name := range []string{"und-Bali", "zh"} {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := LoadFile(refDir + name + ".xml"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkCheckLabel(b *testing.B) {
	for _, bc := range []struct{ lgr, label string }{
		{"und-Bali", "ᬩᬮᬶᬓ᭄ᬱ"},
		{"zh", "中国岩石"},
		{"und-Arab", "مثال"},
	} {
		l, err := LoadFile(refDir + bc.lgr + ".xml")
		if err != nil {
			b.Fatal(err)
		}
		b.Run(bc.lgr, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				l.CheckLabel(bc.label)
			}
		})
	}
}

func BenchmarkVariantLabels(b *testing.B) {
	zh, err := LoadFile(refDir + "zh.xml")
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		zh.VariantLabels("中国", 0)
	}
}
