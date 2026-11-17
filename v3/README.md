# IDN (Internationalized Domain Names) V3

## Overview

IDN V3 menyediakan sistem yang diperbaharui untuk mengelola dan memvalidasi Internationalized Domain Names (IDN) dengan dukungan:

- **Memory Cache**: Sistem cache di memory untuk menghindari pembacaan file berulang
- **Multiple IDN Support**: Mendukung lebih dari 1 IDN table per TLD
- **Easy Initialization**: Inisialisasi IDN yang mudah dengan cache berdasarkan bahasa
- **Dynamic Management**: Pengelolaan bahasa dan TLD secara dinamis

## Features

### 1. IDN Loader (`idn_loader.go`)
- Cache IDN tables di memory dengan thread-safe operation
- Auto-discovery file IDN dari folder `/idn`
- Preloading untuk performa optimal
- Statistics dan monitoring cache

### 2. IDN Manager (`idn_manager.go`)
- Manajemen multiple TLD dengan multiple language support
- Dynamic add/remove bahasa per TLD
- Enable/disable TLD functionality
- Comprehensive validation methods

### 3. Enhanced IDN Table (`idn.go`)
- Support multiple format file IDN (Arabic, Chinese, etc.)
- Flexible URL dan Policy parsing
- Backward compatibility dengan format lama

### 4. Validasi Registry/Registrar (`idna2008.go`, `result.go`, `variants.go`)
- IDNA2008 lengkap: A-label kanonik, PVALID, CONTEXTO/CONTEXTJ, bidi
- `CheckDomain`: zona, tabel IDN, `VariantKey` untuk deteksi bentrok varian
- Kesalahan bertipe (`*ValidationError`) untuk pesan EPP dan pengguna
- Referensi RDE dan pemeriksaan koherensi varian antar tabel

## Installation

```bash
go get github.com/al0fayz/idn/v3
```

## Quick Start

### Simple Usage

```go
package main

import (
    "fmt"
    "path/filepath"
    v3 "github.com/al0fayz/idn/v3"
)

func main() {
    // Setup IDN path
    idnPath := filepath.Join("path", "to", "idn")
    
    // Method 1: Using IDN Loader directly
    loader := v3.NewIdnLoader(idnPath)
    table, err := loader.LoadTable("arabic")
    if err != nil {
        panic(err)
    }
    
    isValid := table.IsValidLabel("مثال") // Arabic text
    fmt.Printf("Arabic label valid: %v\n", isValid)
    
    // Method 2: Using IDN Manager
    manager := v3.NewIdnManager(idnPath)
    err = manager.RegisterTld("example", []string{"latin", "arabic"})
    if err != nil {
        panic(err)
    }
    
    isValid, languages, err := manager.ValidateLabel("example", "café")
    fmt.Printf("Label valid: %v, supported by: %v\n", isValid, languages)
}
```

### Advanced Usage

```go
// Register TLD with multiple languages
err := manager.RegisterTld("global", []string{
    "latin", "cyrillic", "arabic", "chinese", "japanese",
})

// Preload untuk performa lebih baik
err = manager.RegisterTldWithPreload("multilingual", []string{
    "devanagari", "bengali", "tamil", "korean",
})

// Validasi per bahasa spesifik
isValid, err := manager.ValidateLabelForLanguage("global", "arabic", "مثال")

// Dynamic language management
err = manager.AddLanguageToTld("global", "armenian")
err = manager.RemoveLanguageFromTld("global", "chinese")

// Enable/disable TLD
err = manager.DisableTld("global")
err = manager.EnableTld("global")
```

## Penggunaan untuk Registry & Registrar

Satu jalur validasi dipakai kedua sisi, sehingga registrar bisa menolak nama
**sebelum** dikirim lewat EPP dengan alasan yang sama persis dengan registry.

Urutan pemeriksaan `IdnManager.CheckDomain`:

1. **Struktur** — label tidak kosong, maks. 63 oktet (diukur dalam bentuk
   A-label), total maks. 253 oktet, aturan tanda hubung RFC 5891 §4.2.3.1
   (posisi 3–4 dihitung per *code point*).
2. **A-label** — `xn--…` didekode lalu divalidasi sebagai U-label, dan harus
   kanonik (encode ulang = input).
3. **IDNA2008** — NFC, tidak diawali tanda kombinasi, code point PVALID
   (RFC 5892), CONTEXTO/CONTEXTJ, aturan bidi (RFC 5893).
4. **Zona** — akhiran terpanjang yang terdaftar (`co.id` sebelum `id`); nama
   harus tepat satu label di bawah zona.
