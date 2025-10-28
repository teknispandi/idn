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
flowchart TD
    %% === Application Initialization ===
    A[Application Start] --> B["NewIdnManager(idnPath)"]
    B --> C["NewIdnLoader(idnPath)"]
    C --> D["Initialize Cache: map[string]*IdnTable"]

    %% === TLD Registration ===
    D --> E["RegisterTld(tldName, languages)"]
    E --> F[Create TldIdnConfig]
    F --> G["LoadMultipleTables(languages)"]

    %% === Load Tables per Language ===
    G --> H{For each language}
    H --> I["LoadTable(language)"]

    %% === Cache Lookup with RWMutex ===
    I --> J{"Check Cache (RLock)"}
    J -->|Hit| K[Return Cached Table]
    J -->|Miss| L["Acquire WLock (Write Lock)"]

    %% === Double-Check Locking Pattern ===
    L --> M{Double-check Cache}
    M -->|Hit| K
    M -->|Miss| N["findIdnFile(language)"]

    %% === Load from IDN Folder ===
    N --> O["Load File from /idn/<br/>Language-IDN.txt"]
    O --> P["Parse Unicode Lines<br/>U+XXXX → int codepoint"]
    P --> Q["Create NewIdnTable<br/>with validCodepoints map"]
    Q --> R["Cache Table in Memory<br/>(Protected by WLock)"]
    R --> S[Return Table]

    %% === Store in TLD Manager ===
    S --> T["Store in tldTables:<br/>TLD → Language → Table"]

    %% === Separate Label Validation Flow ===
    U["🔍 ValidateLabel(tldName, label)"] --> V["GetTldTables(tldName)"]
    V --> W{"For each supported<br/>language in TLD"}
    W --> X["IsValidLabel(label)"]
    
    %% === Unicode to Rune Conversion ===
    X --> Y{"For each rune in label<br/>(Unicode → Rune conversion)"}
    Y --> Z["Check if rune exists in<br/>validCodepoints map"]
    Z --> AA["Return Validation Result<br/>(bool + valid languages)"]

    %% === Additional Language Validator ===
    Z --> BB{Language Validator exists?}
    BB -->|Yes| CC["Run language-specific<br/>validation rules"]
    BB -->|No| AA
    CC --> AA

    %% === Styling untuk clarity ===
    style A fill:#e1f5fe,stroke:#0288d1,stroke-width:2px
    style D fill:#f3e5f5,stroke:#8e24aa,stroke-width:2px
    style J fill:#fff3e0,stroke:#ff9800,stroke-width:2px
    style R fill:#e8f5e9,stroke:#4caf50,stroke-width:2px
    style U fill:#ffebee,stroke:#f44336,stroke-width:2px
    style Y fill:#fff9c4,stroke:#fbc02d,stroke-width:2px
    style Z fill:#fce4ec,stroke:#e91e63,stroke-width:2px
    
    %% === Connection dari Registration ke Validation ===
    T -.-> U
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

## Diagram Struktur Data dan Interaksi

