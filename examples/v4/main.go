// Contoh pemakaian LGR XML (RFC 7940) dengan package v4.
// Jalankan dari folder ini: cd examples/v4 && go run .
package main

import (
	"fmt"
	"log"

	idn "github.com/al0fayz/idn"
	v3 "github.com/al0fayz/idn/v3"
	v4 "github.com/al0fayz/idn/v4"
)

const lgrDir = "../../lgrs/second-level-reference/"

func main() {
	bali := load("und-Bali")
	de := load("de")
	zh := load("zh")

	fmt.Println("=== Disposisi label asli ===")
	for _, c := range []struct {
		lgr   *v4.LGR
		label string
	}{
		{bali, "ᬩᬮᬶ"},
		{bali, "ᬓ᭄ᬓ᭄ᬓ"}, // dua gantungan di bawah satu basis
		{bali, "᭄ᬓ"},    // adeg-adeg di awal
		{de, "straße"},
		{zh, "中国"},
	} {
		disp, err := c.lgr.Disposition(c.label)
		fmt.Printf("%-9s %-8s %-8s %v\n", c.lgr.GetName(), c.label, disp, errOrEmpty(err))
	}

	fmt.Println("\n=== Varian dan index label ===")
	for _, c := range []struct {
		lgr   *v4.LGR
		label string
	}{
		{de, "straße"},
		{de, "strasse"},
		{zh, "国"},
		{bali, "ᬓ"},
	} {
		variants, _ := c.lgr.VariantLabels(c.label, 10)
		fmt.Printf("%-8s index %-8s →", c.label, c.lgr.VariantIndex(c.label))
		for _, v := range variants {
			fmt.Printf(" %s(%s %v)", v.ULabel, v.Disposition, v.Types)
		}
		fmt.Println()
	}

	fmt.Println("\n=== Dipasang ke v3.IdnManager ===")
	manager := v3.NewIdnManagerFS(idn.Tables())
	for _, l := range []*v4.LGR{bali, zh} {
		if err := manager.RegisterTable(l); err != nil {
			log.Fatal(err)
		}
	}
	must(manager.RegisterTld("id", []string{"latin", "und-bali"}))
	for _, name := range []string{"ᬓ.id", "ᬔ.id"} {
		res, err := manager.CheckDomain(name)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s → %s tabel %s variant_key %s\n", name, res.ASCIIName, res.TableName(), res.VariantKey)
	}
	fmt.Println("(variant_key sama → registry menolak nama kedua)")

	// Tabel .txt Chinese dan LGR zh berbeda data variannya: memasang
	// keduanya di satu zona melanggar Guideline 13 ICANN.
	must(manager.RegisterTld("cn-campur", []string{"chinese", "zh"}))
	conflicts, err := manager.VariantConflicts("cn-campur")
	if err != nil {
		log.Fatal(err)
	}
	for _, c := range conflicts {
		fmt.Println("⚠️ ", c)
	}
}

func load(name string) *v4.LGR {
	l, err := v4.LoadFile(lgrDir + name + ".xml")
	if err != nil {
		log.Fatal(err)
	}
	return l
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func errOrEmpty(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
