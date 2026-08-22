package parser

import (
	"regexp"
	"strings"
	"unicode"
)

var wsRE = regexp.MustCompile(`\s+`)
var currencyRE = regexp.MustCompile(`[¥$€£]\s*([0-9]+(?:\.[0-9]+)?)`)

func NormalizeText(s string) string {
	s = strings.TrimSpace(s)
	s = wsRE.ReplaceAllString(s, " ")
	return s
}

func ParsePrice(s string) (amount string, currency string) {
	s = NormalizeText(s)
	if m := currencyRE.FindStringSubmatch(s); len(m) == 2 {
		cur := "CNY"
		switch {
		case strings.Contains(s, "$"):
			cur = "USD"
		case strings.Contains(s, "€"):
			cur = "EUR"
		case strings.Contains(s, "£"):
			cur = "GBP"
		}
		return m[1], cur
	}
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) || r == '.' {
			b.WriteRune(r)
		}
	}
	return b.String(), ""
}

func FirstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return NormalizeText(v)
		}
	}
	return ""
}
