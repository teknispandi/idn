package v4

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// runeSet adalah kelas RFC 7940 §6.2: himpunan code point tunggal.
type runeSet func(r rune) bool

// pattern mencocokkan bagian label mulai dari pos dan memanggil k dengan
// posisi akhir setiap kecocokan (backtracking); true jika k menerima.
type pattern func(s *matchState, pos int, k func(end int) bool) bool

// matchState adalah label yang dievaluasi dan posisi anchor-nya.
// anchorStart < 0 berarti aturan dievaluasi tanpa anchor (WLE).
type matchState struct {
	label       []rune
	anchorStart int
	anchorEnd   int
}

const (
	stateNew = iota
	stateCompiling
	stateDone
)

type namedSet struct {
	def   *node
	set   runeSet
	state int
}

type namedRule struct {
	def   *node
	pat   pattern
	state int
}

// matches melaporkan apakah aturan bernama cocok di mana pun dalam label
// (RFC 7940 §6.3.8: tanpa <start/>/<end/> aturan tidak terjangkar). Untuk
// aturan konteks, anchor menandai posisi elemen yang dievaluasi (§6.4).
func (l *LGR) matches(ruleName string, label []rune, anchorStart, anchorEnd int) bool {
	p := l.rules[ruleName].pat
	s := &matchState{label: label, anchorStart: anchorStart, anchorEnd: anchorEnd}
	accept := func(int) bool { return true }
	for start := 0; start <= len(label); start++ {
		if p(s, start, accept) {
			return true
		}
	}
	return false
}

func (l *LGR) namedClass(name string) (runeSet, error) {
	ns, ok := l.classes[name]
	if !ok {
		return nil, fmt.Errorf("undefined class %q", name)
	}
	switch ns.state {
	case stateDone:
		return ns.set, nil
	case stateCompiling:
		return nil, fmt.Errorf("class %q is defined recursively", name)
	}
	ns.state = stateCompiling
	set, err := l.classSet(ns.def)
	if err != nil {
		return nil, fmt.Errorf("class %q: %w", name, err)
	}
	ns.set, ns.state = set, stateDone
	return set, nil
}

func (l *LGR) namedPattern(name string) (pattern, error) {
	nr, ok := l.rules[name]
	if !ok {
		return nil, fmt.Errorf("undefined rule %q", name)
	}
	switch nr.state {
	case stateDone:
		return nr.pat, nil
	case stateCompiling:
		return nil, fmt.Errorf("rule %q is defined recursively", name)
	}
	nr.state = stateCompiling
	p, err := l.sequence(nr.def.Children)
	if err != nil {
		return nil, fmt.Errorf("rule %q: %w", name, err)
	}
	nr.pat, nr.state = p, stateDone
	return p, nil
}

// classSet mengompilasi <class> atau operator himpunan (§6.2.4).
func (l *LGR) classSet(n *node) (runeSet, error) {
	switch n.XMLName.Local {
	case "class":
		if ref := n.attr("by-ref"); ref != "" {
			return l.namedClass(ref)
		}
		if tag, ok := n.attrOK("from-tag"); ok {
			members := make(map[rune]bool)
			for _, r := range l.tags[tag] {
				members[r] = true
			}
			return func(r rune) bool { return members[r] }, nil
		}
		if property, ok := n.attrOK("property"); ok {
			return propertySet(property)
		}
		return parseClassText(n.Text)
	case "complement":
		sets, err := l.childSets(n, 1)
		if err != nil {
			return nil, err
		}
		return func(r rune) bool { return !sets[0](r) }, nil
	case "difference":
		sets, err := l.childSets(n, 2)
		if err != nil {
			return nil, err
		}
		return func(r rune) bool { return sets[0](r) && !sets[1](r) }, nil
	case "symmetric-difference":
		sets, err := l.childSets(n, 2)
		if err != nil {
			return nil, err
		}
		return func(r rune) bool { return sets[0](r) != sets[1](r) }, nil
	case "union":
		sets, err := l.childSets(n, -1)
		if err != nil {
			return nil, err
		}
		return func(r rune) bool {
			for _, s := range sets {
				if s(r) {
					return true
				}
			}
			return false
		}, nil
	case "intersection":
		sets, err := l.childSets(n, -1)
		if err != nil {
			return nil, err
		}
		return func(r rune) bool {
			for _, s := range sets {
				if !s(r) {
					return false
				}
			}
			return true
		}, nil
	}
	return nil, fmt.Errorf("<%s> is not a class", n.XMLName.Local)
}

