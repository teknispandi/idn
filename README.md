
# IDN Utils
- validate domain name with:
    - idna 2008
    - validasi idn dengan icann idn guideline v2
    - validasi idn dengan icann idn guideline v4
- cache file for LGR format 
    - idn format (versi 2)
    - format for root zone LGR
    - format for second level LGR
    - format for full avriant set LGR

## LGR Sets
> (pastikan data uptodate dengan anda check url di bawah) — terakhir diperiksa
> 17 November 2026; versi, perubahan, dan cara memperbarui ada di
> [lgrs/README.md](lgrs/README.md)

> idn data format dari google nomulus registry (`core/src/main/java/google/registry/idn`,
> identik dengan Nomulus `master` per 17 November 2026)

Tiga set LGR ICANN di `lgrs/`:

* [`root-zone`](lgrs/root-zone) — RZ-LGR-6, 23 September 2025 ([more info](https://www.icann.org/resources/pages/root-zone-lgr-2015-06-21-en))
* [`second-level-reference`](lgrs/second-level-reference) — 13 Agustus 2026, termasuk Jawa (`und-Java`) dan Bali (`und-Bali`) ([more info](https://www.icann.org/en/contracted-parties/registry-operators/second-level-reference-label-generation-rules-21-06-2015-en))
* [`full-variant-set`](lgrs/full-variant-set) — 25 Oktober 2024, Jerman v3.1 8 April 2026 ([more info](https://newgtldprogram.icann.org/en/application-rounds/round2/rsp/full-variant-set-lgrs))

# Use Case
- use v3 for idn validation using format .txt
    - registry & registrar: `v3.NewIdnManagerFS(idn.Tables())` + `CheckDomain` — lihat [v3/README.md](v3/README.md#penggunaan-untuk-registry--registrar)
- use v4 for idn validation using format .xml (LGR) — dipasang ke `v3.IdnManager` lewat `RegisterTable`, lihat [v4/README.md](v4/README.md)
- integrasi ke nomnom (registry) dan registrar-api: [doc/Integrasi-Registry-Registrar.md](doc/Integrasi-Registry-Registrar.md)