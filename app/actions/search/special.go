package search

import "strings"

var tayloredResults = map[string]string{
	"silvano": "silvano trotta",
}

func checkForSpecialHandling(s string) string {
	sLower := strings.ToLower(s)
	if newSearch, ok := tayloredResults[sLower]; ok {
		return newSearch
	}
	return s
}
