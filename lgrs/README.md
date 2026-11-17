# LGR (Label Generation Rulesets)

Salinan LGR resmi ICANN dalam format XML RFC 7940, dipakai package `v4`.
Nama file = tag bahasa dari `<meta><language>` (mis. `und-Bali.xml`,
`de.xml`), bukan nama file asli ICANN.

Diperiksa terhadap sumber resmi pada **17 November 2026**.

| Folder | Rilis | Sumber |
|---|---|---|
| `root-zone/` (26 file) | RZ-LGR-6, 23 September 2025 | [Root Zone LGR](https://www.icann.org/resources/pages/root-zone-lgr-2015-06-21-en), paket `rz-lgr-6-package-23sep25-en.zip` |
| `second-level-reference/` (61 file) | 13 Agustus 2026 | [Second Level Reference LGRs](https://www.icann.org/en/contracted-parties/registry-operators/second-level-reference-label-generation-rules-21-06-2015-en), paket `lgr-second-level-publication-files-complete-13aug26-en.zip` |
| `full-variant-set/` (59 file) | 25 Oktober 2024; Jerman v3.1, 8 April 2026 | [Full Variant Set LGRs](https://newgtldprogram.icann.org/en/application-rounds/round2/rsp/full-variant-set-lgrs), file `lgr-second-level-*-rsp-full-variant-*-en.xml` |

Perubahan normatif pada pembaruan November 2026 (selain itu hanya metadata,
komentar, referensi, dan tag tanda hubung `sc:Zyyy` → `Hyphen sc:Zyyy`):

- `second-level-reference`: baru `und-Java` (Jawa) dan `und-Cans` (UCAS);
  berubah `ar`, `und-Arab`, `iu-Cans`, `ja`, `und-Jpan`, `und-Latn`, `es`.
  `und-Bali` sudah versi 13 Agustus 2026.
- `root-zone`: baru `und-Thaa`; berubah `und-Beng`, `und-Deva`, `und-Jpan`,
  `und-Khmr`.
- `full-variant-set`: `de` v3 → v3.1.

Beberapa rilis menyatakan `<unicode-version>` 15.1.0/16.0.0, lebih baru dari
Unicode runtime Go (15.0.0). Semua code point di dalamnya sudah dikenal
Unicode 15.0, jadi `v4` tetap memuatnya; LGR yang memakai code point yang
belum dikenal runtime akan ditolak saat dimuat.

Paket second-level 13 Agustus 2026 juga memuat file
`lgr-second-level-*-full-variant-script-13aug26-en.xml`. Itu bukan seri
Full Variant Set (halaman resminya masih merujuk `rsp-full-variant-…-25oct24`),
jadi tidak disalin ke `full-variant-set/`.

## Memperbarui

1. Unduh paket/file resmi dari tiga halaman di atas ke folder sementara.
2. Salin setiap file ke folder yang sesuai dengan nama `<language>.xml`.
3. Jalankan `go test ./v4` — `TestLoadAllLGRs` menolak LGR yang tidak
   didukung, dan `TestBalineseLGRMatchesHandwrittenValidator` memastikan
   `v3.BalineseLanguageValidator` masih sesuai `und-Bali.xml`.
4. Perbarui tabel dan tanggal di README ini.
