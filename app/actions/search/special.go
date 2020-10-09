package search

import "strings"

var tayloredResults = map[string]string{
	"silvano":                "@SilvanoTrotta",
	"trotta":                 "@SilvanoTrotta",
	"silvano trotta":         "@SilvanoTrotta",
	"corbett":                "@CorbettReport",
	"linux gamer":            "thelinuxgamer",
	"linuxgamer":             "thelinuxgamer",
	"tim pool":               "timcast",
	"jordan peterson":        "jordanbpeterson",
	"quartering":             "thequartering",
	"bombards":               "Bombards_Body_Language",
	"bombard body language":  "Bombards_Body_Language",
	"bombards body language": "Bombards_Body_Language",
	"stefan molyneux":        "@freedomain",
}

func checkForSpecialHandling(s string) string {
	sLower := strings.ToLower(s)
	if newSearch, ok := tayloredResults[sLower]; ok {
		return newSearch
	}
	return s
}
