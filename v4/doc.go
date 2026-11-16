// Package v4 memvalidasi label domain dengan Label Generation Ruleset (LGR)
// format XML RFC 7940, misalnya file di lgrs/. LGR dipasang ke
// v3.IdnManager lewat RegisterTable, sehingga validasi IDNA2008, zona dan
// kunci varian tetap satu jalur dengan tabel .txt.
package v4

//go:generate go run ./internal/gen/joiningtype.go
