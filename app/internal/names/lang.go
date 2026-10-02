package names

import (
	"sort"
	"strings"
)

// DefaultLang is used when neither the user nor the instance chooses one.
const DefaultLang = "es_es"

// primary is the main regional variant of each language, used as the first
// fallback for its other variants (es_ar -> es_es).
var primary = map[string]string{
	"es": "es_es", "en": "en_us", "pt": "pt_br", "fr": "fr_fr", "de": "de_de",
	"it": "it_it", "zh": "zh_cn", "ru": "ru_ru", "ja": "ja_jp", "ko": "ko_kr",
}

// NormalizeLang turns "es-AR" or "ES_ar" into "es_ar".
func NormalizeLang(code string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(code), "-", "_"))
}

// LangChain returns the languages to try, most preferred first:
// the requested one, the main variant of its language, the other variants of
// that language that are available, and finally en_us.
func LangChain(requested string, available []string) []string {
	req := NormalizeLang(requested)
	if req == "" {
		req = DefaultLang
	}
	var out []string
	seen := map[string]bool{}
	add := func(c string) {
		if c != "" && !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	add(req)
	family, _, _ := strings.Cut(req, "_")
	add(primary[family])
	var siblings []string
	for _, a := range available {
		a = NormalizeLang(a)
		if strings.HasPrefix(a, family+"_") {
			siblings = append(siblings, a)
		}
	}
	sort.Strings(siblings)
	for _, s := range siblings {
		add(s)
	}
	add("en_us")
	return out
}
