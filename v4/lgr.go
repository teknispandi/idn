package v4

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	v3 "github.com/al0fayz/idn/v3"
)

// LGR adalah Label Generation Ruleset RFC 7940 yang sudah dikompilasi.
// Setelah Parse/Load, LGR tidak berubah dan aman dipakai bersamaan.
//
// *LGR mengimplementasikan v3.LabelTable dan v3.VariantTable:
//
//	lgr, _ := v4.LoadFile("lgrs/second-level-reference/und-Bali.xml")
//	manager.RegisterTable(lgr)                         // nama: "und-bali"
//	manager.RegisterTld("id", []string{"latin", "und-bali"})
type LGR struct {
	name string
	meta Meta

	elements map[string]*element   // kunci: string code point elemen
	byFirst  map[rune][]*element   // elemen per code point pertama, terpanjang dulu
	tags     map[string][]rune     // tag → code point tunggal (untuk from-tag)
	index    map[string]string     // elemen/target varian → index himpunan varian
	rules    map[string]*namedRule // aturan bernama
	classes  map[string]*namedSet  // kelas bernama
	actions  []action
}

// Meta adalah metadata LGR (RFC 7940 §4.3).
type Meta struct {
	Version        string
	Date           string
	ValidityStart  string
	ValidityEnd    string
	Languages      []string
	Scopes         []string
	UnicodeVersion string
}

// element adalah satu anggota repertoire: code point atau sekuens.
type element struct {
	cps       []rune
	when      string
	notWhen   string
	vars      []*variant // varian non-refleksif
	reflexive *variant   // varian ke dirinya sendiri (tipe dicatat untuk label asli)
}

type variant struct {
	cps     []rune
	typ     string
	when    string
	notWhen string
}

// action adalah elemen <action> (RFC 7940 §7).
type action struct {
	disp         v3.Disposition
	match        string
	notMatch     string
	anyVariant   map[string]bool
	allVariants  map[string]bool
	onlyVariants map[string]bool
}

// Aksi default RFC 7940 §7.6, dievaluasi setelah aksi dari file.
var defaultActions = []action{
	{disp: v3.DispositionInvalid, anyVariant: map[string]bool{"invalid": true}},
	{disp: v3.DispositionBlocked, anyVariant: map[string]bool{"blocked": true}},
	{disp: v3.DispositionAllocatable, anyVariant: map[string]bool{"allocatable": true}},
	{disp: v3.DispositionActivated, allVariants: map[string]bool{"activated": true}},
	{disp: v3.DispositionValid},
}

// GetName mengimplementasikan v3.LabelTable
func (l *LGR) GetName() string { return l.name }

// Meta mengembalikan metadata LGR
func (l *LGR) Meta() Meta { return l.meta }

// WithName mengembalikan salinan LGR dengan nama lain, mis. agar LGR dan
// tabel .txt untuk aksara yang sama bisa dipasang berdampingan.
func (l *LGR) WithName(name string) *LGR {
	c := *l
	c.name = strings.ToLower(strings.TrimSpace(name))
	return &c
}

// LoadFile membaca LGR dari file. Namanya adalah nama file tanpa ekstensi,
// huruf kecil ("und-Bali.xml" → "und-bali").
func LoadFile(filename string) (*LGR, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseNamed(f, filename)
}

// Load membaca LGR dari fs.FS; penamaan sama dengan LoadFile.
func Load(fsys fs.FS, filename string) (*LGR, error) {
	f, err := fsys.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseNamed(f, filename)
}

func parseNamed(r io.Reader, filename string) (*LGR, error) {
	l, err := Parse(r)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filename, err)
	}
	l.name = strings.ToLower(strings.TrimSuffix(path.Base(filename), path.Ext(filename)))
	return l, nil
}

// Parse membaca LGR XML. Namanya adalah bahasa pertama di metadata (huruf
// kecil); ganti dengan WithName bila perlu.
func Parse(r io.Reader) (*LGR, error) {
	br := bufio.NewReader(r)
	if bom, _ := br.Peek(3); bytes.Equal(bom, []byte("\xef\xbb\xbf")) {
		br.Discard(3)
	}
	var root node
	if err := xml.NewDecoder(br).Decode(&root); err != nil {
		return nil, fmt.Errorf("invalid LGR XML: %w", err)
	}
	if root.XMLName.Local != "lgr" {
		return nil, fmt.Errorf("root element is <%s>, want <lgr>", root.XMLName.Local)
	}

	l := &LGR{
		elements: make(map[string]*element),
		byFirst:  make(map[rune][]*element),
		tags:     make(map[string][]rune),
		rules:    make(map[string]*namedRule),
		classes:  make(map[string]*namedSet),
	}
	for _, section := range root.Children {
		var err error
		switch section.XMLName.Local {
		case "meta":
			l.parseMeta(&section)
			err = checkUnicodeVersion(l.meta.UnicodeVersion)
		case "data":
			err = l.parseData(&section)
		case "rules":
			err = l.parseRules(&section)
		}
		if err != nil {
			return nil, err
		}
	}
	if len(l.elements) == 0 {
		return nil, fmt.Errorf("LGR has no repertoire")
	}
	if err := l.compile(); err != nil {
		return nil, err
	}
	l.buildVariantIndex()
	if len(l.meta.Languages) > 0 {
		l.name = strings.ToLower(l.meta.Languages[0])
	}
	return l, nil
}

