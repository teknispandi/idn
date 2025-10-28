**versi pedoman teknis IDN dari ICANN / IETF (bukan versi file di IANA)**, yaitu perbedaan antara:

* **IDN versi 3 (IDNA2008)**
* **IDN versi 4 (LGR-based / Root Zone LGR framework)**

Keduanya memang sering disebut “v3” dan “v4” di dokumen dan komunitas IDN karena mewakili **dua generasi besar sistem validasi IDN**.
Mari kita bahas dengan presisi teknis 👇

---

## 🧩 1. Evolusi singkat sistem IDN

| Era       | Nama Teknis                                                             | Tahun           | Ciri Utama                                                                                            | Dipakai di mana                                        |
| --------- | ----------------------------------------------------------------------- | --------------- | ----------------------------------------------------------------------------------------------------- | ------------------------------------------------------ |
| **v1–v2** | IDNA2003 (RFC 3490–3492)                                                | 2003            | Berbasis Unicode 3.2, banyak anomali casing & mapping                                                 | Banyak registry lama (.com, .jp, dll)                  |
| **v3**    | **IDNA2008** (RFC 5890–5895)                                            | 2008            | Berbasis Unicode 5.x+, memperkenalkan *ContextJ*, *ContextO*, *PVALID*, *DISALLOWED*                  | Dasar semua sistem modern, termasuk **Nomulus**        |
| **v4**    | **ICANN Root Zone LGR framework** (RFC 7940 + RFC 7941 + RZ-LGR series) | 2015 → sekarang | Memperkenalkan **Label Generation Rules (LGR XML)**, varian antar-skrip, dan *Whole Label Evaluation* | Digunakan oleh ICANN Root Zone & IDN ccTLD gTLD modern |

---

## ⚙️ 2. Apa itu “IDN v3 (IDNA 2008)”

Ini adalah **kerangka teknis IETF** yang menetapkan aturan global untuk domain berbasis Unicode.

📘 **Referensi utama:**

* RFC 5890 — Definitions and Document Framework
* RFC 5891 — Registration and Processing
* RFC 5892 — Code Points
* RFC 5893 — Bidirectional Scripts (Arabic/Hebrew)
* RFC 5894 — Rationale and Background
* RFC 5895 — Mapping

🧠 **Konsep kunci IDNA 2008:**

* Menetapkan kategori karakter:

  * `PVALID` (boleh digunakan)
  * `CONTEXTJ` / `CONTEXTO` (boleh dengan konteks tertentu)
  * `DISALLOWED`
* Tidak lagi bergantung pada Unicode 3.2 (boleh upgrade)
* Tidak mendukung *mapping heuristik* seperti di IDNA2003 (contoh: full-width Latin → Latin biasa tidak otomatis)
* Tidak mengatur bahasa/spesifik skrip — hanya per-karakter secara global
* Tidak mengenal *variant mapping*

🧩 **Dipakai di Nomulus:**
Nomulus mengikuti **IDNA 2008** melalui library Java bawaan (Java 11+ menggunakan algoritma IDNA 2008), lalu menambahkan **IDN table TXT** untuk membatasi karakter sesuai skrip.

---

## 🧠 3. Apa itu “IDN v4 (LGR Framework / Root Zone LGR)”

Ini adalah evolusi di bawah koordinasi **ICANN IDN Variant Program**, bukan sekadar RFC IETF.

📘 **Dokumen dasar:**

* RFC 7940 — *Representing Label Generation Rulesets (LGRs) in XML*
* RFC 7941 — *LGR XML Format Registration*
* *Root Zone LGR-X (Version X)* — definisi LGR per skrip yang disetujui oleh ICANN Root Zone Project

🧠 **Konsep kunci LGR v4:**

* Menggunakan file **XML berbasis aturan (ruleset)**, bukan daftar code point `.txt`
* Mendefinisikan:

  * Karakter valid (`<char>` dan `<range>`)
  * **Variant sets** (untuk mengelola bentuk alternatif seperti bentuk murda di Jawa)
  * **Whole Label Evaluation (WLE)** rules → mis. “repha hanya boleh di awal”, “konsonan tidak boleh dua kali tanpa pasangan”
