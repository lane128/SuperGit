package lobedesk

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// Match ordinary selectable-text copying: SDK ranges are rune offsets, and
// adjacent selected paragraphs are separated by one newline.
func messageSelectedText(scope ui.Element) string {
	var out strings.Builder
	for index, part := range scope.SelectedTexts() {
		if index > 0 {
			out.WriteByte('\n')
		}
		text := []rune(part.Text)
		start := min(len(text), max(0, part.Start))
		end := min(len(text), max(start, part.End))
		out.WriteString(string(text[start:end]))
	}
	return out.String()
}