func (l *LGR) parseMeta(meta *node) {
	for _, n := range meta.Children {
		text := strings.TrimSpace(n.Text)
		switch n.XMLName.Local {
		case "version":
			l.meta.Version = text
		case "date":
			l.meta.Date = text
		case "validity-start":
			l.meta.ValidityStart = text
		case "validity-end":
			l.meta.ValidityEnd = text
		case "language":
			l.meta.Languages = append(l.meta.Languages, text)
		case "scope":
			l.meta.Scopes = append(l.meta.Scopes, text)
		case "unicode-version":
			l.meta.UnicodeVersion = text
		}
	}
}

func (l *LGR) parseData(data *node) error {
	for i := range data.Children {
		n := &data.Children[i]
		switch n.XMLName.Local {
		case "char":
			cps, err := parseCodePoints(n.attr("cp"))
			if err != nil || len(cps) == 0 {
				return fmt.Errorf("char cp=%q: invalid code point", n.attr("cp"))
			}
			e := &element{cps: cps, when: n.attr("when"), notWhen: n.attr("not-when")}
			for _, v := range n.Children {
				if v.XMLName.Local != "var" {
					continue
				}
				vcps, err := parseCodePoints(v.attr("cp"))
				if err != nil || len(vcps) == 0 {
					return fmt.Errorf("var cp=%q of %q: invalid code point", v.attr("cp"), n.attr("cp"))
				}
				variant := &variant{cps: vcps, typ: v.attr("type"), when: v.attr("when"), notWhen: v.attr("not-when")}
				if string(vcps) == string(cps) {
					e.reflexive = variant
				} else {
					e.vars = append(e.vars, variant)
				}
			}
			if err := l.addElement(e, strings.Fields(n.attr("tag"))); err != nil {
				return err
			}
		case "range":
			first, err1 := parseCodePoint(n.attr("first-cp"))
			last, err2 := parseCodePoint(n.attr("last-cp"))
			if err1 != nil || err2 != nil || last < first {
				return fmt.Errorf("range %q-%q: invalid code points", n.attr("first-cp"), n.attr("last-cp"))
			}
			tags := strings.Fields(n.attr("tag"))
			for r := first; r <= last; r++ {
				e := &element{cps: []rune{r}, when: n.attr("when"), notWhen: n.attr("not-when")}
				if err := l.addElement(e, tags); err != nil {
					return err
				}
			}
		}
	}
	for r := range l.byFirst {
		list := l.byFirst[r]
		sort.SliceStable(list, func(i, j int) bool { return len(list[i].cps) > len(list[j].cps) })
	}
	return nil
}

func (l *LGR) addElement(e *element, tags []string) error {
	key := string(e.cps)
	if _, dup := l.elements[key]; dup {
		return fmt.Errorf("duplicate repertoire element %s", formatCodePoints(e.cps))
	}
	l.elements[key] = e
	l.byFirst[e.cps[0]] = append(l.byFirst[e.cps[0]], e)
	if len(e.cps) == 1 {
		for _, tag := range tags {
			l.tags[tag] = append(l.tags[tag], e.cps[0])
		}
	}
	return nil
}

func (l *LGR) parseRules(rules *node) error {
	for i := range rules.Children {
		n := &rules.Children[i]
		name := n.attr("name")
		switch n.XMLName.Local {
		case "rule":
			if name == "" {
				return fmt.Errorf("top-level <rule> without name")
			}
			l.rules[name] = &namedRule{def: n}
		case "class", "union", "intersection", "difference", "symmetric-difference", "complement":
			if name == "" {
				return fmt.Errorf("top-level <%s> without name", n.XMLName.Local)
			}
			l.classes[name] = &namedSet{def: n}
		case "action":
			a, err := parseAction(n)
			if err != nil {
				return err
			}
			l.actions = append(l.actions, a)
		}
	}
	return nil
}

