package main

import (
	"errors"
	"fmt"
	"log"

	idn "github.com/al0fayz/idn"
	v3 "github.com/al0fayz/idn/v3"
)

// ExampleRegistrar memperlihatkan alur di sisi registrar: input pengguna
// dinormalisasi, divalidasi dengan kebijakan yang sama dengan registry,
// lalu ditampilkan dalam bentuk Unicode dan dikirim dalam bentuk A-label.
func ExampleRegistrar() {
	fmt.Println("=== Registrar ===")

	// Konfigurasi zona harus sama dengan registry (tabel yang disematkan
	// di binary sama dengan milik registry jika versinya sama).
	manager := v3.NewIdnManagerFS(idn.Tables())
	if err := manager.RegisterTld("id", []string{"latin", "und-bali"}); err != nil {
		log.Fatal(err)
	}

	for _, input := range []string{
		"  Café.ID ",
		"ＣＯＮＴＯＨ。ＩＤ", // lebar penuh + titik ideografik
		"ᬩᬮᬶ.id",
		"ab--cd.id",
		"-contoh.id",
		"日本.id",
		"ᬓ᭄ᬓ᭄ᬓ.id",
		"www.contoh.id",
		"contoh.com",
	} {
		name, err := v3.DefaultIdnUtils.NormalizeDomainName(input)
		if err == nil {
			var res *v3.DomainResult
			if res, err = manager.CheckDomain(name); err == nil {
				fmt.Printf("✅ %-20q tampilkan %s, kirim EPP %s\n", input, res.UnicodeName, res.ASCIIName)
				continue
			}
		}
		fmt.Printf("❌ %-20q %s\n", input, pesanUntukPengguna(err))
	}
}

// pesanUntukPengguna menerjemahkan kesalahan validasi menjadi pesan yang
// bisa ditampilkan ke pendaftar.
func pesanUntukPengguna(err error) string {
	var ve *v3.ValidationError
	errors.As(err, &ve)
	posisi := ""
	if ve != nil && ve.Rune != 0 {
		posisi = fmt.Sprintf(" (karakter %q di posisi %d)", ve.Rune, ve.Pos+1)
	}

	switch {
	case errors.Is(err, v3.ErrEmptyLabel):
		return "nama domain belum diisi"
	case errors.Is(err, v3.ErrLabelTooLong), errors.Is(err, v3.ErrDomainTooLong):
		return "nama domain terlalu panjang"
	case errors.Is(err, v3.ErrLeadingHyphen), errors.Is(err, v3.ErrTrailingHyphen):
		return "nama domain tidak boleh diawali atau diakhiri tanda hubung"
	case errors.Is(err, v3.ErrHyphen34):
		return "tanda hubung di posisi ke-3 dan ke-4 tidak diizinkan"
	case errors.Is(err, v3.ErrInvalidLDH), errors.Is(err, v3.ErrDisallowedCodepoint),
		errors.Is(err, v3.ErrCodepointNotInTable):
		return "nama domain memuat karakter yang tidak diizinkan" + posisi
	case errors.Is(err, v3.ErrLanguageRule):
		return "susunan aksara tidak sesuai aturan penulisan" + posisi
	case errors.Is(err, v3.ErrMixedScript):
		return "nama domain mencampur aksara yang berbeda" + posisi
	case errors.Is(err, v3.ErrNotRegistrable):
		return "hanya nama tingkat dua yang bisa didaftarkan, mis. contoh.id"
	case errors.Is(err, v3.ErrUnknownTld), errors.Is(err, v3.ErrTldDisabled):
		return "ekstensi domain tidak tersedia"
	default:
		return "nama domain tidak valid"
	}
}
