# Perbandingan v3 (tabel IDN `.txt`) dan v4 (LGR XML)

Di repo ini, **v3** dan **v4** adalah nama package menurut **format data
tabel IDN**, bukan versi protokol IDNA:

| Package | Format tabel | Sumber di repo |
|---|---|---|
| `v3` | daftar code point `.txt` (format tabel IDN Nomulus/IANA lama), plus kolom varian RFC 3743 | `idn/*.txt` |
| `v4` | Label Generation Ruleset XML, RFC 7940 | `lgrs/*/*.xml` |

Keduanya memakai lapisan **IDNA2008** yang sama (di `v3/idna2008.go`) dan
dipasang ke `v3.IdnManager` yang sama, sehingga `CheckDomain`, `VariantKey`,
kesalahan bertipe, dan referensi RDE berlaku untuk keduanya.

---

## 1. Standar yang terlibat

| Standar | Isi | Dipakai di |
|---|---|---|
| IDNA2003 — RFC 3490–3492 | protokol IDN pertama (Nameprep, Punycode) | tidak dipakai; digantikan IDNA2008 |
| **IDNA2008** — RFC 5890–5894 (+ RFC 5895, pemetaan informatif) | properti code point (PVALID, CONTEXTJ, CONTEXTO, DISALLOWED), aturan bidi, tanda hubung | v3 dan v4 (`prepareLabel`) |
| RFC 3743 (JET) | tabel dan varian aksara CJK: varian *preferred* dan varian karakter | v3, kolom 2–3 `Chinese-IDN.txt` |
| **RFC 7940** | format XML LGR: repertoire, varian, aturan konteks/WLE, aksi | v4 |
| RFC 8228 | panduan merancang LGR dengan varian (simetri, transitivitas) | dirujuk oleh LGR di `lgrs/` |
| UTS #46 | pemetaan input pengguna (huruf kecil, lebar penuh, titik ideografik) | `IdnUtils.NormalizeDomainName` (registrar) |

**Pedoman ICANN** (kebijakan, bukan protokol) — keduanya ada di `doc/`:

- *Guidelines for the Implementation of IDNs* **versi 3.0** (2 September 2011,
  `idn-guidelines-02sep11-en.pdf`): menyesuaikan pedoman dengan IDNA2008.
- *IDN Implementation Guidelines* **versi 4.1** (22 September 2022,
  `idn-guidelines-22sep22-en.pdf`): menggantikan 3.0. Guideline 6(a) — tabel
  IDN wajib berformat LGR RFC 7940 — serta guideline 11, 12, 13 dan 18
  (varian) **ditangguhkan** oleh ICANN Board; 6(b) tetap *mendorong* registry
  yang tabelnya sudah ada di IANA untuk beralih ke format LGR, dan 6(c)
  meminta tabel memuat repertoire lengkap, varian, dan aturan konteks.

Jadi penamaan v3/v4 kebetulan selaras dengan era pedoman: tabel `.txt` adalah
format era pedoman 3.0, LGR XML adalah format yang diarahkan pedoman 4.x.

---

## 2. Struktur file

### v3 — `.txt`

Cuplikan `idn/Latin-IDN.txt`:

```
# Registry: Charleston Road Registry Inc.
# Script: Latn
# Version: 2.0
# Effective Date: 2023-04-04
# URL: https://www.iana.org/domains/idn-tables/tables/google_latn_2.0.txt
# Policy: https://www.registry.google/about/policies/domainabuse/
U+002D                         # HYPHEN-MINUS
U+0030                         # DIGIT ZERO
```

Format yang dikenali parser (`NewIdnTable`):

- `U+XXXX` per baris, komentar `#`;
- rentang `U+XXXX..U+YYYY`;
- format RFC 3743: `U+XXXX(ref);<varian preferred>;<varian karakter>,…`
  (Chinese, Traditional-Chinese) — kolom 2–3 menjadi himpunan varian;
- header `# Script:`, `# Version:`, `# Effective Date:`, `# URL:`, `# Policy:`.

Aturan konteks **tidak** ada di file: aturan per aksara ditulis tangan di Go
(`JapaneseLanguageValidator` untuk tabel `ja`, `ArabicLanguageValidator`,
`BalineseLanguageValidator`) dan dipasang menurut header `# Script:`.

### v4 — LGR XML

Cuplikan `lgrs/second-level-reference/und-Bali.xml` (disingkat):