5. **Tabel IDN TLD** — tabel pertama (sesuai urutan konfigurasi) yang
   menerima label, termasuk aturan bahasanya (Bali, Arab, `ja`).

### Registry

```go
import (
    idn "github.com/al0fayz/idn"
    v3 "github.com/al0fayz/idn/v3"
)

manager := v3.NewIdnManagerFS(idn.Tables()) // atau v3.NewIdnManager("/etc/registry/idn")
manager.RegisterTld("id", []string{"latin", "und-bali"})
manager.RegisterTld("co.id", []string{"latin"})
// Zona khusus IDN: label ASCII murni tidak diizinkan tanpa tabel.
manager.RegisterTldConfig(v3.TldIdnConfig{TldName: "рф", Languages: []string{"cyrillic"}, Enabled: true})

res, err := manager.CheckDomain("xn--caf-dma.id") // input EPP, U-label juga boleh
if err != nil {
    return err // *v3.ValidationError
}
store(res.ASCIIName, res.VariantKey) // UNIQUE(zone, variant_key) mendeteksi bentrok varian
rde(res.TableName())                 // "latin" — idnTableId untuk deposit RDE ("" untuk nama ASCII)
```

**Varian.** `VariantKey` adalah *index label* (RFC 7940 §8.5): setiap code
point diganti anggota terkecil himpunan variannya. Dua nama saling varian jika
dan hanya jika kuncinya sama, jadi satu indeks unik `(zone, variant_key)`
sudah cukup — tanpa membangkitkan semua varian. `res.VariantLabels(limit)`
mengembalikan varian beserta disposisinya (`blocked`, `allocatable`, …) untuk
ditawarkan ke pendaftar yang sama. Sumber varian:

- tabel `.txt` format RFC 3743 (`Chinese-IDN.txt`, `Traditional-Chinese-IDN.txt`):
  varian *preferred* (kolom 2) → `allocatable`, lainnya → `blocked`;
- LGR XML (package [v4](../v4)) — disposisi menurut aksi LGR.

**Koherensi varian.** Index dihitung oleh tabel yang menerima label, jadi
tabel-tabel dalam satu zona harus sepakat tentang himpunan varian setiap code
point yang mereka miliki bersama — Guideline 13 ICANN IDN Implementation
Guidelines (diusulkan di v4.0, ditangguhkan di v4.1; lihat
`doc/idn-guidelines-22sep22-en.pdf`). `manager.VariantConflicts(tld)` /
`manager.CheckVariantCoherence()`
memeriksanya. Contoh nyata dari tabel di repo ini: `chinese` + `japanese` dalam
satu zona berbeda untuk 1.001 aksara Han (mis. 亜: `chinese`→亚,
`japanese`→亜), dan `chinese` (`.txt`) + LGR `zh` berbeda untuk 46 aksara.

**RDE.** Header `# URL:`/`# Policy:` tabel Google di `idn/` menunjuk ke
registrasi Google, bukan milik registry Anda. Isi `TableRefs` dan periksa saat
registry mulai:

```go
manager.RegisterTldConfig(v3.TldIdnConfig{
    TldName: "id", Languages: []string{"latin", "und-bali"}, Enabled: true, AllowASCII: true,
    TableRefs: map[string]v3.TableRef{
        "latin":    {URL: "https://www.iana.org/domains/idn-tables/tables/…", Policy: "https://…"},
        "und-bali": {URL: "https://www.iana.org/domains/idn-tables/tables/…", Policy: "https://…"},
    },
})
if err := manager.CheckRdeTableRefs(); err != nil { log.Fatal(err) }
if err := manager.CheckVariantCoherence(); err != nil { log.Fatal(err) }
refs, _ := manager.RdeTableRefs("id") // untuk <rdeIDN:idnTableRef>
```

**LGR XML.** Tabel dari LGR RFC 7940 dipasang dengan `RegisterTable` lalu
dipakai di `Languages` seperti tabel `.txt`:

```go
bali, _ := v4.LoadFile("lgrs/second-level-reference/und-Bali.xml") // nama "und-bali"
manager.RegisterTable(bali) // menggantikan idn/und-Bali.txt, lengkap dengan varian
manager.RegisterTld("id", []string{"latin", "und-bali"})
```

Jika registrar menyebut tabel IDN secara eksplisit (ekstensi EPP IDN), pakai
`manager.CheckDomainForTable(name, "und-bali")`.

### Registrar

