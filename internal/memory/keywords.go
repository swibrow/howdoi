package memory

import (
	"sort"
	"strings"
	"unicode"
)

var stopWords = map[string]bool{
	"how": true, "do": true, "i": true, "to": true, "the": true,
	"a": true, "an": true, "in": true, "on": true, "for": true,
	"is": true, "it": true, "of": true, "and": true, "or": true,
	"with": true, "from": true, "by": true, "at": true, "as": true,
	"this": true, "that": true, "what": true, "which": true, "where": true,
	"when": true, "who": true, "why": true, "can": true, "will": true,
	"my": true, "me": true, "all": true, "if": true, "not": true,
	"but": true, "so": true, "up": true, "out": true, "about": true,
	"into": true, "just": true, "get": true, "make": true, "use": true,
}

func extractKeywords(question string) []string {
	lower := strings.ToLower(question)

	// Split on non-alphanumeric characters
	words := strings.FieldsFunc(lower, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	seen := make(map[string]bool)
	var keywords []string
	for _, w := range words {
		if len(w) < 2 || stopWords[w] || seen[w] {
			continue
		}
		seen[w] = true
		keywords = append(keywords, w)
	}

	sort.Strings(keywords)
	return keywords
}
