# IDN Configuration Examples

Cuplikan di bawah memperlihatkan API loader/manager. Untuk alur lengkap
registry dan registrar (IDNA2008, zona, `VariantKey`, RDE, pesan kesalahan)
lihat `registry.go` dan `registrar.go` di folder ini, dan `../v4` untuk LGR XML.

## Example 1: Basic Single Language TLD

```go
package main

import (
    "fmt"
    "log"
    v3 "github.com/al0fayz/idn/v3"
)

func main() {
    manager := v3.NewIdnManager("./idn")
    
    // Register TLD dengan single language
    err := manager.RegisterTld("com", []string{"latin"})
    if err != nil {
        log.Fatal(err)
    }
    
    // Validate domain labels
    testLabels := []string{"example", "café", "résumé"}
    for _, label := range testLabels {
        isValid, languages, _ := manager.ValidateLabel("com", label)
        fmt.Printf("%s: valid=%v, languages=%v\n", label, isValid, languages)
    }
}
```

## Example 2: Multi-Language TLD Configuration

```go
package main

import (
    "fmt"
    "log"
    v3 "github.com/al0fayz/idn/v3"
)

func main() {
    manager := v3.NewIdnManager("./idn")
    
    // TLD untuk wilayah Asia dengan multiple script.
    // Jangan pasang "chinese" dan "japanese" bersama: keduanya memuat aksara
    // Han yang sama dengan himpunan varian berbeda (1.001 aksara), sehingga
    // bentrok varian bisa lolos. CheckVariantCoherence di bawah menangkapnya.
    asiaLanguages := []string{"chinese", "korean", "thai", "khmer"}
    err := manager.RegisterTldWithPreload("asia", asiaLanguages)
    if err != nil {
        log.Fatal(err)
    }
    
    // TLD untuk wilayah Middle East
    middleEastLanguages := []string{"arabic", "hebrew", "armenian"}
    err = manager.RegisterTldWithPreload("me", middleEastLanguages)
    if err != nil {
        log.Fatal(err)
    }
    
    // TLD Global dengan banyak script
    globalLanguages := []string{
        "latin", "cyrillic", "greek", "armenian", 
        "arabic", "hebrew", "devanagari", "bengali",
    }
    err = manager.RegisterTldWithPreload("global", globalLanguages)
    if err != nil {
        log.Fatal(err)
    }

    // Tabel dengan varian dalam satu zona harus seragam variannya
    if err := manager.CheckVariantCoherence(); err != nil {
        log.Fatal(err)
    }
    
    // Test validation
    testCases := map[string][]string{
        "asia": {"예시", "ตัวอย่าง", "例子"},
        "me":   {"مثال", "דוגמה", "օրինակ"},
        "global": {"example", "пример", "παράδειγμα", "उदाहरण"},
    }
    
    for tld, labels := range testCases {
        fmt.Printf("\n=== Testing TLD: %s ===\n", tld)
        for _, label := range labels {
            isValid, languages, err := manager.ValidateLabel(tld, label)
            if err != nil {
                fmt.Printf("❌ %s: ERROR - %v\n", label, err)
                continue
            }
            if isValid {
                fmt.Printf("✅ %s: Valid (languages: %v)\n", label, languages)
            } else {
                fmt.Printf("❌ %s: Invalid\n", label)
            }
        }
    }
}
```

## Example 3: Dynamic Language Management

