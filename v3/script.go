package v3

import (
	"sort"
	"unicode"
)

// scriptRange adalah rentang code point dengan satu nilai Script (UAX #24).
type scriptRange struct {
	lo, hi rune
	name   string
}

// scriptRanges berisi seluruh unicode.Scripts, terurut dan tidak tumpang
// tindih (setiap code point punya tepat satu Script), untuk pencarian biner.
var scriptRanges = buildScriptRanges()

func buildScriptRanges() []scriptRange {
	var out []scriptRange
	add := func(lo, hi, stride rune, name string) {
		if stride == 1 {
			out = append(out, scriptRange{lo, hi, name})
			return
		}
		for r := lo; r <= hi; r += stride {
			out = append(out, scriptRange{r, r, name})
		}
	}
	for name, table := range unicode.Scripts {
		for _, r := range table.R16 {
			add(rune(r.Lo), rune(r.Hi), rune(r.Stride), name)
		}
		for _, r := range table.R32 {
			add(rune(r.Lo), rune(r.Hi), rune(r.Stride), name)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].lo < out[j].lo })
	return out
}

// scriptOf mengembalikan nama script Unicode untuk rune, atau "" jika tidak dikenal
func scriptOf(r rune) string {
	i := sort.Search(len(scriptRanges), func(i int) bool { return scriptRanges[i].hi >= r })
	if i < len(scriptRanges) && scriptRanges[i].lo <= r {
		return scriptRanges[i].name
	}
	return ""
}

// Kombinasi yang lazim dalam satu bahasa (UTS #39, "Highly Restrictive"):
// tidak dianggap campuran aksara.
var scriptCombinations = [][]string{
	{"Han", "Hiragana", "Katakana"}, // Jepang
	{"Han", "Hangul"},               // Korea
	{"Han", "Bopomofo"},             // Tionghoa (Taiwan)
}

// mixedScriptAt mengembalikan posisi code point pertama yang membuat label
// memuat lebih dari satu script, atau -1. Common (digit, tanda hubung) dan
// Inherited (tanda diakritik) tidak dihitung.
func mixedScriptAt(runes []rune) int {
	seen := make(map[string]bool)
	for i, r := range runes {
		name := scriptOf(r)
		if name == "" || name == "Common" || name == "Inherited" || seen[name] {
			continue
		}
		seen[name] = true
		if len(seen) > 1 && !withinOneCombination(seen) {
			return i
		}
	}
	return -1
}

func withinOneCombination(scripts map[string]bool) bool {
	for _, combination := range scriptCombinations {
		covered := 0
		for _, name := range combination {
			if scripts[name] {
				covered++
			}
		}
		if covered == len(scripts) {
			return true
		}
	}
	return false
}