// childSets mengompilasi operand operator himpunan; want < 0 berarti ≥ 1.
func (l *LGR) childSets(n *node, want int) ([]runeSet, error) {
	sets := make([]runeSet, 0, len(n.Children))
	for i := range n.Children {
		s, err := l.classSet(&n.Children[i])
		if err != nil {
			return nil, err
		}
		sets = append(sets, s)
	}
	if (want > 0 && len(sets) != want) || len(sets) == 0 {
		return nil, fmt.Errorf("<%s> needs %d operand(s), has %d", n.XMLName.Local, want, len(sets))
	}
	return sets, nil
}

// propertySet menerjemahkan atribut property (§6.2.3). Didukung: gc
// (General_Category, tabel Unicode Go) dan jt (Joining_Type R, D, L dari
// joiningtype_table.go). Properti lain ditolak saat LGR dimuat.
func propertySet(property string) (runeSet, error) {
	name, value, ok := strings.Cut(property, ":")
	if ok {
		var table *unicode.RangeTable
		switch name {
		case "gc":
			table = unicode.Categories[value]
		case "jt":
			table = joiningTypes[value]
		}
		if table != nil {
			return func(r rune) bool { return unicode.Is(table, r) }, nil
		}
	}
	return nil, fmt.Errorf("unsupported Unicode property %q", property)
}

// parseClassText membaca isi <class>: code point dan rentang "XXXX-YYYY".
func parseClassText(text string) (runeSet, error) {
	members := make(map[rune]bool)
	var ranges [][2]rune
	for _, field := range strings.Fields(text) {
		lo, hi, isRange := strings.Cut(field, "-")
		first, err := parseCodePoint(lo)
		if err != nil {
			return nil, err
		}
		if !isRange {
			members[first] = true
			continue
		}
		last, err := parseCodePoint(hi)
		if err != nil || last < first {
			return nil, fmt.Errorf("invalid range %q", field)
		}
		ranges = append(ranges, [2]rune{first, last})
	}
	return func(r rune) bool {
		if members[r] {
			return true
		}
		for _, rg := range ranges {
			if r >= rg[0] && r <= rg[1] {
				return true
			}
		}
		return false
	}, nil
}

// sequence mengompilasi anak-anak sebuah aturan sebagai urutan.
func (l *LGR) sequence(children []node) (pattern, error) {
	ps := make([]pattern, 0, len(children))
	for i := range children {
		p, err := l.element(&children[i])
		if err != nil {
			return nil, err
		}
		ps = append(ps, p)
	}
	return seq(ps), nil
}

func seq(ps []pattern) pattern {
	switch len(ps) {
	case 0:
		return func(s *matchState, pos int, k func(int) bool) bool { return k(pos) }
	case 1:
		return ps[0]
	}
	first, rest := ps[0], seq(ps[1:])
	return func(s *matchState, pos int, k func(int) bool) bool {
		return first(s, pos, func(next int) bool { return rest(s, next, k) })
	}
}

