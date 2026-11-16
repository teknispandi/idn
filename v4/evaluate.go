package v4

import (
	v3 "github.com/al0fayz/idn/v3"
)

// Batas pencarian untuk label dengan banyak partisi atau kombinasi varian.
const (
	maxPartitions   = 64
	maxCombinations = 200_000
)

// placed adalah elemen repertoire pada posisi tertentu di label.
type placed struct {
	e          *element
	start, end int
}

// contextOK mengevaluasi when/not-when (RFC 7940 §5.2) untuk span [start,end).
func (l *LGR) contextOK(when, notWhen string, label []rune, start, end int) bool {
	if when != "" && !l.matches(when, label, start, end) {
		return false
	}
	if notWhen != "" && l.matches(notWhen, label, start, end) {
		return false
	}
	return true
}

// candidates mengembalikan elemen repertoire yang cocok di pos, terpanjang dulu.
func (l *LGR) candidates(label []rune, pos int) []*element {
	var out []*element
	for _, e := range l.byFirst[label[pos]] {
		end := pos + len(e.cps)
		if end <= len(label) && equalRunes(label[pos:end], e.cps) {
			out = append(out, e)
		}
	}
	return out
}

// partitionable[pos]: label[pos:] dapat dipecah menjadi elemen repertoire
// yang konteksnya terpenuhi. Konteks hanya bergantung pada label dan
// posisi, jadi pemrograman dinamis dari belakang sudah tepat.
func (l *LGR) partitionable(label []rune) []bool {
	n := len(label)
	ok := make([]bool, n+1)
	ok[n] = true
	for pos := n - 1; pos >= 0; pos-- {
		for _, e := range l.candidates(label, pos) {
			end := pos + len(e.cps)
			if ok[end] && l.contextOK(e.when, e.notWhen, label, pos, end) {
				ok[pos] = true
				break
			}
		}
	}
	return ok
}

// decompose mengembalikan partisi label yang mengutamakan elemen terpanjang,
// atau *v3.ValidationError yang menunjuk code point pertama yang gagal.
func (l *LGR) decompose(label []rune) ([]placed, error) {
	ok := l.partitionable(label)
	if !ok[0] {
		return nil, l.explainFailure(label)
	}
	var parts []placed
	for pos := 0; pos < len(label); {
		for _, e := range l.candidates(label, pos) {
			end := pos + len(e.cps)
			if ok[end] && l.contextOK(e.when, e.notWhen, label, pos, end) {
				parts = append(parts, placed{e, pos, end})
				pos = end
				break
			}
		}
	}
	return parts, nil
}

// explainFailure mencari posisi terjauh yang masih bisa dicapai dari awal
// label; di sana tidak ada elemen repertoire, atau semua konteksnya gagal.
func (l *LGR) explainFailure(label []rune) error {
	n := len(label)
	reach := make([]bool, n+1)
	reach[0] = true
	failAt := 0
	for pos := 0; pos < n; pos++ {
		if !reach[pos] {
			continue
		}
		failAt = pos
		for _, e := range l.candidates(label, pos) {
			end := pos + len(e.cps)
			if l.contextOK(e.when, e.notWhen, label, pos, end) {
				reach[end] = true
			}
		}
	}

	e := &v3.ValidationError{Kind: v3.ErrCodepointNotInTable, Label: string(label), Pos: failAt, Rune: label[failAt], Table: l.name, Rule: "repertoire"}
	if cands := l.candidates(label, failAt); len(cands) > 0 {
		c := cands[0]
		e.Kind, e.Rule = v3.ErrLanguageRule, c.when
		if c.when == "" || l.matches(c.when, label, failAt, failAt+len(c.cps)) {
			e.Rule = c.notWhen
		}
	}
	return e
}

// variantTypes mengumpulkan tipe varian yang dipakai sebuah label (§8.2
// langkah 3). mapped: setiap posisi berasal dari pemetaan varian (untuk
// only-variants).
type variantTypes struct {
	types  map[string]bool
	mapped bool
}

