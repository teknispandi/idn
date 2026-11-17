package v4

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/net/idna"
	"golang.org/x/text/unicode/norm"
)

// SupportedUnicodeVersion adalah versi Unicode tertinggi yang bisa dilayani
// dengan benar: yang terendah di antara tabel Unicode pustaka standar Go
// (kelas gc:), golang.org/x/net/idna (IDNA2008 di v3) dan
// golang.org/x/text (normalisasi; data jt: juga berasal dari x/text).
func SupportedUnicodeVersion() string {
	v := unicode.Version
	for _, other := range []string{idna.UnicodeVersion, norm.Version} {
		if compareVersions(other, v) < 0 {
			v = other
		}
	}
	return v
}

// checkUnicodeVersion memeriksa <unicode-version> LGR. LGR yang divalidasi
// ICANN dengan Unicode lebih baru (rilis 2025–2026 menyatakan 15.1/16.0)
// tetap dimuat selama setiap code point di repertoire dan target variannya
// sudah dikenal runtime. Code point yang belum dikenal tidak punya properti
// (gc, jt, PVALID) yang bisa dievaluasi, jadi LGR seperti itu ditolak.
// LGR tanpa <unicode-version> diterima.
func (l *LGR) checkUnicodeVersion() error {
	required := l.meta.UnicodeVersion
	if required == "" {
		return nil
	}
	if _, err := parseVersion(required); err != nil {
		return fmt.Errorf("invalid <unicode-version> %q", required)
	}
	supported := SupportedUnicodeVersion()
	if compareVersions(required, supported) <= 0 {
		return nil
	}

	unknown := make(map[rune]bool)
	check := func(cps []rune) {
		for _, r := range cps {
			if !isAssigned(r) {
				unknown[r] = true
			}
		}
	}
	for _, e := range l.elements {
		check(e.cps)
		for _, v := range e.vars {
			check(v.cps)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	list := make([]rune, 0, len(unknown))
	for r := range unknown {
		list = append(list, r)
	}
	sort.Slice(list, func(i, j int) bool { return list[i] < list[j] })
	return fmt.Errorf("LGR requires Unicode %s and uses code points unknown to Unicode %s (%s); upgrade Go, golang.org/x/net and golang.org/x/text",
		required, supported, formatCodePoints(list[:min(len(list), 10)]))
}

// unassigned adalah General_Category Cn. Go memasukkannya ke
// unicode.Categories (juga di dalam "C"), jadi "punya kategori" tidak sama
// dengan "sudah ditetapkan".
var unassigned = unicode.Categories["Cn"]

// isAssigned melaporkan apakah r sudah ditetapkan di tabel Unicode Go.
func isAssigned(r rune) bool {
	if unassigned != nil {
		return !unicode.Is(unassigned, r)
	}
	for name, table := range unicode.Categories {
		if name != "C" && unicode.Is(table, r) {
			return true
		}
	}
	return false
}

// compareVersions membandingkan versi "a.b.c"; versi tak terbaca dianggap 0.
func compareVersions(a, b string) int {
	va, _ := parseVersion(a)
	vb, _ := parseVersion(b)
	for i := 0; i < 3; i++ {
		if va[i] != vb[i] {
			if va[i] < vb[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

func parseVersion(s string) ([3]int, error) {
	var v [3]int
	parts := strings.Split(strings.TrimSpace(s), ".")
	if len(parts) == 0 || len(parts) > 3 {
		return v, fmt.Errorf("invalid version %q", s)
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return v, fmt.Errorf("invalid version %q", s)
		}
		v[i] = n
	}
	return v, nil
}
