# IDN (Internationalized Domain Names) V3 Enhanced

## Overview

IDN V3 Enhanced menyediakan sistem yang diperbaharui untuk mengelola dan memvalidasi Internationalized Domain Names (IDN) dengan dukungan:

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

## File Structure

```
/idn/
  ├── Arabic-IDN.txt          # Arabic script IDN table
  ├── Chinese-IDN.txt         # Chinese script IDN table
  ├── Japanese-IDN.txt        # Japanese script IDN table
  ├── Korean-IDN.txt          # Korean script IDN table
  ├── Latin-IDN.txt           # Latin script IDN table
  ├── Cyrillic-IDN.txt        # Cyrillic script IDN table
  ├── Greek-IDN.txt           # Greek script IDN table
  └── ...                     # Other language tables
```

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

## Performance

### Memory Cache System
```go
type IdnLoader struct {
    idnPath string
    cache   map[string]*IdnTable  // Memory cache
    mutex   sync.RWMutex          // Thread-safe access
}
```
### Cache Performance
- **First Load**: ~695µs (from file)
- **Cached Load**: ~125ns (from memory)
- **Cache Speedup**: ~5500x faster
- **Validations/second**: ~2.5M validations

### Memory Usage
- Efficient memory usage dengan shared instances
- Thread-safe operations
- Automatic garbage collection untuk unused tables

## Examples

Lihat folder `/examples/v3/` untuk contoh lengkap:

- `simple_usage.go` - Basic usage examples
- `main_enhanced.go` - Advanced features demonstration  
- `performance_demo.go` - Performance testing dan benchmarks

## Configuration

### Environment Variables
```bash
export IDN_PATH="/path/to/idn/tables"
export IDN_CACHE_SIZE="100"  # Optional: max cached languages
```

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
// Handle missing IDN files
table, err := loader.LoadTable("nonexistent")
if err != nil {
    if strings.Contains(err.Error(), "IDN file not found") {
        // Handle missing file
        log.Printf("Language not supported: %v", err)
    } else {
        // Handle other errors
        return fmt.Errorf("failed to load IDN table: %w", err)
    }
}

// Handle validation errors
isValid, languages, err := manager.ValidateLabel("tld", "label")
if err != nil {
    return fmt.Errorf("validation failed: %w", err)
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

### 3. Handle Errors Gracefully
```go
// Fallback untuk missing languages
isValid, languages, err := manager.ValidateLabel("tld", "label")
if err != nil {
    // Fallback to basic ASCII validation
    isValid = isBasicASCII(label)
}
```

### 4. Use Appropriate Method
```go
// Untuk single validation
isValid := table.IsValidLabel("label")

// Untuk multiple language support
isValid, languages, err := manager.ValidateLabel("tld", "label")

// Untuk specific language
isValid, err := manager.ValidateLabelForLanguage("tld", "arabic", "label")
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

## Testing Results

### Test Coverage
```bash
$ go test ./v3 -v
=== RUN   TestIdnLabelValidator_FindValidIdnTableForTld
--- PASS: TestIdnLabelValidator_FindValidIdnTableForTld (0.00s)
=== RUN   TestIdnLabelValidator_GetIdnTablesForTld  
--- PASS: TestIdnLabelValidator_GetIdnTablesForTld (0.00s)
=== RUN   TestJapaneseLanguageValidator_IsValidLabelForLanguage
--- PASS: TestJapaneseLanguageValidator_IsValidLabelForLanguage (0.00s)
=== RUN   TestParseIdnTableType
--- PASS: TestParseIdnTableType (0.00s)
PASS
ok      github.com/al0fayz/idn/v3       0.469s
```

### Example Execution
```bash
$ go run examples/v3/simple_usage.go
=== Simple IDN Usage Example ===
--- Using IDN Loader Directly ---
✅ Arabic table loaded successfully
Label 'مثال': true
--- Using IDN Manager ---
✅ 'café' is valid (supported by: [latin])
--- Cache Statistics ---
Cache stats: map[cached_languages:1 languages:[arabic]]
Available languages: [arabic armenian bengali chinese cyrillic ...]
```

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
func (manager *IdnManager) RegisterTld(tldName string, languages []string) error
func (manager *IdnManager) RegisterTldWithPreload(tldName string, languages []string) error
func (manager *IdnManager) ValidateLabel(tldName, label string) (bool, []string, error)
func (manager *IdnManager) ValidateLabelForLanguage(tldName, language, label string) (bool, error)
func (manager *IdnManager) AddLanguageToTld(tldName, language string) error
func (manager *IdnManager) RemoveLanguageFromTld(tldName, language string) error
```

## Migration from V2

### Key Changes
1. **File Location**: IDN tables now read from `/idn` folder instead of embedded files
2. **Multiple Languages**: Support for multiple IDN tables per TLD
3. **Caching**: Built-in memory caching system
4. **Management**: New IDN Manager for complex configurations

### Migration Steps
```go
// Old V2 code
validator := v2.NewIdnValidator()
isValid := validator.ValidateLabel("example", "label")

// New V3 code
manager := v3.NewIdnManager("/path/to/idn")
manager.RegisterTld("example", []string{"latin"})
isValid, _, err := manager.ValidateLabel("example", "label")
```

## Supported IDN Languages

Based on `/idn` folder content:
- Arabic, Armenian, Bengali, Chinese, Cyrillic
- Devanagari, Ethiopic, Georgian, Greek, Gurmukhi
- Hebrew, Japanese, Kannada, Khmer, Korean
- Latin, Malayalam, Myanmar, Oriya, Sinhala
- Tamil, Telugu, Thai, Tibetan, Traditional-Chinese


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
