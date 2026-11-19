package lobedesk

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

var commitMessageURLPattern = regexp.MustCompile("(?i)https?://[^\\s\\p{Z}<>\"`\\x00-\\x1f\\x7f]+")

// Recognize explicit HTTP(S) addresses without changing the message text.
// Preserve ambiguous punctuation as URL data; remove only explicit wrappers.
func commitMessageHTTPURLs(text string) []string {
	var result []string
	seen := map[string]bool{}
	for _, location := range commitMessageURLPattern.FindAllStringIndex(text, -1) {
		var previous rune
		if location[0] > 0 {
			previous, _ = utf8.DecodeLastRuneInString(text[:location[0]])
			if unicode.IsLetter(previous) || unicode.IsNumber(previous) || previous == '_' {
				continue
			}
		}
		raw := text[location[0]:location[1]]
		for _, quote := range [][2]rune{{'\'', '\''}, {'‘', '’'}, {'“', '”'}} {
			candidate := strings.TrimRight(raw, ".,;!。 ，、；！")
			if previous == quote[0] && strings.HasSuffix(candidate, string(quote[1])) {
				raw = strings.TrimSuffix(candidate, string(quote[1]))
			}
		}
		for _, pair := range [][2]string{{"(", ")"}, {"[", "]"}, {"{", "}"}} {
			if string(previous) != pair[0] {
				continue
			}
			// Strip prose punctuation only when it is outside a visible closing
			// wrapper that matches the character immediately before the URL.
			candidate := strings.TrimRight(raw, ".,;!。 ，、；！")
			if strings.HasSuffix(candidate, pair[1]) && strings.Count(candidate, pair[1]) > strings.Count(candidate, pair[0]) {
				raw = strings.TrimSuffix(candidate, pair[1])
			}
		}
		parsed, err := url.Parse(raw)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || seen[raw] {
			continue
		}
		seen[raw] = true
		result = append(result, raw)
	}
	return result
}

type messageTarget struct {
	value, label  string
	commit        bool
	copySelection bool
}

type messageMenuSnapshot struct {
	targets []messageTarget
	origin  messageCommitOrigin
}

// Keep plain selectable text. Inline Link widgets currently consume drags that
// begin on the URL; an explicit context action avoids opening during selection.
func (a *app) commitMessageLinkMenu(c *ui.Context, target ui.Element, text string, scope ui.Element) {
	open := ui.Local(target, "contextMenu", func() bool { return false })
	wasOpen := ui.Local(target, "message-menu-open", func() bool { return false })
	captured := ui.Local(target, "message-targets", func() messageMenuSnapshot { return messageMenuSnapshot{} })
	if !*open {
		snapshot := messageMenuSnapshot{origin: a.messageCommitOrigin()}
		for _, raw := range commitMessageHTTPURLs(text) {
			snapshot.targets = append(snapshot.targets, messageTarget{value: raw, label: raw})
		}
		for _, sha := range commitMessageCommitSHAs(text) {
			snapshot.targets = append(snapshot.targets, messageTarget{value: sha, label: a.t("jumpMessageCommit") + " · " + sha, commit: true})
		}
		// Text without targets keeps the SDK's existing Copy / Select All menu.
		if len(snapshot.targets) > 0 {
			snapshot.targets = append([]messageTarget{{value: messageSelectedText(scope), label: a.t("copySelectedText"), copySelection: true}}, snapshot.targets...)
		}
		*captured = snapshot
	}
	if len(captured.targets) == 0 {
		return
	}
	chosen := messageTargetContextMenu(c, target, open, captured.targets, a.messageCommitReady(captured.origin))
	if *open && !*wasOpen {
		a.messageMenuGeneration++
		captured.origin.menu = a.messageMenuGeneration
	}
	*wasOpen = *open
	index, err := strconv.Atoi(chosen)
	if err != nil || index < 0 || index >= len(captured.targets) {
		return
	}
	selected := captured.targets[index]
	if selected.copySelection {
		if selected.value != "" {
			a.copyText(selected.value)
		}
		return
	}
	if selected.commit {
		a.requestMessageCommit(captured.origin, selected.value)
		return
	}
	c.OpenURLThen(selected.value, func(err error) {
		if err != nil {
			a.queue("error", a.tf("openMessageLinkFailed", err.Error()), "")
			a.invalidate()
		}
	})
}

func messageTargetContextMenu(c *ui.Context, target ui.Element, open *bool, targets []messageTarget, commitReady bool) string {
	tk := theme.TokensOf(c)
	width, height := c.Size()
	menu := ui.ContextMenuBase(c, target, open)
	chosen := ""
	menu.Popup(func(panel ui.Element) {
		panel.Padding(4).Radius(tk.BorderRadius).Background(tk.BgElevated).Transition(motion.Popup()).AttachedOrigin()
		tk.Elevate(panel, theme.BoxShadowSecondary)
		panel.Children(func() {
			ui.Scroll(c).Width(min(float32(520), max(float32(0), width-40))).MaxHeight(max(float32(0), height-40)).Children(func() {
				ui.Column(c).FillWidth().Children(func() {
					for index, entry := range targets {
						if index > 0 && targets[index-1].copySelection {
							ui.Column(c).FillWidth().Height(1).MarginY(4).Background(tk.Border)
						}
						disabled := entry.commit && !commitReady || entry.copySelection && entry.value == ""
						color := tk.Text
						if disabled {
							color = tk.TextDisabled
						}
						item := menu.Item().Label(entry.label).Disabled(disabled).PaddingY(6).PaddingX(12).MinHeight(32).Radius(tk.BorderRadiusSM).AlignSelf(ui.Stretch).FontSize(scaledFont(14)).TextColor(color)
						if item.Highlighted() {
							item.Background(tk.FillTertiary)
						}
						item.Children(func() {
							ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Start).Children(func() {
								icon := lucide.ExternalLink()
								if entry.commit {
									icon = lucide.GitCommitHorizontal()
								}
								if entry.copySelection {
									icon = lucide.Copy()
								}
								ui.Icon(c, icon).FontSize(scaledFont(14)).TextColor(color).Shrink(0)
								ui.Text(c, entry.label).TextColor(color).Grow(1).Basis(0).MinWidth(0)
							})
						})
						if item.Clicked() && !disabled {
							chosen = strconv.Itoa(index)
						}
					}
				})
			})
		})
	})
	return chosen
}
