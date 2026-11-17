# IDN v4 — LGR XML (RFC 7940)

Package `v4` memuat Label Generation Ruleset format XML RFC 7940 — misalnya
file di [`lgrs/`](../lgrs) — dan memakainya untuk memvalidasi label,
menentukan disposisi, menghitung *index label*, dan membangkitkan varian.

`*v4.LGR` mengimplementasikan `v3.LabelTable` dan `v3.VariantTable`, jadi LGR
dipasang ke `v3.IdnManager` dan berbagi jalur yang sama dengan tabel `.txt`:
IDNA2008, pencocokan zona, `VariantKey`, `ValidationError`, dan referensi RDE.

```go
import (
    idn "github.com/al0fayz/idn"
    v3 "github.com/al0fayz/idn/v3"
    v4 "github.com/al0fayz/idn/v4"
)

manager := v3.NewIdnManagerFS(idn.Tables())

bali, err := v4.LoadFile("lgrs/second-level-reference/und-Bali.xml") // nama "und-bali"
if err != nil {
    log.Fatal(err) // LGR rusak/tidak didukung ditolak saat dimuat
}
manager.RegisterTable(bali)
manager.RegisterTld("id", []string{"latin", "und-bali"})

res, err := manager.CheckDomain("ᬓ.id")
// res.TableName() == "und-bali", res.VariantKey sama dengan milik "ᬔ.id"
variants, _ := res.VariantLabels(0) // [{ᬔ blocked}]
```

Dipakai sendiri, tanpa manager (label harus U-label yang sudah lolos IDNA2008):

```go
de, _ := v4.LoadFile("lgrs/second-level-reference/de.xml")
de.Disposition("straße")       // valid
de.VariantLabels("straße", 0)  // [{strasse allocatable [fallback]}]
de.VariantIndex("strasse") == de.VariantIndex("straße") // true
```

Nama tabel = nama file tanpa ekstensi (huruf kecil); ganti dengan `WithName`
bila LGR dan tabel `.txt` untuk aksara yang sama dipasang berdampingan.

`v4.SupportedUnicodeVersion()` adalah versi terendah di antara pustaka standar
Go, `x/net/idna`, dan `x/text` (saat ini 15.0.0). LGR yang `<unicode-version>`-nya
lebih baru (rilis ICANN 2025–2026 menyatakan 15.1/16.0) tetap dimuat selama
setiap code point di repertoire dan target variannya sudah dikenal runtime;
LGR yang memakai code point yang belum dikenal ditolak saat dimuat, karena
properti dan status PVALID-nya tidak bisa dievaluasi. Perbarui Go dan
dependensi itu untuk LGR seperti itu.

## Semantik yang diterapkan

| RFC 7940 | Implementasi |
|---|---|
| Repertoire (§5): `char`, sekuens, `range`, `tag` | Label dipecah menjadi elemen repertoire; semua partisi yang memenuhi konteks dipertimbangkan |
| `when` / `not-when` (§5.2, §6.4) | Aturan dengan `anchor` dievaluasi di posisi elemen; tanpa `anchor`, terhadap seluruh label |
| Aturan (§6.3) | `char`, `class`, `any`, `start`, `end`, `anchor`, `look-behind`, `look-ahead`, `choice`, `rule` (by-ref / inline), `count` (`n`, `n+`, `n:m`). Tanpa `start`/`end`, aturan dicari di mana saja dalam label |
| Kelas (§6.2) | daftar code point/rentang, `from-tag`, `by-ref`, `union`, `intersection`, `difference`, `symmetric-difference`, `complement`; properti `gc:*` dan `jt:R/D/L` |
| Aksi (§7) | berurutan, aksi pertama yang terpicu menang; `match`, `not-match`, `any-variant`, `all-variants`, `only-variants`; aksi default §7.6 ditambahkan di akhir |
| Label asli (§8.3) | invalid bila di luar repertoire/konteks gagal; tipe varian refleksif dicatat. **Hanya disposisi `valid` yang diterima `CheckLabel`** |
| Varian (§8.2) | semua permutasi pemetaan varian pada semua partisi; kondisi varian dievaluasi pada label asli; label varian harus valid sendiri; `invalid` dibuang |
| Index label (§8.5) | anggota terkecil himpunan varian (union-find atas semua pemetaan) |

Keputusan di luar teks RFC:

- Aksi berbasis tipe varian hanya terpicu bila ada tipe yang tercatat. Tanpa
  itu, aksi default `all-variants="activated"` akan berlaku untuk setiap label
  tanpa varian.
- Index label mengabaikan kondisi `when`/`not-when` varian, sehingga lebih
  ketat: label bisa dianggap bentrok walau varian bersyaratnya tidak berlaku,
  tidak pernah sebaliknya.
- Pembangkitan varian dibatasi `limit` (default `v3.DefaultVariantLimit`),
  64 partisi, dan 200.000 kombinasi; hasilnya ditandai `truncated`. Hanya
  varian yang juga lolos IDNA2008 yang dihitung terhadap `limit`. Untuk
  mendeteksi bentrok gunakan `VariantKey`, bukan daftar varian.

## Verifikasi

- Ke-146 LGR di `lgrs/` (rilis ICANN terbaru, lihat [lgrs/README.md](../lgrs/README.md))
  dimuat tanpa error (`TestLoadAllLGRs`).
- LGR aksara Jawa (`und-Java`, 13 Agustus 2026) diuji terhadap aturan yang
  dijelaskan di file-nya (`TestJavaneseLGR`).
- LGR `und-Bali.xml` dibandingkan dengan `v3.BalineseLanguageValidator` yang
  ditulis tangan — dua implementasi independen — untuk semua label 1–3
  karakter dari repertoire ditambah 100.000 label acak (391.918 label,
  246.575 di antaranya sah): tidak ada perbedaan.
- Disposisi varian `zh` (国/國), `de` (ß/ss), dan aturan joining type Arab
  (U+0649) diuji terhadap data di file LGR.

Belum diverifikasi terhadap implementasi referensi ICANN (`lgr-core`).

## Performance

Diukur dengan `go test ./v4 -run '^$' -bench . -benchmem` (Apple M1 Pro,
Go 1.26.1); jalankan ulang di server Anda.

| Operasi | Waktu | Alokasi |
|---|---|---|
| `CheckLabel` und-Bali (`ᬩᬮᬶᬓ᭄ᬱ`) | 7,8 µs | 6,5 KB |
| `CheckLabel` zh (`中国岩石`) | 1,0 µs | 437 B |
| `CheckLabel` und-Arab (`مثال`) | 5,4 µs | 5,7 KB |
| `VariantLabels` zh (`中国`) | 5,2 µs | 3,0 KB |
| Memuat und-Bali.xml | 0,6 ms | 313 KB |
| Memuat zh.xml (3,5 MB) | 153 ms | 92 MB dialokasikan |

Muat LGR sekali saat start; `*LGR` aman dipakai bersamaan.

Contoh lengkap: [`examples/v4`](../examples/v4) (`cd examples/v4 && go run .`).

## Data joining type

Properti `jt:` tidak tersedia di pustaka standar Go. `joiningtype_table.go`
dibangkitkan dari data Unicode milik `golang.org/x/text` dengan menguji aturan
konteks ZWNJ PRECIS (lihat [internal/gen/joiningtype.go](internal/gen/joiningtype.go)):

```bash
go generate ./v4
```

Hanya tipe R, D dan L yang dihasilkan, dan hanya untuk code point yang valid
di PRECIS IdentifierClass (≈ PVALID) — cukup untuk repertoire LGR. LGR yang
memakai properti lain ditolak saat dimuat.