// element mengompilasi satu elemen aturan (§6.3–6.4).
func (l *LGR) element(n *node) (pattern, error) {
	var p pattern
	switch n.XMLName.Local {
	case "char":
		cps, err := parseCodePoints(n.attr("cp"))
		if err != nil || len(cps) == 0 {
			return nil, fmt.Errorf("char cp=%q: invalid code point", n.attr("cp"))
		}
		p = func(s *matchState, pos int, k func(int) bool) bool {
			end := pos + len(cps)
			return end <= len(s.label) && equalRunes(s.label[pos:end], cps) && k(end)
		}
	case "class", "union", "intersection", "difference", "symmetric-difference", "complement":
		set, err := l.classSet(n)
		if err != nil {
			return nil, err
		}
		p = func(s *matchState, pos int, k func(int) bool) bool {
			return pos < len(s.label) && set(s.label[pos]) && k(pos+1)
		}
	case "any":
		p = func(s *matchState, pos int, k func(int) bool) bool {
			return pos < len(s.label) && k(pos+1)
		}
	case "start":
		p = func(s *matchState, pos int, k func(int) bool) bool { return pos == 0 && k(pos) }
	case "end":
		p = func(s *matchState, pos int, k func(int) bool) bool { return pos == len(s.label) && k(pos) }
	case "anchor":
		p = func(s *matchState, pos int, k func(int) bool) bool {
			return s.anchorStart >= 0 && pos == s.anchorStart && k(s.anchorEnd)
		}
	case "look-behind":
		inner, err := l.sequence(n.Children)
		if err != nil {
			return nil, err
		}
		p = func(s *matchState, pos int, k func(int) bool) bool {
			for start := pos; start >= 0; start-- {
				if inner(s, start, func(end int) bool { return end == pos }) {
					return k(pos)
				}
			}
			return false
		}
	case "look-ahead":
		inner, err := l.sequence(n.Children)
		if err != nil {
			return nil, err
		}
		p = func(s *matchState, pos int, k func(int) bool) bool {
			return inner(s, pos, func(int) bool { return true }) && k(pos)
		}
	case "choice":
		alternatives := make([]pattern, 0, len(n.Children))
		for i := range n.Children {
			a, err := l.element(&n.Children[i])
			if err != nil {
				return nil, err
			}
			alternatives = append(alternatives, a)
		}
		p = func(s *matchState, pos int, k func(int) bool) bool {
			for _, a := range alternatives {
				if a(s, pos, k) {
					return true
				}
			}
			return false
		}
	case "rule":
		var err error
		if ref := n.attr("by-ref"); ref != "" {
			p, err = l.namedPattern(ref)
		} else {
			p, err = l.sequence(n.Children)
		}
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported rule element <%s>", n.XMLName.Local)
	}

	if count, ok := n.attrOK("count"); ok {
		min, max, err := parseCount(count)
		if err != nil {
			return nil, err
		}
		p = repeat(p, min, max)
	}
	return p, nil
}

// parseCount membaca atribut count (§6.3.3): "n", "n+" atau "n:m".
// max < 0 berarti tanpa batas.
func parseCount(s string) (min, max int, err error) {
	switch {
	case strings.HasSuffix(s, "+"):
		min, err = strconv.Atoi(strings.TrimSuffix(s, "+"))
		max = -1
	case strings.Contains(s, ":"):
		lo, hi, _ := strings.Cut(s, ":")
		if min, err = strconv.Atoi(lo); err == nil {
			max, err = strconv.Atoi(hi)
		}
	default:
		min, err = strconv.Atoi(s)
		max = min
	}
	if err != nil || min < 0 || (max >= 0 && max < min) {
		return 0, 0, fmt.Errorf("invalid count %q", s)
	}
	return min, max, nil
}

// repeat mencocokkan p sebanyak min..max kali, rakus dengan backtracking.
func repeat(p pattern, min, max int) pattern {
	var rec func(s *matchState, pos, n int, k func(int) bool) bool
	rec = func(s *matchState, pos, n int, k func(int) bool) bool {
		if max < 0 || n < max {
			if p(s, pos, func(next int) bool {
				if next == pos { // kecocokan lebar nol: jangan berulang tanpa akhir
					return n+1 >= min && k(pos)
				}
				return rec(s, next, n+1, k)
			}) {
				return true
			}
		}
		return n >= min && k(pos)
	}
	return func(s *matchState, pos int, k func(int) bool) bool { return rec(s, pos, 0, k) }
}

func equalRunes(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
