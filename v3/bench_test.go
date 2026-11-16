package v3

import (
	"testing"

	idn "github.com/al0fayz/idn"
)

// Jalankan: go test ./v3 -run '^$' -bench . -benchmem
// Hasil dicatat di v3/README.md (bagian Performance).

func BenchmarkCheckDomain(b *testing.B) {
	m := NewIdnManagerFS(idn.Tables())
	for zone, languages := range map[string][]string{
		"id": {"latin", "und-bali"},
		"zh": {"chinese"},
	} {
		if err := m.RegisterTld(zone, languages); err != nil {
			b.Fatal(err)
		}
	}
	for _, bc := range []struct{ name, domain string }{
		{"ascii", "contoh-domain.id"},
		{"latin-idn", "café-kopi.id"},
		{"a-label", "xn--caf-kopi-e2a.id"},
		{"bali", "ᬩᬮᬶ.id"},
		{"chinese-variants", "中国岩石.zh"},
		{"rejected", "xn--ls8h.id"},
	} {
		b.Run(bc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				m.CheckDomain(bc.domain)
			}
		})
	}
}

func BenchmarkLoadTableCold(b *testing.B) {
	for _, language := range []string{"latin", "chinese"} {
		b.Run(language, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := NewIdnLoaderFS(idn.Tables()).LoadTable(language); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkVariantLabels(b *testing.B) {
	m := NewIdnManagerFS(idn.Tables())
	if err := m.RegisterTld("zh", []string{"chinese"}); err != nil {
		b.Fatal(err)
	}
	res, err := m.CheckDomain("巖石.zh")
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		res.VariantLabels(0)
	}
}