```xml
<lgr xmlns="urn:ietf:params:xml:ns:lgr-1.0">
  <meta>
    <language>und-Bali</language>
    <unicode-version>14.0.0</unicode-version>
  </meta>
  <data>
    <char cp="1B13" tag="consonant sc:Bali">
      <var cp="1B14" type="blocked" />
    </char>
    <char cp="1B44" not-when="disallow-for-1B44" tag="sc:Bali sign" />
  </data>
  <rules>
    <class name="c" from-tag="consonant" />
    <rule name="disallow-for-1B44">
      <choice>
        <rule>
          <look-behind>
            <choice><start /><char cp="002D" /><!-- … --></choice>
          </look-behind>
          <anchor />
        </rule>
        <!-- … -->
      </choice>
    </rule>
    <action disp="blocked" any-variant="blocked" />
    <action disp="valid" />
  </rules>
</lgr>
```

Varian ditulis sebagai `<var>` di dalam `<char>`; konteks lewat atribut
`when`/`not-when` yang merujuk `<rule>`; `count` berformat `n`, `n+`, atau
`n:m`; disposisi ditentukan `<action>` secara berurutan.

---

## 3. Mekanisme validasi

Keduanya lewat `IdnManager.CheckDomain`; yang berbeda hanya langkah 4.

| Langkah | v3 (`.txt`) | v4 (LGR) |
|---|---|---|
| 1. Struktur label | panjang A-label, tanda hubung (per code point) | sama |
| 2. A-label | didekode, divalidasi, harus kanonik | sama |
| 3. IDNA2008 | NFC (input non-NFC ditolak, tidak dinormalisasi), PVALID, CONTEXTO/J, bidi | sama |
| 4. Tabel | code point ada di tabel + validator bahasa Go | repertoire (termasuk sekuens) + konteks `when`/`not-when` + aksi WLE |
| 5. Varian | RFC 3743: preferred → `allocatable`, lainnya → `blocked` | `<var>` + aksi → `blocked`/`allocatable`/`activated` |
| 6. Kunci varian | index label (RFC 7940 §8.5) | sama |
| Hasil | `*DomainResult` atau `*ValidationError` | sama; label asli diterima hanya bila disposisinya `valid` |

---

## 4. Model data

| | v3 `*IdnTable` | v4 `*LGR` |
|---|---|---|
| Repertoire | `map[int]bool` code point tunggal | elemen code point atau sekuens, diindeks per code point pertama |
| Varian | union-find atas kolom RFC 3743 | union-find atas semua `<var>` |
| Konteks | `LanguageValidator` (Go) | aturan RFC 7940 dikompilasi ke *backtracking matcher* |
| Properti Unicode | — | `gc:*` (tabel Go), `jt:R/D/L` (dibangkitkan dari `x/text`) |
| Metadata | header `.txt` | `<meta>`: versi, tanggal, bahasa, `unicode-version` |

---

## 5. Performa

Diukur di Apple M1 Pro, Go 1.26.1 (lihat `v3/README.md` dan `v4/README.md`
untuk cara mengulang):

| | v3 | v4 |
|---|---|---|
| Validasi satu nama | `CheckDomain` 0,6–2,4 µs (termasuk IDNA2008) | `CheckLabel` 1–8 µs (hanya tabel) |
| Memuat tabel | Latin 49 µs, Chinese 31 ms | und-Bali 0,6 ms, zh 153 ms |

Kedua format dimuat sekali saat start, jadi perbedaan waktu muat tidak
memengaruhi request.

---

## 6. Kapan memakai yang mana

| Kebutuhan | Pilihan |
|---|---|
| Latin/ASCII dan tabel yang sudah terdaftar di IANA dalam format `.txt` | v3 |
| Aksara dengan aturan susunan (Bali, Jawa, Devanagari, Thai, …) | v4 — aturan berasal dari file, bukan kode Go |
| Varian dengan disposisi selain blocked/allocatable, atau sekuens multi-code-point | v4 |
| Menyerahkan tabel ke IANA sesuai arah pedoman ICANN 4.x | v4 (LGR) |

Untuk `.id`: `latin` (`.txt`) bersama LGR `und-Bali`. `idn/und-Bali.txt`
hanya berisi repertoire (catatan di header-nya: varian, konteks, dan aturan
belum ada), sedangkan `und-Bali.xml` membawa varian dan aturannya. Aksara
Jawa tersedia sebagai LGR `lgrs/second-level-reference/und-Java.xml`
(rilis ICANN 13 Agustus 2026); tidak ada tabel `.txt`-nya.

Satu zona boleh mencampur tabel v3 dan v4, asalkan himpunan varian untuk
code point bersama seragam — periksa dengan `IdnManager.CheckVariantCoherence`
(contoh: `chinese` `.txt` dan LGR `zh` berbeda untuk 46 aksara, jadi jangan
dipasang bersama).