* Dapat memiliki **cross-script variants** (contoh: Cyrillic а vs Latin a)
* Menjadi dasar untuk sistem **Root Zone** (TLD level) dan **Second Level LGR** (registry seperti .id)

🧩 **ICANN menyebut ini kadang “IDN Guideline v4”** karena menjadi pedoman resmi penerapan IDN bagi registry sejak 2018 → 2024 (revisi terakhir).

---

## 🧮 4. Ringkasan perbedaan teknis

| Aspek                    | IDN v3 / IDNA 2008                   | IDN v4 / LGR Framework               |
| ------------------------ | ------------------------------------ | ------------------------------------ |
| Standar dasar            | RFC 5890 – 5895                      | RFC 7940 – 7941                      |
| Basis data               | Unicode Property Table               | LGR XML per skrip                    |
| Unit validasi            | Per karakter                         | Per label (Whole Label Eval)         |
| Varian antar karakter    | ❌ Tidak ada                          | ✅ Ada (`<var>`)                      |
| Konteks skrip/bahasa     | Umum (Latin/Arabic/Bidi)             | Spesifik (mis. Jawa, Thai, Tamil)    |
| Implementasi di registry | IDN tables (.txt)                    | LGR (.xml)                           |
| Contoh di Nomulus        | `Javanese-IDN.txt`                   | (Belum native, perlu adapter XML)    |
| Validasi ekstensi        | Library Java / ICU                   | LGR Evaluator (ICANN tool)           |
| Cocok untuk              | Registry lama / kompatibilitas cepat | Registry baru / multi-skrip kompleks |

---

## 🧩 5. Bagaimana kaitannya dengan **Nomulus**

* Nomulus saat ini masih berada di **arus IDNA 2008 (v3)** — artinya:

  * Validasi Unicode berbasis kategori PVALID, dll.
  * Pembatasan melalui *tabel IDN per bahasa* (TXT)
  * Tidak memahami varian atau aturan LGR XML secara native
* Untuk mendukung **LGR v4**, diperlukan:

  1. **Parser LGR XML** (bisa di luar Nomulus)
  2. Konversi hasilnya ke struktur `IdnTable` di cache
  3. Integrasi dengan pipeline validasi label (yang kamu sudah petakan di flowchart)

---

## ✅ 6. Kesimpulan praktis

| IDN versi              | Digunakan oleh                          | Ciri                                                       | Implementasi di Nomulus                    |
| ---------------------- | --------------------------------------- | ---------------------------------------------------------- | ------------------------------------------ |
| **v3 (IDNA 2008)**     | Semua registry modern, termasuk Nomulus | Validasi karakter berbasis kategori Unicode (tanpa varian) | Native                                     |
| **v4 (LGR Framework)** | ICANN Root Zone, TLD IDN terbaru        | Validasi label lengkap, variant sets, WLE                  | Belum native, bisa diintegrasi via adapter |

---

Ah, jelas sekali sekarang 👍 —
yang kamu maksud adalah **versi pedoman teknis IDN dari ICANN / IETF (bukan versi file di IANA)**, yaitu perbedaan antara:

* **IDN versi 3 (IDNA2008)**
* **IDN versi 4 (LGR-based / Root Zone LGR framework)**

Keduanya memang sering disebut “v3” dan “v4” di dokumen dan komunitas IDN karena mewakili **dua generasi besar sistem validasi IDN**.
Mari kita bahas dengan presisi teknis 👇

---

## 🧩 1. Evolusi singkat sistem IDN

| Era       | Nama Teknis                                                             | Tahun           | Ciri Utama                                                                                            | Dipakai di mana                                        |
| --------- | ----------------------------------------------------------------------- | --------------- | ----------------------------------------------------------------------------------------------------- | ------------------------------------------------------ |
| **v1–v2** | IDNA2003 (RFC 3490–3492)                                                | 2003            | Berbasis Unicode 3.2, banyak anomali casing & mapping                                                 | Banyak registry lama (.com, .jp, dll)                  |
| **v3**    | **IDNA2008** (RFC 5890–5895)                                            | 2008            | Berbasis Unicode 5.x+, memperkenalkan *ContextJ*, *ContextO*, *PVALID*, *DISALLOWED*                  | Dasar semua sistem modern, termasuk **Nomulus**        |
| **v4**    | **ICANN Root Zone LGR framework** (RFC 7940 + RFC 7941 + RZ-LGR series) | 2015 → sekarang | Memperkenalkan **Label Generation Rules (LGR XML)**, varian antar-skrip, dan *Whole Label Evaluation* | Digunakan oleh ICANN Root Zone & IDN ccTLD gTLD modern |