```go
name, _ := v3.DefaultIdnUtils.NormalizeDomainName(userInput) // "Café。ID " → "café.id"
res, err := manager.CheckDomain(name)
if err != nil {
    var ve *v3.ValidationError
    if errors.As(err, &ve) {
        // ve.Kind, ve.Rune, ve.Pos, ve.Table, ve.Rule → pesan untuk pengguna
    }
    return err
}
showToUser(res.UnicodeName) // "café.id"
sendEPP(res.ASCIIName)      // "xn--caf-dma.id"
```

### Jenis kesalahan

Semua penolakan berupa `*ValidationError`; bedakan dengan `errors.Is`:

| Error | Arti |
|---|---|
| `ErrEmptyLabel`, `ErrLabelTooLong`, `ErrDomainTooLong` | panjang label/nama |
| `ErrLeadingHyphen`, `ErrTrailingHyphen`, `ErrHyphen34`, `ErrInvalidLDH` | aturan LDH / tanda hubung |
| `ErrInvalidALabel` | punycode rusak, tidak kanonik, atau dekode ke ASCII saja |
| `ErrNotNFC`, `ErrLeadingCombiningMark`, `ErrDisallowedCodepoint`, `ErrContextRule`, `ErrBidiRule` | IDNA2008 |
| `ErrCodepointNotInTable`, `ErrLanguageRule`, `ErrNoIdnTable` | kebijakan tabel IDN TLD |
| `ErrUnknownTld`, `ErrTldDisabled`, `ErrNotRegistrable` | zona tidak terdaftar / nonaktif / bukan nama yang bisa didaftarkan |

Jika beberapa tabel menolak, yang dilaporkan adalah tabel yang menolak paling
belakang dalam label (alasan yang paling relevan bagi pengguna).

### Catatan kebijakan

- Huruf ASCII A–Z diturunkan otomatis; huruf besar non-ASCII (mis. `É`)
  ditolak — normalisasi input pengguna dengan `NormalizeDomainName`.
- `RegisterTld` mengizinkan label LDH murni tanpa tabel (`AllowASCII: true`);
  kunci variannya tetap dihitung dari tabel TLD (varian bisa melintasi ASCII,
  mis. `ß` ↔ `ss`).
- `RdeTableRefs` tidak pernah mengarang URL: tabel tanpa referensi membuatnya
  mengembalikan error.

## File Structure

```
/idn/
  ├── Arabic-IDN.txt          # Arabic script IDN table
  ├── Chinese-IDN.txt         # Chinese script IDN table (dengan varian RFC 3743)
  ├── Japanese-IDN.txt        # Japanese script IDN table
  ├── Korean-IDN.txt          # Korean script IDN table
  ├── Latin-IDN.txt           # Latin script IDN table
  ├── und-Bali.txt            # Balinese (dari LGR second-level ICANN)
  └── ...                     # Other language tables
```

Folder ini juga disematkan ke binary lewat package root modul
(`idn.Tables()`, ±2 MB), sehingga server tidak perlu salinan file tabel.

## Supported IDN Formats

### Standard Format
```
# Registry: Example Registry
# Script: Arab
# Version: 1.0
# URL: https://example.com/idn-table
# Policy: https://example.com/idn-policy

U+0061  # LATIN SMALL LETTER A
U+0062  # LATIN SMALL LETTER B
U+002D  # HYPHEN-MINUS
```

### Chinese Format
```
U+002D(0);U+002D(0);
U+0030(0);U+0030(0);
U+4E00(0);U+4E00(0);
```

### Arabic Format with Comments
```
U+0660  # ARABIC-INDIC DIGIT ZERO
U+0661  # ARABIC-INDIC DIGIT ONE
U+0627  # ARABIC LETTER ALEF
```

### Rentang
```
U+0061..U+007A  # a..z
```

Baris yang memuat beberapa entri `U+XXXX # …` (seperti beberapa baris di
`Greek-IDN.txt`) dibaca semua entrinya.

## Performance

Diukur dengan `go test ./v3 -run '^$' -bench . -benchmem` (Apple M1 Pro,
Go 1.26.1). Angka akan berbeda di mesin lain; jalankan ulang di server Anda.

