package lobedesk

import "strings"

// commandMatchScore ranks contiguous text above ordered abbreviations. Scores
// are deterministic; callers retain catalog order for ties. Each word must
// match one complete name/alias, never the join between two different aliases.
func commandMatchScore(query string, names ...string) int {
	score := 0
	for _, word := range strings.Fields(strings.ToLower(query)) {
		best := -1
		for _, name := range names {
			name = strings.ToLower(name)
			if at := strings.Index(name, word); at >= 0 {
				best = max(best, 10000-at)
				continue
			}
			text, needle := []rune(name), []rune(word)
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
					best = max(best, max(0, 1000-(end-start+1-len(needle))*8-start))
				}
			}
		}
		if best < 0 {
			return -1
		}
		score += best
	}
	return score
}