// firstAction mengembalikan aksi pertama yang terpicu (§8.3). Aksi berbasis
// tipe varian hanya terpicu bila ada tipe yang tercatat; tanpa itu aksi
// default all-variants="activated" akan berlaku untuk setiap label.
func (l *LGR) firstAction(label []rune, vt variantTypes) *action {
	for i := range l.actions {
		a := &l.actions[i]
		if a.match != "" && !l.matches(a.match, label, -1, -1) {
			continue
		}
		if a.notMatch != "" && l.matches(a.notMatch, label, -1, -1) {
			continue
		}
		if a.anyVariant != nil && !anyIn(vt.types, a.anyVariant) {
			continue
		}
		if a.allVariants != nil && !allIn(vt.types, a.allVariants) {
			continue
		}
		if a.onlyVariants != nil && (!vt.mapped || !allIn(vt.types, a.onlyVariants)) {
			continue
		}
		return a
	}
	return &l.actions[len(l.actions)-1]
}

func anyIn(types, set map[string]bool) bool {
	for t := range types {
		if set[t] {
			return true
		}
	}
	return false
}

func allIn(types, set map[string]bool) bool {
	if len(types) == 0 {
		return false
	}
	for t := range types {
		if !set[t] {
			return false
		}
	}
	return true
}

// Disposition mengembalikan disposisi label asli (§8.3). Untuk label yang
// tidak "valid", error menjelaskan sebabnya (*v3.ValidationError).
func (l *LGR) Disposition(label string) (v3.Disposition, error) {
	runes := []rune(label)
	if len(runes) == 0 {
		return v3.DispositionInvalid, &v3.ValidationError{Kind: v3.ErrEmptyLabel, Pos: -1, Table: l.name}
	}
	parts, err := l.decompose(runes)
	if err != nil {
		return v3.DispositionInvalid, err
	}

	// Label asli: setiap elemen dipetakan ke dirinya sendiri; tipe varian
	// refleksifnya yang dicatat.
	vt := variantTypes{types: make(map[string]bool), mapped: true}
	for _, p := range parts {
		r := p.e.reflexive
		if r != nil && l.contextOK(r.when, r.notWhen, runes, p.start, p.end) {
			vt.types[r.typ] = true
		} else {
			vt.mapped = false
		}
	}

	a := l.firstAction(runes, vt)
	if a.disp == v3.DispositionValid {
		return a.disp, nil
	}
	rule := a.match
	if rule == "" {
		rule = "action disp=" + string(a.disp)
	}
	return a.disp, &v3.ValidationError{Kind: v3.ErrLanguageRule, Label: label, Pos: -1, Table: l.name, Rule: rule}
}

// CheckLabel mengimplementasikan v3.LabelTable: label diterima hanya bila
// disposisinya "valid".
func (l *LGR) CheckLabel(label string) error {
	_, err := l.Disposition(label)
	return err
}

// VariantIndex mengimplementasikan v3.VariantTable (RFC 7940 §8.5).
func (l *LGR) VariantIndex(label string) string {
	if !l.HasVariants() {
		return label
	}
	runes := []rune(label)
	parts, err := l.decompose(runes)
	if err != nil {
		return label
	}
	out := make([]rune, 0, len(runes))
	for _, p := range parts {
		if idx, ok := l.index[string(p.e.cps)]; ok {
			out = append(out, []rune(idx)...)
		} else {
			out = append(out, p.e.cps...)
		}
	}
	return string(out)
}