| Operasi | Waktu | Alokasi |
|---|---|---|
| `CheckDomain("contoh-domain.id")` (ASCII) | 0,7 µs | 496 B |
| `CheckDomain("café-kopi.id")` (IDN Latin) | 1,4 µs | 672 B |
| `CheckDomain("xn--caf-kopi-e2a.id")` (A-label) | 1,5 µs | 800 B |
| `CheckDomain("ᬩᬮᬶ.id")` (Bali, aturan bahasa) | 1,9 µs | 864 B |
| `CheckDomain("中国岩石.zh")` (dengan `VariantKey`) | 2,4 µs | 864 B |
| `CheckDomain("xn--ls8h.id")` (ditolak) | 0,6 µs | 376 B |
| `VariantLabels` untuk `巖石` (7 varian) | 6,6 µs | 2,7 KB |
| Memuat `latin` tanpa cache | 49 µs | 43 KB |
| Memuat `chinese` tanpa cache (19.557 code point + varian) | 31 ms | 24,7 MB dialokasikan |

Tabel dimuat sekali per proses dan di-cache (`IdnLoader`, aman untuk banyak
goroutine); `IdnManager` juga aman dipakai bersamaan. Muat tabel saat start
(`RegisterTld`) agar request pertama tidak menanggung biaya pemuatan.

## Examples

Lihat folder `/examples/v3/` (jalankan dari folder itu; `main.go` memanggil
satu fungsi — aktifkan yang ingin dicoba):

- `registry.go` — `ExampleRegistry`: konfigurasi zona, pemeriksaan saat start,
  validasi create, bentrok varian lewat `VariantKey`, referensi RDE
- `registrar.go` — `ExampleRegistrar`: normalisasi input pengguna dan pesan
  kesalahan berbahasa Indonesia
- `bali.go` — validator aksara Bali
- `simple_usage.go`, `main_enhanced.go`, `performance_demo.go` — API loader/manager

Contoh LGR XML ada di `/examples/v4/`.

## Configuration

### Programmatic Configuration
```go
// Custom validator
validator := &CustomLanguageValidator{}
table, err := NewIdnTable("custom", data, validator)

// Cache management
loader.ClearCache()
stats := loader.GetCacheStats()
```

## Error Handling

```go
// Tabel yang tidak ada: gagal saat start, jangan dilanjutkan tanpa tabel.
if err := manager.RegisterTld("id", []string{"latin", "und-bali"}); err != nil {
    log.Fatal(err)
}

// Penolakan label: bedakan jenisnya dengan errors.Is / errors.As.
_, err := manager.CheckDomain(name)
var ve *v3.ValidationError
switch {
case err == nil:
case errors.Is(err, v3.ErrCodepointNotInTable) && errors.As(err, &ve):
    log.Printf("karakter %q di posisi %d tidak ada di tabel %s", ve.Rune, ve.Pos, ve.Table)
case errors.Is(err, v3.ErrUnknownTld):
    log.Printf("zona tidak terdaftar")
default:
    log.Printf("ditolak: %v", err)
}
```

## Best Practices

### 1. Use Preloading for Known Languages
```go
// Preload commonly used languages at startup
commonLanguages := []string{"latin", "chinese", "arabic", "japanese"}
err := manager.RegisterTldWithPreload("global", commonLanguages)
```

### 2. Monitor Cache Performance
```go
// Regular cache monitoring
stats := loader.GetCacheStats()
log.Printf("Cache efficiency: %d cached languages", stats["cached_languages"])
```

### 3. Gagal Tertutup (fail closed)
Jika tabel atau konfigurasi TLD tidak tersedia, tolak pendaftaran — jangan
jatuh ke validasi ASCII atau validator permisif. Registry yang meloloskan nama
saat tabelnya hilang menerima nama yang tidak bisa ditarik kembali.

```go
res, err := manager.CheckDomain(name)
if err != nil {
    return err // termasuk ErrUnknownTld / ErrTldDisabled
}
```

### 4. Use Appropriate Method
```go
// Registry dan registrar: validasi lengkap nama yang akan didaftarkan
res, err := manager.CheckDomain("café.id")

// Satu label untuk TLD tertentu (IDNA2008 + tabel)
res, err := manager.CheckLabel("id", "café")

// Hanya cek tabel untuk U-label yang sudah divalidasi
err := table.CheckLabel("café")
```

### 5. Dynamic Management
```go
// Add language to existing TLD
manager.AddLanguageToTld("global", "armenian")

// Remove language from TLD  
manager.RemoveLanguageFromTld("global", "chinese")

// Enable/disable TLD
manager.DisableTld("global")
manager.EnableTld("global")
```

## Testing

```bash
go test ./...          # semua test
go test -race ./...    # termasuk pemakaian bersamaan IdnManager/LGR
go test ./v3 -run CheckDomain -v
```

Test yang menjaga kebenaran terhadap sumber luar:

