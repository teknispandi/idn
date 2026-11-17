package main

import (
	"fmt"
	"log"

	idn "github.com/al0fayz/idn"
	v3 "github.com/al0fayz/idn/v3"
)

// ExampleJawa memperlihatkan validasi aksara Jawa (und-Java, LGR ICANN
// 13 Agustus 2026): langsung lewat validator, dan lewat IdnManager seperti
// yang dipakai registry/registrar.
func ExampleJawa() {
	fmt.Println("=== Aksara Jawa: validator ===")
	validator := &v3.JavaneseLanguageValidator{}
	for _, tt := range []struct {
		name  string
		label string
	}{
		{"✅ jawa", "ꦗꦮ"},
		{"✅ pasangan (ja + pangkon + wa)", "ꦗ꧀ꦮ"},
		{"✅ taling-tarung (ko)", "ꦏꦺꦴ"},
		{"✅ aksara rekan (za) + wulu", "ꦗ꦳ꦶ"},
		{"✅ cakra + wulu (kri)", "ꦏꦿꦶ"},
		{"❌ pangkon di awal", "꧀ꦗ"},
		{"❌ dua pasangan bertumpuk", "ꦏ꧀ꦏ꧀ꦏ"},
		{"❌ cecak telu bukan setelah ka/ga/ja/da/pa/wa", "ꦤ꦳"},
		{"❌ tarung tanpa taling", "ꦏꦴ"},
		{"❌ sandhangan setelah sandhangan", "ꦏꦶꦺ"},
	} {
		if err := validator.CheckLabelForLanguage(tt.label); err != nil {
			fmt.Printf("%-48s %s → %v\n", tt.name, tt.label, err)
		} else {
			fmt.Printf("%-48s %s → valid\n", tt.name, tt.label)
		}
	}

	fmt.Println("\n=== Aksara Jawa: lewat IdnManager ===")
	manager := v3.NewIdnManagerFS(idn.Tables())
	// Cukup tambahkan "und-java" ke daftar tabel zona.
	if err := manager.RegisterTld("id", []string{"latin", "und-bali", "und-java"}); err != nil {
		log.Fatal(err)
	}
	for _, name := range []string{"ꦗꦮ.id", "ꦲꦤꦕꦫꦏ.id", "ꦏ꧀ꦏ꧀ꦏ.id"} {
		res, err := manager.CheckDomain(name)
		if err != nil {
			fmt.Printf("❌ %-14s %v\n", name, err)
			continue
		}
		fmt.Printf("✅ %-14s simpan %s (tabel %s)\n", name, res.ASCIIName, res.TableName())
	}
}
