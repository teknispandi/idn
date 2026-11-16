# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Go library (`github.com/al0fayz/idn`) that validates IDN labels for a domain registry. The core is ported from Google Nomulus's IDN validation and extended with runtime-loaded tables and script-specific LGR validators.

- `v3/` is the only implemented package. It validates against IDN tables in Nomulus `.txt` format.
- The root package `idn` (`tables.go`) only embeds `idn/*.txt`. `idn.Tables()` returns an `fs.FS` for `v3.NewIdnManagerFS`.
- `v4/` is an RFC 7940 LGR XML engine. It depends on `v3`, and an `*v4.LGR` plugs into `v3.IdnManager` via `RegisterTable`.
- `v3/README.md` still has a "Migration from V2" section, but no v2 exists in this repo.

## Commands

```bash
go test ./...                       # all tests
go test ./v3 -run TestBalinese -v   # single test / pattern
go test ./v3 -run '^$' -bench .     # benchmarks only
go vet ./...
go generate ./v4                    # regenerate v4/joiningtype_table.go (must be byte-identical)
cd examples/v3 && go run .          # runs the demo main.go currently calls
cd examples/v4 && go run .          # LGR XML demo
go test ./v3 ./v4 -run '^$' -bench . -benchmem   # numbers quoted in v3/README.md and v4/README.md
```

Tests and examples find data through relative paths: `../idn` and `../lgrs/...` from `v3/`, and `../../idn` from `examples/v3/`. `go test` runs from the package dir automatically, but `go run` for the examples needs the `cd`. `examples/v3/main.go` calls one demo function. To switch demos, change which call is uncommented.

## Architecture (v3)

The full validation pipeline is `IdnManager.CheckDomain` / `CheckLabel` (`idn_manager.go`), which runs these layers in order:

1. **`prepareLabel` / `prepareDomain` (`idna2008.go`)** is the single IDNA2008 gate, shared by every entry point. It turns a U-label or A-label into a `preparedLabel` holding both forms. It lowercases ASCII only; non-ASCII uppercase is rejected, not mapped. A-labels are decoded, revalidated and must round-trip canonically. It also implements what `x/net/idna` lacks: hyphen positions counted in code points, RFC 5892 CONTEXTO, NV8 rejection via a PVALID approximation (`isIDNA2008Allowed`), and the bidi rule. `idna.Registration` runs last as a safety net, which also covers CONTEXTJ.
2. **Zone match** (`checkDomain`) picks the longest registered suffix (`co.id` before `id`). The name must be exactly one label under the zone and must not itself be a zone. Zone keys are normalized to A-labels.
3. **Tables** (`matchTables`, `result.go`) try the TLD's tables in configured order, and the first one that accepts the label is recorded. That table is the RDE `idnTableId`, so the order must stay deterministic. Pure-LDH labels skip the tables when `TldIdnConfig.AllowASCII` is set, but they still take their `VariantKey` from the first table that accepts them, because variants can cross into ASCII (`ß` ↔ `ss`). On rejection it returns the table error that got furthest into the label.

Tables are the `LabelTable` interface (`variants.go`). `*IdnTable` (`.txt`) and `*v4.LGR` both implement it, plus `VariantTable`. `LabelResult.VariantKey` is the RFC 7940 §8.5 index label in ASCII. Registries enforce `UNIQUE(zone, variant_key)` instead of generating variants. RFC 3743 variant columns in `.txt` tables (the Chinese tables) are parsed into union-find sets; preferred variants become `allocatable` and the rest `blocked`. `RdeTableRefs` takes `TldIdnConfig.TableRefs` first, then table headers, and errors rather than returning an empty URL.

`IdnTable.CheckLabel` (`idn.go`) does only the code point and language-rule check, on a U-label. URL and Policy come solely from the table's `# URL:` / `# Policy:` headers and are `nil` otherwise; never fabricate them, because they go into RDE deposits.

All rejections are `*ValidationError` (`errors.go`). Its `Kind` is one of the `Err*` sentinels (match with `errors.Is`), plus position, rune, table and rule. Keep that shape when adding checks, because registrars build user-facing messages from it.

Language validators attach to a table in two ways. One is by table name in `languageValidators`: only `"ja"`, for the legacy embedded `v3/japanese.txt`. The other is by the `# Script:` header in `scriptValidators` (`arab`, `und-bali`), done automatically in `NewIdnTable`. `idn/Japanese-IDN.txt` v2.0 intentionally gets no Japanese validator, because it dropped U+30FB/U+30FC. Implement `LanguageRuleChecker` to report the rule name. `JapaneseLanguageValidator` follows Nomulus and uses Unicode *blocks*, not scripts (U+30FC is block Katakana, script Common).