- `TestIDNA2008AgainstIANA` — baris tabel turunan IANA (IDNA2008, Unicode 12.0.0)
- `TestIDNA2008MatchesPrecis` — pembanding PRECIS (`x/text`) untuk semua code point
- `TestTablesAreIDNA2008Valid` — setiap code point tabel lolos lapisan IDNA2008
- `TestBalineseRepertoireMatchesLGR` — repertoire Bali sama dengan LGR XML

## API Documentation

### Core Classes

#### IdnLoader
```go
func NewIdnLoader(idnPath string) *IdnLoader
func (loader *IdnLoader) LoadTable(language string) (*IdnTable, error)
func (loader *IdnLoader) LoadMultipleTables(languages []string) (map[string]*IdnTable, error)
func (loader *IdnLoader) ListAvailableLanguages() ([]string, error)
func (loader *IdnLoader) PreloadLanguages(languages []string) error
func (loader *IdnLoader) GetCacheStats() map[string]interface{}
```

#### IdnManager  
```go
func NewIdnManager(idnPath string) *IdnManager
func NewIdnManagerFS(fsys fs.FS) *IdnManager
func (manager *IdnManager) RegisterTld(tldName string, languages []string) error
func (manager *IdnManager) RegisterTldConfig(config TldIdnConfig) error
func (manager *IdnManager) RegisterTable(table LabelTable) error
func (manager *IdnManager) CheckDomain(domain string) (*DomainResult, error)
func (manager *IdnManager) CheckDomainForTable(domain, table string) (*DomainResult, error)
func (manager *IdnManager) CheckLabel(tldName, label string) (*LabelResult, error)
func (manager *IdnManager) RdeTableRefs(tldName string) ([]TableRef, error)
func (manager *IdnManager) CheckRdeTableRefs() error
func (manager *IdnManager) VariantConflicts(tldName string) ([]VariantConflict, error)
func (manager *IdnManager) CheckVariantCoherence() error
func (manager *IdnManager) ValidateLabel(tldName, label string) (bool, []string, error)
func (manager *IdnManager) ValidateLabelForLanguage(tldName, language, label string) (bool, error)
func (manager *IdnManager) AddLanguageToTld(tldName, language string) error
func (manager *IdnManager) RemoveLanguageFromTld(tldName, language string) error
```

#### Hasil validasi
```go
type DomainResult struct {
    LabelResult              // ULabel, ALabel, IsIDN, Table, VariantKey
    Zone, ASCIIName, UnicodeName string
}
func (r *LabelResult) TableName() string
func (r *LabelResult) VariantLabels(limit int) ([]VariantLabel, bool)
```

## Perubahan Perilaku sejak v1.0.2

Signature lama tetap ada, tetapi beberapa perilaku diperketat karena versi
v1.0.2 meloloskan nama yang tidak valid:

| Sebelumnya (v1.0.2) | Sekarang |
|---|---|
| A-label (`xn--…`) tidak didekode, lolos sebagai ASCII | didekode, divalidasi, harus kanonik |
| Label kosong, `-abc`, `ab--cd` dianggap valid | ditolak |
| `DisableTld` tidak berpengaruh | validasi TLD nonaktif mengembalikan `ErrTldDisabled` |
| Urutan bahasa dari `ValidateLabel` acak | sesuai urutan konfigurasi |
| `ToASCII`/`ToUnicode` tanpa validasi (`"CAFÉ"` → `xn--CAF-pia`) | IDNA2008 ketat; error untuk input tidak valid |
| `GetURL`/`GetPolicy` mengarang URL IANA | `nil` bila header tidak ada |
| `IsValidDomainLabel` mengukur byte UTF-8 | mengukur oktet A-label, aturan IDNA2008 |
| `und-Bali.txt` tidak bisa dimuat loader | dimuat sebagai `und-bali` |

Fitur baru (`CheckDomain`, varian, RDE, LGR v4) hanya menambah API.

## Supported IDN Languages

Based on `/idn` folder content:
- Arabic, Armenian, Bengali, Chinese, Cyrillic
- Devanagari, Ethiopic, Georgian, Greek, Gurmukhi
- Hebrew, Japanese, Kannada, Khmer, Korean
- Latin, Malayalam, Myanmar, Oriya, Sinhala
- Tamil, Telugu, Thai, Tibetan, Traditional-Chinese
- und-Bali (Balinese, dari LGR second-level ICANN)


## Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

## License

Licensed under the Apache License, Version 2.0. See LICENSE file for details.

## Support

For questions and support:
- Create an issue in the GitHub repository
- Check existing documentation in `/doc` folder
- Review examples in `/examples/v3` folder