---

## ⚙️ 2. Apa itu “IDN v3 (IDNA 2008)”

Ini adalah **kerangka teknis IETF** yang menetapkan aturan global untuk domain berbasis Unicode.

📘 **Referensi utama:**

* RFC 5890 — Definitions and Document Framework
* RFC 5891 — Registration and Processing
* RFC 5892 — Code Points
* RFC 5893 — Bidirectional Scripts (Arabic/Hebrew)
* RFC 5894 — Rationale and Background
* RFC 5895 — Mapping

🧠 **Konsep kunci IDNA 2008:**

* Menetapkan kategori karakter:

  * `PVALID` (boleh digunakan)
  * `CONTEXTJ` / `CONTEXTO` (boleh dengan konteks tertentu)
  * `DISALLOWED`
* Tidak lagi bergantung pada Unicode 3.2 (boleh upgrade)
* Tidak mendukung *mapping heuristik* seperti di IDNA2003 (contoh: full-width Latin → Latin biasa tidak otomatis)
* Tidak mengatur bahasa/spesifik skrip — hanya per-karakter secara global
* Tidak mengenal *variant mapping*

🧩 **Dipakai di Nomulus:**
Nomulus mengikuti **IDNA 2008** melalui library Java bawaan (Java 11+ menggunakan algoritma IDNA 2008), lalu menambahkan **IDN table TXT** untuk membatasi karakter sesuai skrip.

---

## 🧠 3. Apa itu “IDN v4 (LGR Framework / Root Zone LGR)”

Ini adalah evolusi di bawah koordinasi **ICANN IDN Variant Program**, bukan sekadar RFC IETF.

📘 **Dokumen dasar:**

* RFC 7940 — *Representing Label Generation Rulesets (LGRs) in XML*
* RFC 7941 — *LGR XML Format Registration*
* *Root Zone LGR-X (Version X)* — definisi LGR per skrip yang disetujui oleh ICANN Root Zone Project

🧠 **Konsep kunci LGR v4:**

* Menggunakan file **XML berbasis aturan (ruleset)**, bukan daftar code point `.txt`
* Mendefinisikan:

  * Karakter valid (`<char>` dan `<range>`)
  * **Variant sets** (untuk mengelola bentuk alternatif seperti bentuk murda di Jawa)
  * **Whole Label Evaluation (WLE)** rules → mis. “repha hanya boleh di awal”, “konsonan tidak boleh dua kali tanpa pasangan”
* Dapat memiliki **cross-script variants** (contoh: Cyrillic а vs Latin a)
* Menjadi dasar untuk sistem **Root Zone** (TLD level) dan **Second Level LGR** (registry seperti .id)

🧩 **ICANN menyebut ini kadang “IDN Guideline v4”** karena menjadi pedoman resmi penerapan IDN bagi registry sejak 2018 → 2024 (revisi terakhir).

---

## 🧮 4. Ringkasan perbedaan teknis

| Aspek                    | IDN v3 / IDNA 2008                   | IDN v4 / LGR Framework               |
| ------------------------ | ------------------------------------ | ------------------------------------ |
| Standar dasar            | RFC 5890 – 5895                      | RFC 7940 – 7941                      |
| Basis data               | Unicode Property Table               | LGR XML per skrip                    |
| Unit validasi            | Per karakter                         | Per label (Whole Label Eval)         |
| Varian antar karakter    | ❌ Tidak ada                          | ✅ Ada (`<var>`)                      |
| Konteks skrip/bahasa     | Umum (Latin/Arabic/Bidi)             | Spesifik (mis. Jawa, Thai, Tamil)    |
| Implementasi di registry | IDN tables (.txt)                    | LGR (.xml)                           |
| Contoh di Nomulus        | `Javanese-IDN.txt`                   | (Belum native, perlu adapter XML)    |
| Validasi ekstensi        | Library Java / ICU                   | LGR Evaluator (ICANN tool)           |
| Cocok untuk              | Registry lama / kompatibilitas cepat | Registry baru / multi-skrip kompleks |