```go
package main

import (
    "fmt"
    "log"
    v3 "github.com/al0fayz/idn/v3"
)

func main() {
    manager := v3.NewIdnManager("./idn")
    
    // Start dengan basic configuration
    err := manager.RegisterTld("dynamic", []string{"latin"})
    if err != nil {
        log.Fatal(err)
    }
    
    // Tambahkan bahasa secara bertahap
    languagesToAdd := []string{"arabic", "chinese", "cyrillic", "greek"}
    
    for _, lang := range languagesToAdd {
        fmt.Printf("Adding %s language...\n", lang)
        err := manager.AddLanguageToTld("dynamic", lang)
        if err != nil {
            fmt.Printf("❌ Failed to add %s: %v\n", lang, err)
            continue
        }
        
        // Test dengan label dari bahasa yang baru ditambahkan
        var testLabel string
        switch lang {
        case "arabic":
            testLabel = "مثال"
        case "chinese":
            testLabel = "例子"
        case "cyrillic":
            testLabel = "пример"
        case "greek":
            testLabel = "παράδειγμα"
        }
        
        if testLabel != "" {
            isValid, err := manager.ValidateLabelForLanguage("dynamic", lang, testLabel)
            if err != nil {
                fmt.Printf("❌ Validation error for %s: %v\n", lang, err)
            } else {
                fmt.Printf("✅ %s language added and tested: %v\n", lang, isValid)
            }
        }
    }
    
    // Show final configuration
    config, _ := manager.GetTldConfig("dynamic")
    fmt.Printf("\nFinal configuration for 'dynamic' TLD:\n")
    fmt.Printf("Languages: %v\n", config.Languages)
    fmt.Printf("Enabled: %v\n", config.Enabled)
}
```

## Example 4: Performance Monitoring

```go
package main

import (
    "fmt"
    "time"
    v3 "github.com/al0fayz/idn/v3"
)

func main() {
    manager := v3.NewIdnManager("./idn")
    
    // Setup multiple TLDs
    tldConfigs := map[string][]string{
        "com":    {"latin"},
        "asia":   {"chinese", "japanese", "korean"},
        "arab":   {"arabic", "hebrew"},
        "global": {"latin", "cyrillic", "greek", "armenian"},
    }
    
    // Register all TLDs with timing
    fmt.Println("=== TLD Registration Performance ===")
    for tld, languages := range tldConfigs {
        start := time.Now()
        err := manager.RegisterTldWithPreload(tld, languages)
        duration := time.Since(start)
        
        if err != nil {
            fmt.Printf("❌ %s: Failed (%v)\n", tld, err)
        } else {
            fmt.Printf("✅ %s: %v (%d languages)\n", tld, duration, len(languages))
        }
    }
    
    // Performance test
    fmt.Println("\n=== Validation Performance Test ===")
    testLabels := []string{
        "example", "test-domain", "café", "мир", 
        "例え", "مثال", "παράδειγμα", "пример",
    }
    
    iterations := 1000
    totalValidations := 0
    
    start := time.Now()
    for i := 0; i < iterations; i++ {
        for tld := range tldConfigs {
            for _, label := range testLabels {
                _, _, _ = manager.ValidateLabel(tld, label)
                totalValidations++
            }
        }
    }
    totalTime := time.Since(start)
    
    fmt.Printf("Total validations: %d\n", totalValidations)
    fmt.Printf("Total time: %v\n", totalTime)
    fmt.Printf("Average per validation: %v\n", totalTime/time.Duration(totalValidations))
    fmt.Printf("Validations per second: %.0f\n", float64(totalValidations)/totalTime.Seconds())
    
    // Cache statistics
    fmt.Println("\n=== Cache Statistics ===")
    stats := manager.GetStats()
    fmt.Printf("Manager stats: %+v\n", stats)
}
```

## Example 5: Error Handling (gagal tertutup)

Bila tabel atau konfigurasi TLD tidak tersedia, hentikan start atau tolak
pendaftaran. Jangan jatuh ke validasi ASCII sederhana: registry yang
meloloskan nama saat tabelnya hilang menerima nama yang tidak bisa ditarik
kembali.

