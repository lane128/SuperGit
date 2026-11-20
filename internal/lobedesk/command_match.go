package lobedesk

import (
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// Each query word may match the title, a translation or the command ID.
// Consecutive letters rank above abbreviations. Equal scores keep catalog order.
func commandMatchScore(query string, names ...string) int {
	score := 0
	for _, word := range strings.Fields(strings.ToLower(query)) {
		best := -1
		for _, name := range names {
			match, _, _ := commandWordMatch(word, strings.ToLower(name))
			best = max(best, match)
		}
		if best < 0 {
			return -1
		}
		score += best
	}
	return score
}

// Return the rune interval that won the ranking so the painted letters agree
// with the search result. word is a nonempty, lowercase query token.
func commandWordMatch(word, name string) (score, from, to int) {
	if at := strings.Index(name, word); at >= 0 {
		from := utf8.RuneCountInString(name[:at])
		return 10000 - at, from, from + utf8.RuneCountInString(word)
	}
	text, needle := []rune(name), []rune(word)
	score = -1
	for start := range text {
		if text[start] != needle[0] {
			continue
		}
		next, end := 1, start
		for i := start + 1; i < len(text) && next < len(needle); i++ {
			if text[i] == needle[next] {
				next++
				end = i
			}
		}
		if next == len(needle) {
			value := max(0, 1000-(end-start+1-len(needle))*8-start)
			if value > score {
				score, from, to = value, start, end+1
			}
		}
	}
	return
}

func commandHighlightSpans(query, label string) []ui.Span {
	if strings.TrimSpace(query) == "" {
		return nil
	}
	lower := strings.ToLower(label)
	letters := []rune(lower)
	marked := make([]bool, len(letters))
	found := false
	for _, word := range strings.Fields(strings.ToLower(query)) {
		score, from, to := commandWordMatch(word, lower)
		if score < 0 {
			continue
		}
		needle := []rune(word)
		for i := from; i < to && len(needle) > 0; i++ {
			if letters[i] == needle[0] {
				marked[i], found = true, true
				needle = needle[1:]
			}
		}
	}
	if !found {
		return nil
	}
	original := []rune(label)
	var spans []ui.Span
	for start := 0; start < len(marked); {
		end := start + 1
		for end < len(marked) && marked[end] == marked[start] {
			end++
		}
		span := ui.Span{Text: string(original[start:end])}
		if marked[start] {
			span.Weight = 700
		}
		spans = append(spans, span)
		start = end
	}
	return spans
}
