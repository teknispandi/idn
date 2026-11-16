package main

import (
	"errors"
	"fmt"
	"log"

	idn "github.com/al0fayz/idn"
	v3 "github.com/al0fayz/idn/v3"
)

// ExampleRegistry memperlihatkan alur di sisi registry: konfigurasi zona,
// pemeriksaan saat start, validasi domain:create, deteksi bentrok varian
// lewat VariantKey, dan referensi tabel untuk deposit RDE.
func ExampleRegistry() {
	fmt.Println("=== Registry ===")
	manager := v3.NewIdnManagerFS(idn.Tables())

	// URL di bawah hanya contoh (.invalid, RFC 2606). Isi dengan URL tabel
	// milik registry di IANA Repository of IDN Practices.
	refs := func(tables ...string) map[string]v3.TableRef {
		m := make(map[string]v3.TableRef)
		for _, t := range tables {
			m[t] = v3.TableRef{URL: "https://contoh.invalid/idn-tables/" + t + ".txt", Policy: "https://contoh.invalid/kebijakan-idn"}
		}
		return m
	}
	zones := []v3.TldIdnConfig{
		{TldName: "id", Languages: []string{"latin", "und-bali"}, Enabled: true, AllowASCII: true, TableRefs: refs("latin", "und-bali")},
		{TldName: "co.id", Languages: []string{"latin"}, Enabled: true, AllowASCII: true, TableRefs: refs("latin")},
		{TldName: "zh", Languages: []string{"chinese"}, Enabled: true, AllowASCII: true, TableRefs: refs("chinese")},
	}
	for _, zone := range zones {
		if err := manager.RegisterTldConfig(zone); err != nil {
			log.Fatal(err)
		}
	}

	// Pemeriksaan saat start: gagal di sini lebih baik daripada saat deposit.
	if err := manager.CheckRdeTableRefs(); err != nil {
		log.Fatal(err)
	}
	if err := manager.CheckVariantCoherence(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ referensi RDE dan koherensi varian semua zona OK")

	// Pengganti tabel domain: UNIQUE(zone, variant_key).
	registered := make(map[string]string) // zone|variant_key → nama terdaftar

	create := func(name string) {
		res, err := manager.CheckDomain(name)
		if err != nil {
			fmt.Printf("❌ %-22s %s — %v\n", name, eppResult(err), err)
			return
		}
		key := res.Zone + "|" + res.VariantKey
		if owner, taken := registered[key]; taken {
			fmt.Printf("❌ %-22s 2302 — bentrok dengan %s (varian atau nama yang sama)\n", name, owner)
			return
		}
		registered[key] = res.ASCIIName
		fmt.Printf("✅ %-22s disimpan %s (tabel %q, variant_key %s)\n", name, res.ASCIIName, res.TableName(), res.VariantKey)

		variants, truncated := res.VariantLabels(10)
		for _, v := range variants {
			if v.Disposition == v3.DispositionAllocatable {
				fmt.Printf("   ↳ boleh dialokasikan ke pendaftar yang sama: %s (%s)\n", v.ULabel, v.ALabel)
			}
		}
		if truncated {
			fmt.Println("   ↳ (daftar varian dipotong)")
		}
	}

	for _, name := range []string{
		"contoh.id",
		"xn--caf-dma.id", // A-label dari EPP
		"ᬩᬮᬶ.id",
		"巖石.zh",
		"岩石.zh", // varian 巖石: ditolak
		"contoh.co.id",
		"www.contoh.id", // bukan nama yang bisa didaftarkan
		"xn--ls8h.id",   // emoji
		"日本.id",         // aksara tidak ada di tabel .id
	} {
		create(name)
	}

	rde, err := manager.RdeTableRefs("id")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("RDE idnTableRef untuk .id: %+v\n", rde)
}

// eppResult adalah contoh pemetaan kesalahan validasi ke result code EPP
// (RFC 5730 §3). Kebijakan pemetaannya keputusan registry.
func eppResult(err error) string {
	switch {
	case errors.Is(err, v3.ErrUnknownTld), errors.Is(err, v3.ErrTldDisabled),
		errors.Is(err, v3.ErrNotRegistrable),
		errors.Is(err, v3.ErrCodepointNotInTable), errors.Is(err, v3.ErrLanguageRule),
		errors.Is(err, v3.ErrNoIdnTable):
		return "2306" // Parameter value policy error
	default:
		return "2005" // Parameter value syntax error (struktur/IDNA2008)
	}
}
