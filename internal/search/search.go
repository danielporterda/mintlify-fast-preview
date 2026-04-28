package search

import (
	"sort"
	"strings"
	"unicode"
)

type Document struct {
	Route string
	Title string
	Body  string
}

type Result struct {
	Route string
	Title string
	Score int
}

type Index struct {
	docs []Document
}

func NewIndex(docs []Document) *Index {
	copied := append([]Document(nil), docs...)
	return &Index{docs: copied}
}

func (i *Index) Search(query string) []Result {
	terms := tokenize(query)
	var results []Result
	for _, doc := range i.docs {
		score := 0
		titleTokens := tokenize(doc.Title)
		bodyTokens := tokenize(doc.Body)
		for _, term := range terms {
			score += 3 * count(titleTokens, term)
			score += count(bodyTokens, term)
		}
		if score > 0 {
			results = append(results, Result{Route: doc.Route, Title: doc.Title, Score: score})
		}
	}
	sort.SliceStable(results, func(a, b int) bool {
		return results[a].Score > results[b].Score
	})
	return results
}

func tokenize(input string) []string {
	return strings.FieldsFunc(strings.ToLower(input), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func count(tokens []string, term string) int {
	n := 0
	for _, token := range tokens {
		if token == term {
			n++
		}
	}
	return n
}
