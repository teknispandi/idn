# Integrasi Modul IDN ke Registry (nomnom) dan Registrar (registrar-api)

Dokumen ini menjelaskan cara memakai modul `github.com/al0fayz/idn` di
**nomnom** (registry PANDI) dan **registrar-api** (registrar berbasis EPP).
Isinya rencana implementasi; kode kedua proyek itu belum diubah.

Kondisi kode yang dikutip di sini diperiksa pada November 2026. Nama fungsi
dan path file bisa bergeser — cocokkan lagi sebelum mengerjakan.

## 1. Versi modul

API yang dipakai di dokumen ini (`CheckDomain`, `VariantKey`, `RdeTableRefs`,
`CheckVariantCoherence`, package `v4`) **belum ada di tag terakhir `v1.0.2`**.
Buat tag baru dari `main` sebelum integrasi, misalnya `v1.1.0`: signature lama
tidak berubah, tetapi perilakunya diperketat (lihat
[v3/README.md](../v3/README.md#perubahan-perilaku-sejak-v102)).

```bash
go get github.com/al0fayz/idn@v1.1.0   # GOPRIVATE=github.com/al0fayz/* sudah dipakai registrar-api
```

Jangan memakai `replace => ../idn`: Dockerfile nomnom menjalankan
`go mod download` dengan build context di root repo nomnom, sehingga path di
luar repo tidak ikut ter-copy.

## 2. Konsep yang dipakai kedua sisi

```go
manager := v3.NewIdnManagerFS(idn.Tables())  // tabel disematkan di binary
manager.RegisterTldConfig(v3.TldIdnConfig{...}) // per zona: tabel, AllowASCII, TableRefs
res, err := manager.CheckDomain(name)          // U-label atau A-label
```

| Field `DomainResult` | Dipakai untuk |
|---|---|
| `ASCIIName` | disimpan di kolom `name`, dikirim di EPP `<domain:name>`, dipakai DNS |
| `UnicodeName` | tampilan ke pengguna, RDAP `unicodeName` |
| `Zone` | zona hasil kecocokan akhiran terpanjang (`co.id` sebelum `id`) |
| `TableName()` | tabel IDN yang menerima label → RDE `idnTableId`; kosong untuk nama ASCII |
| `VariantKey` | index label RFC 7940 §8.5; dua nama saling varian ⇔ kuncinya sama |

Penolakan selalu `*v3.ValidationError`; jenisnya dibedakan dengan `errors.Is`
(daftar lengkap di [v3/README.md](../v3/README.md#jenis-kesalahan)).

**Konfigurasi zona harus identik di registry dan registrar** — tabel, urutan
tabel, dan `AllowASCII`. Kalau tidak, registrar meloloskan nama yang ditolak
registry atau sebaliknya. Sumber kebenarannya ada di registry (lihat 3.1).

## 3. nomnom (registry)

### 3.0 Kondisi saat ini

Yang perlu diketahui sebelum mengganti `pkg/idn`:

1. **Container tidak membawa tabel IDN.** `idn_table_dir` default-nya path
   relatif `deploy/idn-tables`, sedangkan `deploy/docker/Dockerfile.service`
   hanya menyalin binary dan `config.yaml`. `loadIDNTables` mengembalikan
   `nil` tanpa error bila direktori tidak ada, sehingga setiap TLD memakai
   `idn.Validator{}` yang permisif. Hanya deployment systemd
   (`WorkingDirectory=/opt/nomnom`) yang memuat tabel.
2. **Pengecekan varian tidak pernah berjalan.** `DomainServer.SetIDNVariants`
   tidak dipanggil di mana pun, jadi `idnVariants` selalu kosong; endpoint
   `/admin/idn-variants/...` selalu 404. Selain itu `ExpandLabel` hanya
   mengganti satu karakter per langkah (varian `A'B'` dari `AB` tidak pernah
   dicek) dan tidak mengenali A-label.
3. **Hasil validasi dibuang.** Di `DomainServer.Create`,
   `validator.Validate(label)` mengembalikan A-label tetapi hasilnya diabaikan;
   kolom `name` berisi nama persis seperti dikirim klien (tidak di-lowercase,
   tidak di-punycode).
4. **`pkg/idn` memakai profil `idna.Lookup`**, bukan `idna.Registration`, dan
   hanya mengecek tabel untuk label berprefiks `xn--`.
5. **`CheckAvailability` tidak memvalidasi IDN** sama sekali.
6. **Kunci tabel = nama file.** `loadIDNTables` memakai nama file sebagai kunci
   TLD, jadi `latin.txt`, `arabic.txt`, dst. di `deploy/idn-tables/` tidak
   terpakai untuk TLD mana pun.
7. Tidak ada kolom IDN per domain, field IDN di proto `Domain`, ekstensi EPP
   IDN, maupun `idnTableRef` di RDE. Satu-satunya kait yang ada:
   kolom `tld.idn_table_name` (migrasi `00011_tld_policy.sql`), bisa diubah
   lewat `PUT /admin/tld/{tld}/policy`, tetapi tidak dibaca oleh alur mana pun.
8. Nama duplikat tidak dipetakan ke `AlreadyExists`: pelanggaran indeks unik
   `idx_domain_name_active` kembali sebagai `codes.Internal` → EPP 2400.

### 3.1 Konfigurasi dan boot (`services/registry-core/cmd/main.go`)

Ganti `loadIDNTables` dengan pembuatan `*v3.IdnManager`:

```go
manager := v3.NewIdnManagerFS(idn.Tables()) // tidak bergantung pada file di container

// Opsional: LGR XML (mis. Bali dengan variannya) menggantikan tabel .txt bernama sama.
bali, err := v4.Load(lgrFS, "und-Bali.xml")
if err != nil { return err }
manager.RegisterTable(bali)

for _, tld := range tlds { // dari tabel tld
    cfg := v3.TldIdnConfig{
        TldName:    tld.Name,
        Languages:  splitTables(tld.IDNTableName), // mis. "latin,und-bali"
        Enabled:    true,
        AllowASCII: true,
        TableRefs:  rdeRefs[tld.Name], // dari config, lihat 3.6
    }
    if err := manager.RegisterTldConfig(cfg); err != nil {
        return err // gagal tertutup
    }
}
if err := errors.Join(manager.CheckRdeTableRefs(), manager.CheckVariantCoherence()); err != nil {
    return err
}
```

- `tld.idn_table_name` cocok menjadi sumber daftar tabel per zona (isi dengan
  daftar dipisah koma, urutan = prioritas).
- **Gagal tertutup**: TLD tanpa konfigurasi tabel tidak boleh jatuh ke
  validator permisif. Untuk zona yang memang hanya ASCII, daftarkan dengan
  `Languages: nil, AllowASCII: true`.
- `CheckVariantCoherence` menolak kombinasi tabel yang varian bersamanya tidak
  seragam (Guideline 13 ICANN IDN Implementation Guidelines; diusulkan di
  v4.0 dan ditangguhkan di v4.1, tetapi tetap syarat teknis `VariantKey`). Contoh dari
  tabel di modul ini: `chinese` + `japanese` berbeda untuk 1.001 aksara Han.
- Ekspos konfigurasi ini ke registrar (endpoint admin read-only atau dokumen
  yang diterbitkan) agar registrar memakai konfigurasi yang sama.

### 3.2 Alur create (`internal/rpc/domain.go`, `DomainServer.Create`)

Jalankan `CheckDomain` **paling awal**, sebelum reserved dan BSA, supaya semua
pengecekan berikutnya memakai bentuk kanonik:

```go
res, err := s.idn.CheckDomain(in.Name)
if err != nil {
    return status.Error(grpcCodeFor(err), err.Error())
}
// Mulai di sini gunakan res.ASCIIName / res.ALabel / res.Zone, bukan in.Name.
```

Lalu:

1. **Reserved dan BSA**: cocokkan `res.ALabel` (dan `res.ULabel` bila daftar
   reserved/BSA berisi U-label). Untuk mencakup varian, hitung `VariantKey`
   untuk setiap entri reserved/BSA saat diimpor (per TLD) dan bandingkan
   kuncinya — tidak perlu membangkitkan varian.
2. **Bentrok varian**: cukup indeks unik `(tld_name, variant_key)` (3.4).
   Tangkap pelanggarannya sebagai `codes.AlreadyExists` (sekaligus memperbaiki
   poin 3.0-8 untuk nama yang sama persis).
3. **Varian allocatable**: bila kebijakan PANDI mengizinkan varian dialokasikan
   ke pendaftar yang sama (Guideline 11 ICANN — juga ditangguhkan di v4.1 —
   mengusulkan varian *allocatable hanya ke pendaftar yang sama* atau
   *blocked*), bentrok kunci dengan domain milik
   pendaftar yang sama dan `res.VariantLabels` menyebut nama itu
   `allocatable` boleh diteruskan. Tanpa kebijakan itu, semua bentrok ditolak.
4. Simpan `res.ASCIIName` sebagai `name`, ditambah kolom baru di 3.4.

Pemetaan kesalahan yang disarankan:

| Kesalahan | gRPC | EPP (lewat `flow/create.go`) |
|---|---|---|
| struktur/IDNA2008 (`ErrEmptyLabel`, `ErrHyphen34`, `ErrInvalidALabel`, `ErrDisallowedCodepoint`, `ErrBidiRule`, …) | `InvalidArgument` | 2005 |
| kebijakan tabel (`ErrCodepointNotInTable`, `ErrLanguageRule`, `ErrNoIdnTable`, `ErrMixedScript`), `ErrUnknownTld`, `ErrTldDisabled`, `ErrNotRegistrable` | `InvalidArgument` atau `FailedPrecondition` | 2005 / 2306 — putuskan, lalu samakan dengan registrar |
| bentrok nama atau varian | `AlreadyExists` | 2302 |

Teks `err.Error()` sudah menyebut code point, posisi, tabel, dan nama aturan
(mis. `rule disallow-for-1B44`) — teruskan sebagai `<msg>`/`<reason>`.

### 3.3 Cek ketersediaan (`DomainServer.CheckAvailability`)

Untuk setiap nama: `CheckDomain` dulu (nama tidak valid → `avail=false`,
`reason` dari kesalahan), lalu cari `(tld_name, variant_key)` yang aktif
(`reason "In use"` atau `"Variant of <nama>"`), lalu BSA.

### 3.4 Migrasi database (`services/registry-core/migrations/`)

Konvensi: goose, nomor 5 digit; file berikutnya `00031_domain_idn.sql`.

```sql
-- +goose Up
-- +goose StatementBegin
ALTER TABLE domain
    ADD COLUMN IF NOT EXISTS unicode_name TEXT,
    ADD COLUMN IF NOT EXISTS idn_table    TEXT,
    ADD COLUMN IF NOT EXISTS variant_key  TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE domain
    DROP COLUMN IF EXISTS variant_key,
    DROP COLUMN IF EXISTS idn_table,
    DROP COLUMN IF EXISTS unicode_name;
-- +goose StatementEnd
```

Isi kolom baru untuk data lama (backfill) dengan job yang menjalankan
`CheckDomain(name)` per domain aktif. **Laporkan, jangan perbaiki diam-diam**,
nama yang ditolak atau yang `name`-nya berbeda dari `ASCIIName` (huruf besar,
U-label tersimpan): data itu masuk sebelum validasi berjalan benar dan perlu
keputusan manual. Setelah backfill bersih, buat indeks unik di migrasi
terpisah (`00032_domain_variant_key_unique.sql`):

```sql
CREATE UNIQUE INDEX IF NOT EXISTS idx_domain_variant_key_active
    ON domain (tld_name, variant_key)
    WHERE deletion_time = '9999-12-31 23:59:59+00';
```

Tambahkan juga `unicode_name`, `idn_table` ke proto `registryv1.Domain` agar
RDE dan RDAP bisa memakainya.

### 3.5 EPP (`services/epp-service`)

- `flow/create.go`: teruskan nama apa adanya; normalisasi dan validasi terjadi
  di registry-core. Kembalikan `res.ASCIIName` di respons `<domain:creData>`.
- Ekstensi IDN `urn:ietf:params:xml:ns:idn-1.0` (draft-ietf-eppext-idnmap)
  belum ada di `pkg/epp/codec` dan tercatat sebagai pekerjaan P3 (E-04) di
  `docs/PRODUCTION-READINESS.md`. Bila ditambahkan: `<idn:table>` →
  `CheckDomainForTable(name, table)`, dan tolak bila `<idn:uname>` tidak sama
  dengan `res.ULabel`. Klien `epp-pandi-client` sudah punya
  `frames.NewIDNCreate(table, uname)`.
- `<domain:check>`: lihat 3.3.

### 3.6 RDE (`services/rde-service`)

Deposit belum memuat `rdeIDN`. Yang perlu ditambahkan:

- header: satu `<rdeIDN:idnTableRef id="…">` per tabel, dari
  `manager.RdeTableRefs(tld)` (`ID`, `URL`, `Policy`);
- per domain: `<domain:idnTableId>` = kolom `idn_table`, `<domain:uName>` =
  `unicode_name` (hanya untuk nama IDN);
- konfigurasi `TableRefs` (URL tabel PANDI di IANA Repository of IDN Practices
  dan URL kebijakannya) di config registry-core, bukan di file tabel —
  header `# URL:` tabel di modul menunjuk registrasi Google.

`CheckRdeTableRefs` saat boot memastikan deposit tidak pernah berisi
referensi kosong.

### 3.7 Lain-lain di nomnom

- **RDAP**: isi `UnicodeName` (field sudah ada di `pkg/rdap/types.go`) dari
  `unicode_name`.
- **`nomctl idn import-lgr`**: konversi LGR ke format teks (`pkg/idn/lgr`)
  membuang aturan WLE dan sekuens multi-code-point. Ganti dengan memuat LGR
  langsung lewat `v4.Load`; perintah ini bisa diubah menjadi validator LGR
  (`v4.Load` gagal untuk LGR yang tidak didukung).
- **`services/registrar-api` (internal nomnom)**: `createDomain` dan
  `checkDomains` meneruskan nama tanpa validasi; validasi tetap terjadi di
  registry-core setelah 3.2, tetapi normalisasi input (`NormalizeDomainName`)
  sebaiknya dilakukan di sini.
- **Metrik** `nomnom_idn_variant_rejected_total{tld,reason}`: alasan
  `variant_registered` dipancarkan dari pelanggaran indeks unik pada 3.2.
- Setelah semua pemakai pindah, hapus `pkg/idn` (`Validator`, `VariantMap`,
  `ExpandLabel`, `LoadVariants`) dan endpoint `/admin/idn-variants`; ganti
  dengan endpoint yang memanggil `res.VariantLabels`.

## 4. registrar-api (registrar EPP)

### 4.0 Kondisi saat ini

1. Tidak ada dukungan IDN: tidak ada pemakaian `idna`/punycode, satu kolom
   `name` tanpa bentuk Unicode.
2. Validator `isDomain` (`validators/Validator.go`) hanya menerima ASCII huruf
   kecil, mengizinkan `_`, dan bagian TLD-nya opsional (`"abc."` lolos).
   Karena `binding` berjalan sebelum `strings.ToLower`, input berhuruf besar
   ditolak walau valid.
3. **Endpoint reseller** (`handler/handlerUser/reseller/`: check, register,
   renew, nameservers, contact) **tidak memvalidasi format sama sekali** —
   hanya `ToLower`/`TrimSpace`. Input Unicode akan diteruskan mentah ke
   `<domain:name>`.
4. Penentuan zona tidak konsisten: handler lama memakai `dom[1]` atau
   `dom[1]+"."+dom[2]`, reseller memakai dua label terakhir (`domainZone`).
5. Harga dihitung dari `len(sld)` dalam **byte**; untuk U-label nilainya
   berbeda dari jumlah karakter maupun panjang A-label.
6. EPP lewat `github.com/al0fayz/epp-pandi-client/v2`, yang tidak
   menormalisasi nama; ekstensi IDN-nya (`frames.NewIDNCreate`) belum dipakai.

### 4.1 Validasi dan normalisasi

Buat satu manager saat start dengan konfigurasi zona dari registry (2), lalu
gunakan satu fungsi di semua titik masuk nama domain — user, admin, global,
dan reseller (termasuk parameter path `:name`):

```go
func prepareDomain(input string) (*v3.DomainResult, error) {
    name, err := v3.DefaultIdnUtils.NormalizeDomainName(input) // "Café。ID" → "café.id"
    if err != nil {
        return nil, err
    }
    return idnManager.CheckDomain(name)
}
```

- Ganti tag `binding:"isDomain"` dengan validator kustom yang memanggil
  `prepareDomain`, atau panggil `prepareDomain` di handler sebelum binding
  lain — yang penting normalisasi terjadi **sebelum** validasi.
- Pakai `res.Zone` untuk mencari harga, `epp_settings`, dan whitelist zona
  reseller, menggantikan `dom[1]…` dan `domainZone`.
- Batas panjang (`max=100`, `len(search) > 65`) sudah dicakup
  `ErrLabelTooLong`/`ErrDomainTooLong` yang mengukur bentuk A-label.

### 4.2 Penyimpanan dan pencarian

- Simpan `res.ASCIIName` di `domains.name`, `transfer_domains.name`, dan
  `reseller_domain_jobs.domain`; tambahkan kolom `unicode_name` untuk tampilan
  (migrasi baru di `migrations/sql/`).
- Semua pencarian berdasarkan nama (`FindByName`, `FindByNameAndUserID`, …)
  memakai `ASCIIName` hasil `prepareDomain`, sehingga `café.id`,
  `CAFÉ.ID`, dan `xn--caf-dma.id` menemukan baris yang sama.
- Tampilkan `UnicodeName` di respons API, email, item pembayaran, dan webhook
  (sertakan keduanya di webhook agar penerima tidak perlu mengonversi).

### 4.3 EPP

- Kirim `res.ASCIIName` ke semua fungsi di `core/Domain.go`
  (`SearchDomain`, `CheckDomain`, `CreateDomain`, `InfoDomain`, …), termasuk
  dari worker antrean (`queues/CreateDomain.go`, dst.) — worker membaca
  `domains.name`, jadi cukup dipastikan kolom itu berisi A-label.
- Setelah registry mendukung ekstensi IDN (3.5), tambahkan
  `frames.NewIDNCreate(res.TableName(), res.UnicodeName)` pada create untuk
  nama IDN.

### 4.4 Pesan kesalahan

Format respons yang ada:

- handler lama: `422` dengan `errors: []validators.ValidationError{Field, Reason}`;
- reseller: `respondErr(c, status, message, err)` dengan pesan berbahasa
  Indonesia.

Terjemahkan jenis kesalahan ke pesan pengguna dengan `errors.Is` — contoh
lengkap ada di `pesanUntukPengguna` pada
[examples/v3/registrar.go](../examples/v3/registrar.go). Sertakan karakter
dan posisinya (`ve.Rune`, `ve.Pos+1`) untuk `ErrCodepointNotInTable` dan
`ErrLanguageRule`; bagi pendaftar aksara Bali, "susunan aksara tidak sesuai
aturan penulisan (karakter '᭄' di posisi 4)" jauh lebih berguna daripada
"Format Domain is invalid".

### 4.5 Harga nama IDN

`len(sld)` harus diganti dengan ukuran yang ditetapkan kebijakan harga:
jumlah karakter (`utf8.RuneCountInString(res.ULabel)`) atau panjang A-label
(`len(res.ALabel)`). Ini keputusan bisnis; hasilnya berbeda untuk aksara
non-Latin.

## 5. Urutan kerja yang disarankan

1. Tag modul (1).
2. nomnom: boot dengan `IdnManager` dan tabel tersemat (3.1) — langsung
   menutup celah container tanpa tabel.
3. nomnom: `CheckDomain` di create, availability, dan registrar-api internal
   (3.2, 3.3, 3.7); simpan `ASCIIName`.
4. nomnom: migrasi kolom + backfill + laporan data lama; indeks unik
   `variant_key` (3.4).
5. registrar-api: `prepareDomain` di semua titik masuk, terutama reseller
   (4.1), lalu penyimpanan dan pesan kesalahan (4.2, 4.4).
6. nomnom: RDE `rdeIDN` dan RDAP `unicodeName` (3.6, 3.7).
7. Ekstensi EPP IDN di kedua sisi (3.5, 4.3).
8. Hapus `pkg/idn` di nomnom.

## 6. Keputusan yang perlu diambil sebelum implementasi

| Keputusan | Pilihan |
|---|---|
| Tabel per zona `.id`, `co.id`, dst. dan urutannya | mis. `latin` + `und-bali` (LGR), dan `und-java` (LGR Jawa, rilis ICANN 13 Agustus 2026) bila aksara Jawa akan dibuka |
| Campuran aksara per zona (`SingleScript`) | ikuti tabel (default), atau satu script saja |
| Varian `allocatable` | boleh dialokasikan ke pendaftar yang sama, atau semua varian diblokir |
| Result code EPP untuk penolakan kebijakan tabel | 2005 atau 2306 |
| URL tabel dan kebijakan PANDI di IANA untuk RDE | wajib sebelum deposit memuat `rdeIDN` |
| Dasar harga nama IDN | jumlah karakter atau panjang A-label |
| Data lama yang ditolak saat backfill | dibiarkan, dihubungi pendaftarnya, atau dihapus |