There are two table sources:

1. **Loader/manager** (`idn_loader.go`, `idn_manager.go`) reads from an `fs.FS`: a directory via `NewIdnLoader(path)` or the embedded `idn.Tables()`. Names match file stems case-insensitively, without `-IDN.txt`/`.txt` (`und-bali` → `und-Bali.txt`). `IdnManager` is concurrency-safe. Per-TLD table slices are copy-on-write, so they can be used after the lock is released.
2. **Nomulus-compatible registry** (`registry.go`, `validator.go`). `//go:embed *.txt` embeds `v3/*.txt`, keyed by `IdnTableType`. `IdnLabelValidator.CheckLabelForTld` / `FindValidIdnTableForTld` follow Nomulus: even ASCII labels must match a table, and it falls back to `ExtendedLatin` + `Japanese`.

`VariantConflicts` / `CheckVariantCoherence` implement ICANN Guideline 13. Every pair of a TLD's tables must agree on the variant index of every shared element, because `VariantKey` comes from whichever table accepted the label. Tables expose their sets through `VariantSetTable`.

`TestTablesAreIDNA2008Valid` asserts that every table code point passes the IDNA2008 layer. `TestIDNA2008AgainstIANA` pins rows quoted from IANA's idna-tables-12.0.0. Run it after adding or updating a table. `TestIDNA2008MatchesPrecis` pins the PVALID approximation against `x/text/secure/precis`; the only allowed differences are IgnorableBlocks and some `Lm` modifier letters. Unstable is detected through the UTS46 mapping (`mapProfile`), not `x/text/cases.Fold`, which flips Cherokee case.

## Architecture (v4)

`lgr.go` parses the XML into a generic `node` tree, then builds the repertoire. Sequences and ranges are expanded, and the `tags` index feeds `from-tag`. `compile()` eagerly compiles every named class and rule, so a broken or unsupported LGR fails at load time; keep that property. `pattern.go` is a continuation-passing backtracking matcher over `[]rune`. Go `regexp` can't express look-behind/look-ahead/anchor. Rules are searched anywhere in the label unless `start`/`end` anchor them. `evaluate.go` decomposes labels with DP (contexts depend only on label and position), evaluates actions in order with RFC 7940 §7.6 default actions appended, and generates variants with hard caps.

`TestBalineseLGRMatchesHandwrittenValidator` compares the engine against `v3.BalineseLanguageValidator` on ~392k labels. Any change to either must keep them in agreement. `jt:` data comes from `internal/gen/joiningtype.go`, which probes PRECIS ZWNJ context rules. `x/net/idna` can't be used for this, because its joiner state machine treats type U like R.

### Script validators follow the LGR XML, not its prose

`BalineseLanguageValidator` implements `lgrs/second-level-reference/und-Bali.xml`. The XML `<rules>` are normative. The prose description inside the LGR is not, and reading it literally caused past bugs: two consonants before a virama were wrongly rejected, and so was a word-final adeg-adeg. When changing a script validator:

- Tie each check to its named LGR rule (`follows-c`, `disallow-for-1B44`, …), as the existing comments do.
- Keep the repertoire identical to the XML `<data>`. `TestBalineseRepertoireMatchesLGR` enforces this.
- Count hyphen rules in runes inside the validator. `x/net/idna` counts bytes, so its checks never fire for multi-byte scripts.

## Data directories

- `idn/`: Nomulus-format IDN tables read by `IdnLoader`. Keep only the latest version of each table (see `idn/README.md`).
- `lgrs/`: ICANN LGR XML sets (`root-zone`, `second-level-reference`, `full-variant-set`). They are the source of truth for script rules. Check them against ICANN's published versions (links in `README.md`).
- `doc/`: notes on RFC 5890–5893, the ICANN IDN guideline PDFs, `Notes.md` (v3 flow diagrams), `Perbandingan.md` (v3 vs v4/LGR comparison) and `Integrasi-Registry-Registrar.md` (integration plan for nomnom and registrar-api). Keep the integration plan in sync when public APIs change.

## Conventions

Code comments, docs and recent commit messages are mostly in Bahasa Indonesia. Match the language already used in the file you're editing.
