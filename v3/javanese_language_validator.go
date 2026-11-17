package v3

// --- Javanese Validator (und-Java) ---
//
// Mengikuti ICANN Second Level Reference LGR for the Javanese Script
// (13 Agustus 2026), lgrs/second-level-reference/und-Java.xml. Seperti
// validator Bali, rule XML-nya yang normatif; setiap pengecekan di bawah
// menyebut nama rule LGR-nya. Kesesuaiannya dijaga oleh
// TestJavaneseLGRMatchesHandwrittenValidator di package v4.
//
// Varian (mis. KA ↔ KA MURDA, layar+NA ↔ layar+NA MURDA) tidak ditangani di
// sini; untuk pemblokiran varian pakai LGR lewat package v4.

type JavaneseLanguageValidator struct{}

const (
	javaneseCecakTelu = 0xA9B3 // hanya sah di dalam aksara rekan
	javanesePangkon   = 0xA9C0 // virama
	javaneseTaling    = 0xA9BA
	javaneseDirgaMure = 0xA9BB
	javaneseTarung    = 0xA9B4
)

// Kelas dari tag LGR.
var (
	// tag "Consonant"
	javaneseConsonants = runeSet(
		0xA98F, 0xA990, 0xA991, 0xA992, 0xA993, 0xA994, 0xA995, 0xA997,
		0xA99A, 0xA99B, 0xA99D, 0xA99F, 0xA9A0, 0xA9A1, 0xA9A2, 0xA9A3,
		0xA9A4, 0xA9A5, 0xA9A6, 0xA9A7, 0xA9A8, 0xA9A9, 0xA9AA, 0xA9AB,
		0xA9AD, 0xA9AE, 0xA9AF, 0xA9B0, 0xA9B1, 0xA9B2,
	)
	// tag "Vowel" (aksara swara)
	javaneseVowels = runeSet(0xA984, 0xA986, 0xA988, 0xA989, 0xA98A, 0xA98C, 0xA98D, 0xA98E)
	// tag "Dependent-Vowel" (sandhangan swara)
	javaneseDependentVowels = runeSet(0xA9B4, 0xA9B6, 0xA9B8, 0xA9BA, 0xA9BB, 0xA9BC)
	// tag "Medial-Consonant" (keret, pengkal, cakra)
	javaneseMedials = runeSet(0xA9BD, 0xA9BE, 0xA9BF)
	// tag "Sign" (cecak, layar, wignyan)
	javaneseSigns = runeSet(0xA981, 0xA982, 0xA983)
	// class "aksara-rekan-base": KA, GA, JA, DA, PA, WA
	javaneseRekanBases = runeSet(0xA98F, 0xA992, 0xA997, 0xA9A2, 0xA9A5, 0xA9AE)
)

func runeSet(runes ...rune) map[rune]bool {
	set := make(map[rune]bool, len(runes))
	for _, r := range runes {
		set[r] = true
	}
	return set
}

// isJavaneseRepertoire: code point tunggal di repertoire LGR (52) + hyphen.
// U+A9B3 tidak termasuk; ia hanya sah sebagai bagian aksara rekan.
func isJavaneseRepertoire(r rune) bool {
	return r == '-' || r == javanesePangkon || javaneseConsonants[r] || javaneseVowels[r] ||
		javaneseDependentVowels[r] || javaneseMedials[r] || javaneseSigns[r]
}

func (j *JavaneseLanguageValidator) IsValidLabelForLanguage(label string) bool {
	return j.CheckLabelForLanguage(label) == nil
}

// CheckLabelForLanguage seperti IsValidLabelForLanguage, tetapi menyebut
// rule LGR yang dilanggar beserta posisi code point-nya.
func (j *JavaneseLanguageValidator) CheckLabelForLanguage(label string) error {
	runes := []rune(label)
	n := len(runes)
	if n == 0 {
		return newLabelError(ErrEmptyLabel, label)
	}
	fail := func(i int, kind error, rule string) error {
		e := newRuneError(kind, label, i, runes[i])
		e.Rule = rule
		return e
	}

	// at returns the rune at i, or -1 outside the label.
	at := func(i int) rune {
		if i < 0 || i >= n {
			return -1
		}
		return runes[i]
	}
	// afterConsonantOrRekan: look-behind "consonant" atau rule "aksara-rekan"
	// (basis + CECAK TELU) tepat sebelum posisi i.
	afterConsonantOrRekan := func(i int) bool {
		prev := at(i - 1)
		return javaneseConsonants[prev] || (prev == javaneseCecakTelu && javaneseRekanBases[at(i-2)])
	}

	for i, r := range runes {
		prev := at(i - 1)

		switch {
		case r == javaneseCecakTelu:
			// Hanya sebagai elemen sekuens aksara rekan.
			if !javaneseRekanBases[prev] {
				return fail(i, ErrCodepointNotInTable, "aksara-rekan")
			}

		case !isJavaneseRepertoire(r):
			return fail(i, ErrCodepointNotInTable, "repertoire")

		case r == '-':
			// hyphen-minus-disallowed (RFC 5891 §4.2.3.1), dihitung per rune.
			if i == 0 || i == n-1 || (i == 3 && prev == '-') {
				return fail(i, ErrLanguageRule, "hyphen-minus-disallowed")
			}

		case javaneseSigns[r]:
			if !afterConsonantOrRekan(i) && !javaneseMedials[prev] && !javaneseVowels[prev] && !javaneseDependentVowels[prev] {
				return fail(i, ErrLanguageRule, "follows-cons-aksara-rekan-med-vowel-dep")
			}

		case r == javaneseTarung:
			if prev != javaneseTaling && prev != javaneseDirgaMure {
				return fail(i, ErrLanguageRule, "follows-A9BA-or-A9BB")
			}

		case javaneseDependentVowels[r]:
			if !afterConsonantOrRekan(i) && !javaneseVowels[prev] && !javaneseMedials[prev] {
				return fail(i, ErrLanguageRule, "follows-cons-aksara-rekan-med-vowel")
			}

		case javaneseMedials[r]:
			if !afterConsonantOrRekan(i) {
				return fail(i, ErrLanguageRule, "follows-cons-aksara-rekan")
			}

		case r == javanesePangkon:
			if !afterConsonantOrRekan(i) && !javaneseVowels[prev] {
				return fail(i, ErrLanguageRule, "follows-cons-aksara-rekan-vowel")
			}
		}
	}

	// Aksi WLE, dievaluasi setelah konteks seperti di LGR.
	// pangkon-disallowed: (K|V) ꧀ (K|V) ꧀ (K|V) di mana pun — mencegah
	// tumpukan pasangan dua tingkat.
	isBase := func(r rune) bool { return javaneseConsonants[r] || javaneseVowels[r] }
	for i := 0; i+4 < n; i++ {
		if isBase(runes[i]) && runes[i+1] == javanesePangkon && isBase(runes[i+2]) &&
			runes[i+3] == javanesePangkon && isBase(runes[i+4]) {
			return fail(i+3, ErrLanguageRule, "pangkon-disallowed")
		}
	}

	return nil
}
