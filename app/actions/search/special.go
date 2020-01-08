package search

import "strings"

var tayloredResults = map[string]string{
	"porn": "porn sex anal oral creampie",
}

func checkForSpecialHandling(s string) string {
	sLower := strings.ToLower(s)
	if newSearch, ok := tayloredResults[sLower]; ok {
		return newSearch
	}
	return s
}
