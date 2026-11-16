// Package idn menyematkan tabel IDN dari folder idn/ ke dalam binary,
// sehingga registry maupun registrar bisa memvalidasi tanpa menyalin file
// tabel ke server:
//
//	manager := v3.NewIdnManagerFS(idn.Tables())
//
// Package ini hanya berisi data; logika validasi ada di package v3.
// Import package ini menambah ±2 MB ke binary.
package idn

import (
	"embed"
	"io/fs"
)

//go:embed idn/*.txt
var embedded embed.FS

// Tables mengembalikan tabel IDN yang disematkan, dengan file tabel di
// root fs.FS (mis. "Latin-IDN.txt", "und-Bali.txt").
func Tables() fs.FS {
	sub, err := fs.Sub(embedded, "idn")
	if err != nil {
		panic(err) // pola embed tetap; tidak mungkin terjadi
	}
	return sub
}