---

## 🧩 5. Bagaimana kaitannya dengan **Nomulus**

* Nomulus saat ini masih berada di **arus IDNA 2008 (v3)** — artinya:

  * Validasi Unicode berbasis kategori PVALID, dll.
  * Pembatasan melalui *tabel IDN per bahasa* (TXT)
  * Tidak memahami varian atau aturan LGR XML secara native
* Untuk mendukung **LGR v4**, diperlukan:

  1. **Parser LGR XML** (bisa di luar Nomulus)
  2. Konversi hasilnya ke struktur `IdnTable` di cache
  3. Integrasi dengan pipeline validasi label (yang kamu sudah petakan di flowchart)

---

## ✅ 6. Kesimpulan praktis

| IDN versi              | Digunakan oleh                          | Ciri                                                       | Implementasi di Nomulus                    |
| ---------------------- | --------------------------------------- | ---------------------------------------------------------- | ------------------------------------------ |
| **v3 (IDNA 2008)**     | Semua registry modern, termasuk Nomulus | Validasi karakter berbasis kategori Unicode (tanpa varian) | Native                                     |
| **v4 (LGR Framework)** | ICANN Root Zone, TLD IDN terbaru        | Validasi label lengkap, variant sets, WLE                  | Belum native, bisa diintegrasi via adapter |

---

# 📘 **Perbandingan antara IDN versi 3 (IDNA2008 + IDN Table .txt)** dan **IDN versi 4 (LGR XML Framework)**

## 🧩 1. Konsep Dasar

| Aspek                | **IDN v3 (IDNA2008)**                          | **IDN v4 (LGR Framework / RFC 7940)**                                   |
| -------------------- | ---------------------------------------------- | ----------------------------------------------------------------------- |
| Definisi utama       | RFC 5890–5895 (IETF)                           | RFC 7940–7941 (ICANN/IETF)                                              |
| Unit validasi        | Per **karakter**                               | Per **label (string domain)**                                           |
| Format data          | File `.txt` (IDN Table)                        | File `.xml` (LGR)                                                       |
| Tujuan utama         | Memastikan karakter valid & tidak campur skrip | Memastikan **karakter valid + konteks + varian + aturan label lengkap** |
| Tingkat kompleksitas | Sederhana, cepat                               | Kompleks, mendukung aturan spesifik bahasa                              |
| Cocok untuk          | Registry umum (Latin, Cyrillic, Arabic)        | Registry multi-skrip (Jawa, Thai, Devanagari, dll.)                     |

---

## 🧠 2. Struktur File

### 🧾 IDN v3 — Format `.txt`

Contoh: `Javanese-IDN.txt`

```
# IDN Table for Javanese
# Version: 1.0
U+A980 ; valid  # JAVANESE SIGN PANYANGGA
U+A981 ; valid  # JAVANESE SIGN CECAK
U+A982 ; valid  # JAVANESE SIGN LAYAR
U+A983 ; valid  # JAVANESE SIGN WIGNYAN
U+A984..U+A9B2 ; valid # JAVANESE LETTERS
U+A9B4 ; valid  # JAVANESE VOWEL SIGN TARUNG
```

➡️ Hanya daftar **codepoint** yang diizinkan, tanpa hubungan antar-karakter atau konteks.
🧮 Parsing-nya sederhana → cocok untuk `map[rune]bool` di Nomulus.

---

### 🧾 IDN v4 — Format `.xml` (LGR)

Contoh: `javanese-lgr.xml`

```xml
<lgr xmlns="urn:ietf:params:xml:ns:lgr-1.0">
  <meta>
    <language>Javanese</language>
    <unicode-version>15.0.0</unicode-version>
  </meta>
  <char cp="A984" tag="consonant"/>
  <char cp="A985" tag="consonant"/>
  <char cp="A9B4" tag="vowel"/>
  
  <!-- Variant definition -->
  <variant cp="A9B4" type="blocked"/>

  <!-- Whole Label Evaluation (WLE) rule -->
  <rule name="NoDoubleRepha">
    <comment>Prevent double repha in label</comment>
    <class by-ref="consonant"/>
    <match count=">1" ref="A9A0"/>
  </rule>
</lgr>
```

➡️ File XML ini menyertakan:

* `<char>` dan `<range>` → daftar karakter valid
* `<variant>` → hubungan antar karakter yang mirip (varian)
* `<rule>` → aturan konteks label (*Whole Label Evaluation*)

---

## ⚙️ 3. Mekanisme Validasi

| Tahap                 | **IDN v3 (Nomulus sekarang)** | **IDN v4 (LGR Framework)**       |
| --------------------- | ----------------------------- | -------------------------------- |
| 1️⃣ Input             | Label Unicode                 | Label Unicode                    |
| 2️⃣ Normalisasi       | NFC                           | NFC                              |
| 3️⃣ Validasi karakter | Cek `U+xxxx` di tabel `.txt`  | Cek `<char>` & `<range>` di XML  |
| 4️⃣ Cek varian        | Tidak ada                     | Gunakan `<variant>` rule         |
| 5️⃣ Cek aturan label  | Tidak ada                     | Gunakan `<rule>` / `<wle>`       |
| 6️⃣ Output            | Boolean valid/invalid         | Valid, invalid, atau varian list |

📍 IDN v3 hanya tahu “boleh/tidak boleh”.
📍 IDN v4 tahu “boleh”, “varian ini diblokir”, atau “melanggar aturan label”.

---

## 🧮 4. Model Data Internal

| Struktur          | v3 (`.txt`)    | v4 (`.xml`)                         |
| ----------------- | -------------- | ----------------------------------- |
| `validCodepoints` | `map[int]bool` | `map[int]Char{tag,variant,context}` |
| `variantSets`     | ❌              | ✅ `map[int][]Variant`               |
| `rules` (WLE)     | ❌              | ✅ `[]Rule{match,ref,count}`         |
| `languageTag`     | String         | String                              |
| `unicodeVersion`  | Tidak disimpan | Disimpan eksplisit                  |
| `parseTime`       | Mikrodetik     | Lebih lama (parse XML)              |

---

## 🧱 5. Implementasi di Registry (mis. Nomulus)

| Komponen         | v3                     | v4 (adaptasi)                    |
| ---------------- | ---------------------- | -------------------------------- |
| **File storage** | `/idn/{Lang}-IDN.txt`  | `/lgr/{Lang}.xml`                |
| **Loader**       | `IdnLoader`            | `LgrLoader`                      |
| **Cache**        | `map[string]*IdnTable` | `map[string]*LgrTable`           |
| **Validator**    | `IsValidLabel()`       | `EvaluateLabel()`                |
| **Varian check** | Manual blocklist       | XML `<variant>` rules            |
| **Context rule** | Tidak didukung         | `<rule>` / `<wle>` evaluasi      |
| **Performance**  | Sangat cepat           | Sedikit lebih lambat tapi akurat |

---

## 📊 6. Kelebihan dan Kekurangan

| Aspek                   | **v3 (TXT)**    | **v4 (LGR)**                 |
| ----------------------- | --------------- | ---------------------------- |
| Kecepatan validasi      | ⚡ Sangat cepat  | ⚙️ Sedikit lebih lambat      |
| Ketepatan linguistik    | ❌ Terbatas      | ✅ Akurat, sesuai skrip       |
| Dukungan varian         | ❌ Tidak ada     | ✅ Ada                        |
| Dukungan aturan konteks | ❌ Tidak ada     | ✅ Ada (WLE)                  |
| Kompatibilitas Nomulus  | ✅ Native        | ❌ Butuh adaptor              |
| Cocok untuk             | Latin, Cyrillic | Jawa, Arab, Devanagari, Thai |
| Format modern ICANN     | ❌               | ✅                            |

---

## 🧠 7. Kesimpulan Praktis

| Tujuan                                            | Solusi                                 |
| ------------------------------------------------- | -------------------------------------- |
| Validasi cepat untuk TLD biasa (Latin, ASCII)     | Gunakan **IDN v3 (.txt)**              |
| Validasi skrip kompleks (Jawa, Bali, Arab, Tamil) | Gunakan **IDN v4 (LGR XML)**           |
| Integrasi di Nomulus                              | Buat **adapter parser XML → IdnTable** |
| Validasi antar varian                             | Gunakan `<variant>` mapping            |
| Validasi struktur kata/label                      | Gunakan `<rule>` WLE                   |

---

