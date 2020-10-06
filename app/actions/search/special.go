package search

import "strings"

var tayloredResults = map[string]string{
	"silvano":         "@SilvanoTrotta",
	"trotta":          "@SilvanoTrotta",
	"silvano trotta":  "@SilvanoTrotta",
	"linux gamer":     "@thelinuxgamer",
	"linuxgamer":      "@thelinuxgamer",
	"tim pool":        "@timcast",
	"jordan peterson": "@jordanbpeterson",
	"quartering":      "@thequartering",
	"bombards":        "@Bombards_Body_Language",
	"body language":   "@Bombards_Body_Language",
}

func checkForSpecialHandling(s string) string {
	sLower := strings.ToLower(s)
	if newSearch, ok := tayloredResults[sLower]; ok {
		return newSearch
	}
	return s
}
