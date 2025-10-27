
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
> (pastikan data uptodate dengan anda check url di bawah)

> idn data format dari google nomulus registry

LGR Server comes bundled with three sets of LGRs:

* [`root-zone`](lgrs/root-zone) ([more info](https://www.icann.org/resources/pages/root-zone-lgr-2015-06-21-en))
* [`second-level-reference`](lgrs/second-level-reference) ([more info](https://www.icann.org/en/contracted-parties/registry-operators/second-level-reference-label-generation-rules-21-06-2015-en))
* [`full-variant-set`](lgrs/full-variant-set) ([more info](https://newgtldprogram.icann.org/en/application-rounds/round2/rsp/full-variant-set-lgrs))

# Use Case
- use v3 for idn validation using format .txt
- use v4 for idn validation using format .xml (LGR)