// VariantLabels mengimplementasikan v3.VariantTable (RFC 7940 §8.2): semua
// permutasi pemetaan varian pada semua partisi label, tanpa label yang
// disposisinya "invalid". Kosong jika label asli tidak valid.
func (l *LGR) VariantLabels(label string, limit int) ([]v3.VariantLabel, bool) {
	if !l.HasVariants() {
		return nil, false
	}
	if limit <= 0 {
		limit = v3.DefaultVariantLimit
	}
	runes := []rune(label)
	if disp, _ := l.Disposition(label); disp == v3.DispositionInvalid {
		return nil, false
	}

	g := &variantGenerator{l: l, original: runes, limit: limit, seen: map[string]bool{label: true}}
	for _, partition := range l.partitions(runes) {
		g.generate(partition)
		if g.truncated {
			break
		}
	}
	return g.out, g.truncated
}

// partitions menghitung semua partisi label yang memenuhi konteks (§8.2:
// "all eligible partitions"), paling banyak maxPartitions.
func (l *LGR) partitions(label []rune) [][]placed {
	ok := l.partitionable(label)
	var out [][]placed
	var walk func(pos int, acc []placed)
	walk = func(pos int, acc []placed) {
		if len(out) >= maxPartitions {
			return
		}
		if pos == len(label) {
			out = append(out, append([]placed(nil), acc...))
			return
		}
		for _, e := range l.candidates(label, pos) {
			end := pos + len(e.cps)
			if ok[end] && l.contextOK(e.when, e.notWhen, label, pos, end) {
				walk(end, append(acc, placed{e, pos, end}))
			}
		}
	}
	if ok[0] {
		walk(0, nil)
	}
	return out
}

type variantOption struct {
	cps    []rune
	typ    string
	mapped bool
}

type variantGenerator struct {
	l         *LGR
	original  []rune
	limit     int
	seen      map[string]bool
	out       []v3.VariantLabel
	tried     int
	truncated bool
}

func (g *variantGenerator) generate(partition []placed) {
	options := make([][]variantOption, len(partition))
	for i, p := range partition {
		identity := variantOption{cps: p.e.cps}
		if r := p.e.reflexive; r != nil && g.l.contextOK(r.when, r.notWhen, g.original, p.start, p.end) {
			identity.typ, identity.mapped = r.typ, true
		}
		options[i] = []variantOption{identity}
		for _, v := range p.e.vars {
			// Kondisi varian dievaluasi pada label asli (§8.2 langkah 2).
			if g.l.contextOK(v.when, v.notWhen, g.original, p.start, p.end) {
				options[i] = append(options[i], variantOption{cps: v.cps, typ: v.typ, mapped: true})
			}
		}
	}

	var buf []rune
	chosen := make([]variantOption, len(partition))
	var walk func(i int)
	walk = func(i int) {
		if g.truncated {
			return
		}
		if i == len(partition) {
			g.emit(buf, chosen)
			return
		}
		for _, o := range options[i] {
			chosen[i] = o
			n := len(buf)
			buf = append(buf, o.cps...)
			walk(i + 1)
			buf = buf[:n]
		}
	}
	walk(0)
}

func (g *variantGenerator) emit(label []rune, chosen []variantOption) {
	g.tried++
	if g.tried > maxCombinations {
		g.truncated = true
		return
	}
	s := string(label)
	if g.seen[s] {
		return
	}
	g.seen[s] = true

	// Label varian harus valid sendiri: repertoire dan konteks code point (§8.3).
	if _, err := g.l.decompose(label); err != nil {
		return
	}
	vt := variantTypes{types: make(map[string]bool), mapped: true}
	var types []string
	for _, o := range chosen {
		if o.typ != "" && !vt.types[o.typ] {
			vt.types[o.typ] = true
			types = append(types, o.typ)
		}
		vt.mapped = vt.mapped && o.mapped
	}
	disp := g.l.firstAction(label, vt).disp
	if disp == v3.DispositionInvalid {
		return
	}
	if len(g.out) == g.limit {
		g.truncated = true
		return
	}
	g.out = append(g.out, v3.VariantLabel{ULabel: s, Disposition: disp, Types: types})
}
