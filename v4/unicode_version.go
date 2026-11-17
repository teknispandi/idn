package v4

import (
	"fmt"
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

// checkUnicodeVersion menolak LGR yang menuntut Unicode lebih baru daripada
// SupportedUnicodeVersion: properti dan status PVALID code point yang baru
// ditambahkan akan salah tanpa ketahuan. LGR tanpa <unicode-version> diterima.
func checkUnicodeVersion(required string) error {
	if required == "" {
		return nil
	}
	if _, err := parseVersion(required); err != nil {
		return fmt.Errorf("invalid <unicode-version> %q", required)
	}
	if supported := SupportedUnicodeVersion(); compareVersions(required, supported) > 0 {
		return fmt.Errorf("LGR requires Unicode %s, but only %s is available; upgrade Go, golang.org/x/net and golang.org/x/text", required, supported)
	}
	return nil
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