```mermaid
flowchart LR
    %% === Data Structure Overview ===
    subgraph "IdnManager Structure"
        IDM["🏗️ IdnManager"]
        IDM --> LOADER["loader: *IdnLoader"]
        IDM --> CONFIGS["tldConfigs: map[string]*TldIdnConfig"]
        IDM --> TABLES["tldTables: map[string]map[string]*IdnTable"]
        IDM --> MUTEX1["mutex: sync.RWMutex"]
    end

    subgraph "IdnLoader Structure" 
        LOADER --> PATH["idnPath: string"]
        LOADER --> CACHE["cache: map[string]*IdnTable"]
        LOADER --> MUTEX2["mutex: sync.RWMutex"]
    end

    subgraph "IdnTable Structure"
        TABLE["🗂️ IdnTable"]
        TABLE --> NAME["name: string"]
        TABLE --> CODEPOINTS["validCodepoints: map[int]bool"]
        TABLE --> VALIDATOR["languageValidator: LanguageValidator"]
        TABLE --> URL["url: *url.URL"]
        TABLE --> POLICY["policy: *url.URL"]
    end

    %% === File System Interaction ===
    subgraph "File System (/idn folder)"
        FILES["📁 /idn/"]
        FILES --> AR["Arabic-IDN.txt"]
        FILES --> CH["Chinese-IDN.txt"]
        FILES --> JP["Japanese-IDN.txt"]
        FILES --> MORE["... (26+ languages)"]
    end

    %% === Connections ===
    CACHE -.-> TABLE
    TABLES -.-> TABLE
    PATH -.-> FILES
    
    %% === Concurrency Flow ===
    subgraph "🔒 Concurrency Control"
        RW1["Reader threads<br/>(Multiple)"]
        RW2["Writer thread<br/>(Single)"]
        RW1 --> MUTEX1
        RW2 --> MUTEX1
        RW1 --> MUTEX2
        RW2 --> MUTEX2
    end

    style IDM fill:#e3f2fd,stroke:#1976d2,stroke-width:3px
    style LOADER fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px  
    style TABLE fill:#e8f5e9,stroke:#388e3c,stroke-width:2px
    style FILES fill:#fff3e0,stroke:#f57c00,stroke-width:2px
```

## Diagram Sequence untuk Validation Process

```mermaid
sequenceDiagram
    participant App as Application
    participant Mgr as IdnManager
    participant Ldr as IdnLoader
    participant FS as File System
    participant Tbl as IdnTable

    Note over App,Tbl: 🚀 Initialization Phase
    App->>Mgr: NewIdnManager(idnPath)
    Mgr->>Ldr: NewIdnLoader(idnPath)
    
    Note over App,Tbl: 📝 TLD Registration Phase
    App->>Mgr: RegisterTld("asia", ["chinese", "japanese"])
    Mgr->>Ldr: LoadMultipleTables(languages)
    
    loop For each language
        Ldr->>Ldr: RLock() - Check cache
        alt Cache Hit
            Ldr-->>Mgr: Return cached table
        else Cache Miss
            Ldr->>Ldr: Lock() - Write lock
            Ldr->>Ldr: Double-check cache
            alt Still not in cache
                Ldr->>FS: findIdnFile(language)
                FS-->>Ldr: Return file path
                Ldr->>FS: loadFromFile(path)
                FS-->>Ldr: Return file content
                Ldr->>Tbl: NewIdnTable(language, data, validator)
                Tbl-->>Ldr: Return IdnTable
                Ldr->>Ldr: cache[language] = table
            end
            Ldr->>Ldr: Unlock()
            Ldr-->>Mgr: Return table
        end
    end
    
    Mgr->>Mgr: Store in tldTables[TLD][Language]
    Mgr-->>App: Registration complete

    Note over App,Tbl: 🔍 Label Validation Phase  
    App->>Mgr: ValidateLabel("asia", "测试")
    Mgr->>Mgr: GetTldTables("asia")
    
    loop For each language in TLD
        Mgr->>Tbl: IsValidLabel("测试")
        Tbl->>Tbl: for _, r := range label
        Note over Tbl: Unicode string → rune conversion
        Tbl->>Tbl: Check validCodepoints[int(rune)]
        alt Has language validator
            Tbl->>Tbl: languageValidator.IsValidLabelForLanguage(label)
        end
        Tbl-->>Mgr: Return validation result
    end
    
    Mgr-->>App: Return (isValid, validLanguages, error)
```

Flow ini sudah mengikuti best practices untuk concurrent access, memory efficiency, dan performance optimization dengan:

- **Double-check locking pattern** untuk menghindari race condition
- **RWMutex** untuk memisahkan read dan write operations
- **Lazy loading** untuk efisiensi memory
- **Unicode to Rune conversion** yang otomatis di Go
- **Multi-level caching** (IdnLoader cache + IdnManager tables)