func parseAction(n *node) (action, error) {
	a := action{
		disp:     v3.Disposition(n.attr("disp")),
		match:    n.attr("match"),
		notMatch: n.attr("not-match"),
	}
	if a.disp == "" {
		return a, fmt.Errorf("<action> without disp")
	}
	typeSet := func(attr string) map[string]bool {
		v, ok := n.attrOK(attr)
		if !ok {
			return nil
		}
		set := make(map[string]bool)
		for _, t := range strings.Fields(v) {
			set[t] = true
		}
		return set
	}
	a.anyVariant = typeSet("any-variant")
	a.allVariants = typeSet("all-variants")
	a.onlyVariants = typeSet("only-variants")
	return a, nil
}

// compile mengompilasi semua kelas dan aturan bernama sekarang, sehingga
// LGR rusak (referensi hilang, siklus, properti tak dikenal) ditolak saat
// dimuat, bukan saat label pertama dievaluasi.
func (l *LGR) compile() error {
	for name := range l.classes {
		if _, err := l.namedClass(name); err != nil {
			return err
		}
	}
	for name := range l.rules {
		if _, err := l.namedPattern(name); err != nil {
			return err
		}
	}
	check := func(ruleName, where string) error {
		if ruleName == "" {
			return nil
		}
		if _, ok := l.rules[ruleName]; !ok {
			return fmt.Errorf("%s refers to undefined rule %q", where, ruleName)
		}
		return nil
	}
	for _, e := range l.elements {
		where := "char " + formatCodePoints(e.cps)
		if err := firstErr(check(e.when, where), check(e.notWhen, where)); err != nil {
			return err
		}
		variants := e.vars
		if e.reflexive != nil {
			variants = append([]*variant{e.reflexive}, e.vars...)
		}
		for _, v := range variants {
			if err := firstErr(check(v.when, where), check(v.notWhen, where)); err != nil {
				return err
			}
		}
	}
	for _, a := range l.actions {
		if err := firstErr(check(a.match, "action"), check(a.notMatch, "action")); err != nil {
			return err
		}
	}
	l.actions = append(l.actions, defaultActions...)
	return nil
}

// buildVariantIndex membentuk himpunan varian (union-find atas semua
// pemetaan varian, tanpa memandang tipe atau kondisi) dan memilih anggota
// terkecil sebagai index (RFC 7940 §8.5). Mengabaikan kondisi when/not-when
// membuat index lebih ketat: label bisa dianggap bentrok walau varian
// bersyaratnya tidak berlaku, tidak pernah sebaliknya.
func (l *LGR) buildVariantIndex() {
	parent := make(map[string]string)
	var find func(k string) string
	find = func(k string) string {
		p, ok := parent[k]
		if !ok || p == k {
			parent[k] = k
			return k
		}
		root := find(p)
		parent[k] = root
		return root
	}
	for key, e := range l.elements {
		for _, v := range e.vars {
			a, b := find(key), find(string(v.cps))
			if a != b {
				if lessRunes([]rune(b), []rune(a)) {
					a, b = b, a
				}
				parent[b] = a
			}
		}
	}
	l.index = make(map[string]string)
	for k := range parent {
		if root := find(k); root != k {
			l.index[k] = root
		}
	}
}

// HasVariants melaporkan apakah LGR mendefinisikan varian
func (l *LGR) HasVariants() bool { return len(l.index) > 0 }

func lessRunes(a, b []rune) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}

func parseCodePoint(s string) (rune, error) {
	v, err := strconv.ParseUint(strings.TrimSpace(s), 16, 32)
	if err != nil || v > 0x10FFFF {
		return 0, fmt.Errorf("invalid code point %q", s)
	}
	return rune(v), nil
}

func parseCodePoints(s string) ([]rune, error) {
	var out []rune
	for _, f := range strings.Fields(s) {
		r, err := parseCodePoint(f)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func formatCodePoints(cps []rune) string {
	parts := make([]string, len(cps))
	for i, r := range cps {
		parts[i] = fmt.Sprintf("U+%04X", r)
	}
	return strings.Join(parts, " ")
}

func firstErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// node adalah elemen XML generik; isi <rules> boleh bersarang bebas.
type node struct {
	XMLName  xml.Name
	Attrs    []xml.Attr `xml:",any,attr"`
	Text     string     `xml:",chardata"`
	Children []node     `xml:",any"`
}

func (n *node) attrOK(name string) (string, bool) {
	for _, a := range n.Attrs {
		if a.Name.Local == name {
			return a.Value, true
		}
	}
	return "", false
}

func (n *node) attr(name string) string {
	v, _ := n.attrOK(name)
	return v
}

// VariantElements mengimplementasikan v3.VariantSetTable: setiap elemen
// repertoire dan target varian beserta index himpunan variannya.
func (l *LGR) VariantElements() map[string]string {
	out := make(map[string]string, len(l.elements))
	for key, e := range l.elements {
		out[key] = key
		for _, v := range e.vars {
			out[string(v.cps)] = string(v.cps)
		}
	}
	for key, root := range l.index {
		out[key] = root
	}
	return out
}
