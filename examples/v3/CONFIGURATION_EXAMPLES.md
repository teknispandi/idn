# IDN Configuration Examples

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
    
    // TLD untuk wilayah Asia dengan multiple script
    asiaLanguages := []string{"chinese", "japanese", "korean", "thai", "khmer"}
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
    
    // Test validation
    testCases := map[string][]string{
        "asia": {"例え", "예시", "ตัวอย่าง", "例子"},
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

## Example 5: Error Handling and Fallback

```go
package main

import (
    "fmt"
    "log"
    "strings"
    v3 "github.com/al0fayz/idn/v3"
)

func main() {
    manager := v3.NewIdnManager("./idn")
    
    // Try to register TLD with both existing and non-existing languages
    mixedLanguages := []string{"latin", "arabic", "nonexistent", "chinese", "invalid-lang"}
    
    fmt.Println("=== Robust TLD Registration ===")
    var validLanguages []string
    
    for _, lang := range mixedLanguages {
        // Try to load individual language first
        loader := v3.NewIdnLoader("./idn")
        _, err := loader.LoadTable(lang)
        if err != nil {
            if strings.Contains(err.Error(), "IDN file not found") {
                fmt.Printf("⚠️  Language '%s' not available, skipping\n", lang)
                continue
            } else {
                fmt.Printf("❌ Error loading '%s': %v\n", lang, err)
                continue
            }
        }
        
        validLanguages = append(validLanguages, lang)
        fmt.Printf("✅ Language '%s' validated\n", lang)
    }
    
    // Register with only valid languages
    if len(validLanguages) > 0 {
        err := manager.RegisterTld("robust", validLanguages)
        if err != nil {
            log.Printf("Failed to register TLD: %v", err)
        } else {
            fmt.Printf("✅ TLD 'robust' registered with languages: %v\n", validLanguages)
        }
    }
    
    // Validation with error handling
    fmt.Println("\n=== Robust Validation ===")
    testLabels := []string{"example", "invalid-chars-™", "café", "тест"}
    
    for _, label := range testLabels {
        isValid, languages, err := manager.ValidateLabel("robust", label)
        
        if err != nil {
            // Fallback to basic validation
            fmt.Printf("⚠️  %s: Validation error (%v), using fallback\n", label, err)
            isBasicASCII := isBasicASCII(label)
            fmt.Printf("   Fallback result: %v\n", isBasicASCII)
        } else {
            if isValid {
                fmt.Printf("✅ %s: Valid (supported by: %v)\n", label, languages)
            } else {
                fmt.Printf("❌ %s: Invalid for this TLD\n", label)
            }
        }
    }
}

// Helper function untuk basic ASCII validation
func isBasicASCII(s string) bool {
    for _, r := range s {
        if r > 127 || (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && 
           (r < '0' || r > '9') && r != '-' && r != '.' {
            return false
        }
    }
    return len(s) > 0
}
```

## Configuration File Example (JSON)

```json
{
  "idn_path": "/path/to/idn/tables",
  "cache_config": {
    "enabled": true,
    "preload_common": true,
    "max_cached_languages": 50
  },
  "tlds": {
    "com": {
      "languages": ["latin"],
      "enabled": true,
      "preload": true
    },
    "asia": {
      "languages": ["chinese", "japanese", "korean", "thai"],
      "enabled": true,
      "preload": true
    },
    "global": {
      "languages": ["latin", "cyrillic", "arabic", "chinese", "japanese"],
      "enabled": true,
      "preload": false
    }
  }
}
```

## Best Practice Configuration

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"
    v3 "github.com/al0fayz/idn/v3"
)

type Config struct {
    IdnPath     string `json:"idn_path"`
    CacheConfig struct {
        Enabled           bool `json:"enabled"`
        PreloadCommon     bool `json:"preload_common"`
        MaxCachedLanguages int `json:"max_cached_languages"`
    } `json:"cache_config"`
    TLDs map[string]struct {
        Languages []string `json:"languages"`
        Enabled   bool     `json:"enabled"`
        Preload   bool     `json:"preload"`
    } `json:"tlds"`
}

func main() {
    // Load configuration
    config, err := loadConfig("config.json")
    if err != nil {
        fmt.Printf("Failed to load config: %v\n", err)
        return
    }
    
    // Initialize manager
    manager := v3.NewIdnManager(config.IdnPath)
    
    // Register TLDs based on configuration
    for tldName, tldConfig := range config.TLDs {
        if !tldConfig.Enabled {
            fmt.Printf("Skipping disabled TLD: %s\n", tldName)
            continue
        }
        
        var err error
        if tldConfig.Preload {
            err = manager.RegisterTldWithPreload(tldName, tldConfig.Languages)
        } else {
            err = manager.RegisterTld(tldName, tldConfig.Languages)
        }
        
        if err != nil {
            fmt.Printf("❌ Failed to register TLD %s: %v\n", tldName, err)
        } else {
            fmt.Printf("✅ Registered TLD %s with %d languages\n", tldName, len(tldConfig.Languages))
        }
    }
    
    // Show final statistics
    stats := manager.GetStats()
    fmt.Printf("\nFinal statistics: %+v\n", stats)
}

func loadConfig(filename string) (*Config, error) {
    file, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    var config Config
    decoder := json.NewDecoder(file)
    err = decoder.Decode(&config)
    return &config, err
}
```