```go
package main

import (
    "errors"
    "fmt"
    "log"

    idn "github.com/al0fayz/idn"
    v3 "github.com/al0fayz/idn/v3"
)

func main() {
    manager := v3.NewIdnManagerFS(idn.Tables())

    // Tabel yang tidak ada membuat registrasi TLD gagal: hentikan start.
    if err := manager.RegisterTld("robust", []string{"latin", "nonexistent"}); err != nil {
        log.Printf("konfigurasi ditolak: %v", err)
    }
    if err := manager.RegisterTld("robust", []string{"latin"}); err != nil {
        log.Fatal(err)
    }

    for _, name := range []string{"example.robust", "invalid-chars-™.robust", "café.robust", "тест.robust", "contoh.unknown"} {
        res, err := manager.CheckDomain(name)
        var ve *v3.ValidationError
        switch {
        case err == nil:
            fmt.Printf("✅ %s → %s (tabel %q)\n", name, res.ASCIIName, res.TableName())
        case errors.Is(err, v3.ErrUnknownTld):
            fmt.Printf("❌ %s: zona tidak terdaftar\n", name)
        case errors.As(err, &ve) && ve.Rune != 0:
            fmt.Printf("❌ %s: karakter %q di posisi %d ditolak (%v)\n", name, ve.Rune, ve.Pos+1, ve.Kind)
        default:
            fmt.Printf("❌ %s: %v\n", name, err)
        }
    }
}
```

## Configuration File Example (JSON)

`v3.TldIdnConfig` sudah memiliki tag JSON, jadi konfigurasi zona bisa dibaca
langsung. URL di bawah hanya contoh (`.invalid`); isi dengan URL tabel milik
registry di IANA Repository of IDN Practices.

```json
{
  "idn_path": "",
  "tlds": [
    {
      "tld_name": "id",
      "languages": ["latin", "und-bali"],
      "enabled": true,
      "allow_ascii": true,
      "table_refs": {
        "latin":    {"url": "https://contoh.invalid/idn/latin.txt", "policy": "https://contoh.invalid/kebijakan-idn"},
        "und-bali": {"url": "https://contoh.invalid/idn/und-bali.xml", "policy": "https://contoh.invalid/kebijakan-idn"}
      }
    },
    {
      "tld_name": "co.id",
      "languages": ["latin"],
      "enabled": true,
      "allow_ascii": true,
      "table_refs": {
        "latin": {"url": "https://contoh.invalid/idn/latin.txt", "policy": "https://contoh.invalid/kebijakan-idn"}
      }
    }
  ]
}
```

`idn_path` kosong berarti memakai tabel yang disematkan di binary.

## Best Practice Configuration

```go
package main

import (
    "encoding/json"
    "errors"
    "log"
    "os"

    idn "github.com/al0fayz/idn"
    v3 "github.com/al0fayz/idn/v3"
)

type Config struct {
    IdnPath string            `json:"idn_path"`
    TLDs    []v3.TldIdnConfig `json:"tlds"`
}

func main() {
    config, err := loadConfig("config.json")
    if err != nil {
        log.Fatal(err)
    }

    manager := v3.NewIdnManagerFS(idn.Tables())
    if config.IdnPath != "" {
        manager = v3.NewIdnManager(config.IdnPath)
    }

    // TLD nonaktif tetap didaftarkan, agar validasinya mengembalikan
    // ErrTldDisabled (bukan ErrUnknownTld).
    for _, tld := range config.TLDs {
        if err := manager.RegisterTldConfig(tld); err != nil {
            log.Fatalf("TLD %s: %v", tld.TldName, err)
        }
    }

    // Gagal di sini lebih baik daripada saat deposit RDE atau saat bentrok
    // varian lolos.
    if err := errors.Join(manager.CheckRdeTableRefs(), manager.CheckVariantCoherence()); err != nil {
        log.Fatal(err)
    }
    log.Printf("IDN siap: %v", manager.ListRegisteredTlds())
}

func loadConfig(filename string) (*Config, error) {
    file, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var config Config
    err = json.NewDecoder(file).Decode(&config)
    return &config, err
}
```
