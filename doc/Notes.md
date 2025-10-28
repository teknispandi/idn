# Ref
- [IDN Repository Table ](https://www.iana.org/domains/idn-tables)
- [ICANN IDN Guideline](https://www.icann.org/resources/pages/implementation-guidelines-2012-02-25-en)


# RFC 
- [Internationalized Domain Names for Applications (IDNA)](https://datatracker.ietf.org/doc/html/rfc5890)
- [Internationalized Domain Names in Applications (IDNA): Protocol](https://datatracker.ietf.org/doc/html/rfc5891)
- [The Unicode Code Points and Internationalized Domain Names for Applications (IDNA)](https://datatracker.ietf.org/doc/html/rfc5892)
- [Right-to-Left Scripts for Internationalized Domain Names for Applications (IDNA)](https://datatracker.ietf.org/doc/html/rfc5893)

# Flow V3

### 1. **Load IDN Format dari Folder /idn**
```
📁 /idn/
├── Arabic-IDN.txt
├── Chinese-IDN.txt  
├── Japanese-IDN.txt
├── Hebrew-IDN.txt
└── ... (26+ bahasa lainnya)
```

**Implementasi di idn_loader.go:**
- `NewIdnLoader(idnPath string)` - Inisialisasi loader dengan path ke folder /idn
- `findIdnFile(language string)` - Mencari file IDN dengan pola nama: `{Language}-IDN.txt`
- `loadFromFile(filename string)` - Membaca file IDN line by line

### 2. **Register TLD ke idn_manager dengan Bahasa yang di-Support (untuk Caching)**
```go
// Di idn_manager.go
manager := NewIdnManager("/path/to/idn")
manager.RegisterTld("asia", []string{"chinese", "japanese", "korean"})
```

**Flow:**
- `IdnManager` menyimpan mapping: `TLD -> [Bahasa1, Bahasa2, ...]`
- Setiap TLD punya `TldIdnConfig` yang berisi daftar bahasa yang didukung
- Tables disimpan dalam struktur: `map[string]map[string]*IdnTable` (TLD -> Language -> Table)

### 3. **Caching Bahasa dengan Membaca File menggunakan sync.RWMutex**

**Implementasi Double-Check Locking Pattern di idn_loader.go:**
```go
func (loader *IdnLoader) LoadTable(language string) (*IdnTable, error) {
    // 1. Read Lock - cek cache
    loader.mutex.RLock()
    if table, exists := loader.cache[language]; exists {
        loader.mutex.RUnlock()
        return table, nil
    }
    loader.mutex.RUnlock()

    // 2. Write Lock - load dan cache
    loader.mutex.Lock()
    defer loader.mutex.Unlock()

    // 3. Double-check locking
    if table, exists := loader.cache[language]; exists {
        return table, nil
    }

    // 4. Load dari file dan cache
    filename, err := loader.findIdnFile(language)
    data, err := loader.loadFromFile(filename)
    table, err := NewIdnTable(language, data, validator)
    
    loader.cache[language] = table
    return table, nil
}
```

### 4. **Masukan Format Unicode ke Rune**

**Implementasi di idn.go:**
```go
func (t *IdnTable) IsValidLabel(label string) bool {
    // Iterasi melalui setiap codepoint dalam label
    for _, r := range label {  // <- Unicode string di-convert ke rune
        if !t.validCodepoints[int(r)] {
            return false
        }
    }
    return true
}
```

**Parsing Unicode dari File IDN:**
```go
func readCodepoint(line string, pattern *regexp.Regexp) (int, error) {
    // Parse format: U+XXXX menjadi int codepoint
    matches := pattern.FindStringSubmatch(line)
    hexString := matches[1]
    codepoint, err := strconv.ParseInt(hexString, 16, 32)
    return int(codepoint), nil
}
```

## Diagram Flow IDN v3

```mermaid
graph TD
    A[Application Start] --> B[NewIdnManager idnPath]
    B --> C[NewIdnLoader idnPath]
    C --> D[Initialize Cache map string *IdnTable]
    
    D --> E[RegisterTld tldName, languages]
    E --> F[Create TldIdnConfig]
    F --> G[LoadMultipleTables languages]
    
    G --> H{For each language}
    H --> I[LoadTable language]
    
    I --> J{Check Cache with RLock}
    J -->|Hit| K[Return Cached Table]
    J -->|Miss| L[Acquire WLock]
    
    L --> M[Double-check Cache]
    M -->|Hit| K
    M -->|Miss| N[findIdnFile language]
    
    N --> O[Load file from /idn/{Language}-IDN.txt]
    O --> P[Parse Unicode U+XXXX to int codepoint]
    P --> Q[NewIdnTable with validCodepoints map]
    Q --> R[Cache Table in memory]
    R --> S[Return Table]
    
    S --> T[Store in tldTables TLD->Language->Table]
    
    U[ValidateLabel tldName, label] --> V[GetTldTables tldName]
    V --> W{For each supported language}
    W --> X[IsValidLabel label]
    X --> Y{For each rune in label}
    Y --> Z[Check if rune in validCodepoints]
    Z --> AA[Return validation result]
    
    style A fill:#e1f5fe
    style D fill:#f3e5f5
    style J fill:#fff3e0
    style R fill:#e8f5e8
    style Z fill:#ffebee
```

## Struktur Data Key

1. **IdnManager**
   ```go
   type IdnManager struct {
       loader     *IdnLoader                    // Loader untuk file IDN
       tldConfigs map[string]*TldIdnConfig      // TLD -> Config
       tldTables  map[string]map[string]*IdnTable // TLD -> Language -> Table  
       mutex      sync.RWMutex                  // Concurrent access protection
   }
   ```

2. **IdnLoader** 
   ```go
   type IdnLoader struct {
       idnPath string                   // Path ke folder /idn
       cache   map[string]*IdnTable     // Language -> Cached Table
       mutex   sync.RWMutex            // Cache protection
   }
   ```

3. **IdnTable**
   ```go
   type IdnTable struct {
       name              string           // Nama bahasa
       validCodepoints   map[int]bool     // Unicode codepoint -> valid
       languageValidator LanguageValidator // Validator tambahan
   }
   ```

## Key Features Flow:

1. **Lazy Loading**: Table hanya di-load saat dibutuhkan
2. **Thread-Safe Caching**: Menggunakan sync.RWMutex dengan double-check locking
3. **Unicode to Rune Conversion**: Otomatis saat validasi label
4. **Multi-TLD Support**: Satu manager bisa handle multiple TLD dengan bahasa berbeda
5. **Language Validation**: Support validator khusus per bahasa
