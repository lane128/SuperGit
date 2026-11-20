package lobedesk

import (
	"maps"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestCompactHistorySelectionKeepsRefBadgesSteady(t *testing.T) {
	oldScale := uiFontScaleValue()
	setUIFontScale(fontScaleStandard)
	t.Cleanup(func() { setUIFontScale(oldScale) })
	commits := manyLaneHistory(14)
	commits[0].Subject, commits[0].Refs = "Check in", "HEAD -> main, branch-alpha-very-long, tag: alpha"
	commits[1].Subject = "Another commit"
	a := &app{page: "history", locale: localeEnglish, showGraph: true, commits: commits,
		hideTime: true, hideAuthor: true, historyDetailsHidden: true}
	view := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.historySplit.rowWidth, _ = c.Size()
		a.commitList(c)
	}, 367, 420)
	settle := func() {
		for range 3 {
			view.Frame()
		}
	}
	click := func(subject string) {
		t.Helper()
		row, ok := view.AccessByRole(ui.RoleOption, subject)
		if !ok || row.Offscreen {
			t.Fatalf("missing commit row %q", subject)
		}
		view.ClickAt(row.Bounds.X+historyRowPadding+a.historyGraph.Width+historyRowGraphGap+20, row.Bounds.Y+row.Bounds.H/2)
		settle()
	}
	badges := func() map[string]ui.Rect {
		row, _ := view.AccessByRole(ui.RoleOption, commits[0].Subject)
		found := make(map[string]ui.Rect)
		for _, node := range view.AccessAll() {
			if node.Role == ui.RoleText && !node.Offscreen && node.Bounds.Y >= row.Bounds.Y && node.Bounds.Y+node.Bounds.H <= row.Bounds.Y+row.Bounds.H {
				found[node.Name] = node.Bounds
			}
		}
		return found
	}
	for _, width := range []int{367, 580, 700, 367} {
		view.SetSize(width, 420)
		settle()
		click(commits[1].Subject)
		before := badges()
		graphWidth, graphOffset := a.historyGraph.Width, a.historyGraphScroll.X
		click(commits[0].Subject)
		after := badges()
		if a.selectedSHA != commits[0].SHA || a.historyGraph.Width != graphWidth || a.historyGraphScroll.X != graphOffset {
			t.Fatalf("width %d: selection or graph geometry changed unexpectedly", width)
		}
		if before["main"].W == 0 || !maps.Equal(before, after) {
			t.Fatalf("width %d: selection shifted or folded ref badges: %v -> %v", width, before, after)
		}
	}
}

func TestCompactHistoryKeepsSubjectAndRefOverflowVisibleAcrossResize(t *testing.T) {
	commit := gitrepo.Commit{
		SHA: "a", Subject: "Validate deployment configuration before merging",
		Refs: "HEAD -> main, feature/wide-release-candidate-one, feature/wide-release-candidate-two, feature/wide-release-candidate-three, feature/wide-release-candidate-four, feature/wide-release-candidate-five",
	}
	a := &app{showGraph: true, hideAuthor: true, hideTime: true, selectedSHA: commit.SHA, historyDetailsHidden: true}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		width, _ := c.Size()
		a.historySplit.rowWidth = width
		a.commitRow(c, commit, 0, 0, 1, &commitGraph{Width: width * 0.26})
	}, 479, 100)
	var fullBranchWidth float32
	for _, width := range []int{720, 479, 480, 720, 480, 295} {
		tt.SetSize(width, 100)
		tt.Frame()
		row, ok := tt.AccessByRole(ui.RoleOption, commit.Subject)
		if !ok || !row.Selected {
			t.Fatalf("%d points: selected commit row missing", width)
		}
		main, ok := tt.AccessByRole(ui.RoleText, "main")
		if width == 720 {
			fullBranchWidth = main.Bounds.W
		}
		if !ok || fullBranchWidth <= 0 || main.Bounds.W < fullBranchWidth-0.5 || main.Bounds.X < float32(width)*0.26+85 {
			t.Errorf("%d points: badges crowd out the title or clip the current branch (full width %.2f): %+v", width, fullBranchWidth, main.Bounds)
		}
		visible := false
		for _, label := range []string{"+3", "+4", "+5"} {
			if ref, ok := tt.AccessByRole(ui.RoleText, label); ok {
				visible = ref.Bounds.W >= 13 && ref.Bounds.X+ref.Bounds.W < row.Bounds.X+row.Bounds.W
				break
			}
		}
		if !visible {
			t.Errorf("%d points: overflow count is clipped or missing", width)
		}
	}
}

func TestNarrowHistoryKeepsRoomForLongSubjectWithCurrentBranch(t *testing.T) {
	commit := gitrepo.Commit{
		SHA: "a", Subject: "Release with long branch and tag names",
		Author: "Long fixture author", Date: "2026-11-17T12:34:56Z",
		Refs: "HEAD -> main, release/long-branch-name, tag: production, tag: candidate",
	}
	a := &app{locale: localeEnglish, showGraph: true, historyDetailsHidden: true, selectedSHA: commit.SHA}
	const graphWidth = 76
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.historySplit.rowWidth, _ = c.Size()
		ui.Column(c).FillWidth().PaddingX(historyListBorder + historyListPadding).Children(func() {
			a.commitRow(c, commit, 0, 0, 1, &commitGraph{Width: graphWidth})
		})
	}, 260, 160)
	fullDate := formatDateTime(commit.Date)
	for _, tc := range []struct {
		width int
		date  string
	}{{260, fullDate[5:16]}, {295, fullDate}, {309, fullDate}, {720, fullDate}, {260, fullDate[5:16]}} {
		width := tc.width
		tt.SetSize(width, 160)
		tt.Frame()
		date, hasDate := tt.AccessByRole(ui.RoleText, tc.date)
		row, hasRow := tt.AccessByRole(ui.RoleOption, commit.Subject)
		if !hasDate || !hasRow || !row.Selected {
			t.Fatalf("width %d: selected subject or timestamp missing", width)
		}
		for _, label := range []string{"main", "+1", "+2", "+3", "+4"} {
			if ref, ok := tt.AccessByRole(ui.RoleText, label); ok && ref.Bounds.Y+ref.Bounds.H/2 < date.Bounds.Y+date.Bounds.H/2-5 {
				subjectStart := row.Bounds.X + historyRowPadding + graphWidth + historyRowGraphGap
				want := float32(133)
				if width <= 309 {
					want = 125 // Even the compact overflow may have to stay on the title line.
				}
				if room := ref.Bounds.X - subjectStart; room < want {
					t.Errorf("width %d: %s leaves only %.1f points before its badge", width, label, room)
				}
			}
		}
	}
	for _, label := range []string{"+3", "+4"} {
		if ref, ok := tt.AccessByRole(ui.RoleText, label); ok && !ref.Offscreen {
			tt.RightClickAt(ref.Bounds.X+ref.Bounds.W/2, ref.Bounds.Y+ref.Bounds.H/2)
			if !tt.HasText("release/long-branch-name") {
				t.Fatal("narrow row overflow did not expose the folded branch")
			}
			return
		}
	}
	t.Fatal("narrow row lost the refs overflow control")
}

func TestNarrowHistoryKeepsBranchOverflowOffTheSubjectLine(t *testing.T) {
	commit := gitrepo.Commit{
		SHA: "a", Subject: "Parallel lane 01: validate branch topology",
		Author: "Fixture author", Date: "2026-11-17T12:34:56Z", Refs: "lane-01",
	}
	a := &app{locale: localeEnglish, showGraph: true, historyDetailsHidden: true, selectedSHA: commit.SHA}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		width, _ := c.Size()
		a.historySplit.rowWidth = width
		ui.Column(c).FillWidth().PaddingX(historyListBorder + historyListPadding).Children(func() {
			a.commitRow(c, commit, 0, 0, 1, &commitGraph{Width: width * 0.26})
		})
	}, 260, 100)
	fullDate := formatDateTime(commit.Date)
	for _, tc := range []struct {
		width int
		ref   string
		date  string
	}{{260, "+1", fullDate[5:16]}, {720, "lane-01", fullDate}, {260, "+1", fullDate[5:16]}} {
		tt.SetSize(tc.width, 100)
		tt.Frame()
		stamp, hasStamp := tt.AccessByRole(ui.RoleText, tc.date)
		ref, hasRef := tt.AccessByRole(ui.RoleText, tc.ref)
		if !hasStamp || !hasRef || ref.Offscreen || ref.Bounds.W < 10 ||
			ref.Bounds.Y+ref.Bounds.H <= stamp.Bounds.Y || stamp.Bounds.Y+stamp.Bounds.H <= ref.Bounds.Y ||
			ref.Bounds.X+ref.Bounds.W > stamp.Bounds.X {
			t.Errorf("width %d: expected %q beside timestamp %q, got ref %+v and time %+v", tc.width, tc.ref, tc.date, ref.Bounds, stamp.Bounds)
		}
	}
}

func TestNarrowHistoryUsesSpareTitleSpaceToKeepAuthorReadable(t *testing.T) {
	commit := gitrepo.Commit{
		SHA: "a", Subject: "Merge topic", Author: "Fixture",
		Date: "2026-11-17T12:34:56Z", Refs: "topic",
	}
	a := &app{showGraph: true, historyDetailsHidden: true, selectedSHA: commit.SHA}
	var authorWidth float32
	view := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.historySplit.rowWidth, _ = c.Size()
		authorWidth, _ = c.MeasureText(0, ui.Span{Text: commit.Author, Size: scaledFont(11.5)})
		ui.Column(c).FillWidth().PaddingX(historyListBorder + historyListPadding).Children(func() {
			a.commitRow(c, commit, 0, 0, 1, &commitGraph{Width: 56})
		})
	}, 260, 100)

	fullDate := formatDateTime(commit.Date)
	compactDate := fullDate[5:16]
	for _, tc := range []struct {
		width int
		label string
		date  string
		title bool
	}{
		{260, "+1", compactDate, false},
		{208, "+1", compactDate, true}, // A short title has room for the folded ref when the author does not.
		{720, "topic", fullDate, false},
		{208, "+1", compactDate, true},
	} {
		view.SetSize(tc.width, 100)
		view.Frame()
		author, authorOK := view.AccessByRole(ui.RoleText, commit.Author)
		date, dateOK := view.AccessByRole(ui.RoleText, tc.date)
		if tc.date == compactDate && date.Description != fullDate {
			t.Errorf("width %d: compact date lost the accessible full timestamp, got %q", tc.width, date.Description)
		}
		ref, refOK := view.AccessByRole(ui.RoleText, tc.label)
		if !authorOK || !dateOK || !refOK || ref.Offscreen || author.Bounds.W < authorWidth-0.5 ||
			(ref.Bounds.Y+ref.Bounds.H <= date.Bounds.Y+3) != tc.title {
			t.Fatalf("width %d: author needs %.1f points and %s title placement must be %v, got author %+v, ref %+v, time %+v",
				tc.width, authorWidth, tc.label, tc.title, author.Bounds, ref.Bounds, date.Bounds)
		}
	}
	ref, _ := view.AccessByRole(ui.RoleText, "+1")
	view.RightClickAt(ref.Bounds.X+ref.Bounds.W/2, ref.Bounds.Y+ref.Bounds.H/2)
	if !view.HasText("topic") {
		t.Fatal("moving the overflow off the metadata line lost its branch menu")
	}
}

func TestHistoryTimestampsStayReadableAndAlignedWithLongRefs(t *testing.T) {
	oldScale := uiFontScaleValue()
	setUIFontScale(fontScaleLarge)
	t.Cleanup(func() { setUIFontScale(oldScale) })
	const timestamp = "2026-11-17T12:34:56Z"
	a := &app{showGraph: true, locale: localeEnglish, historyDetailsHidden: true, selectedSHA: "a"}
	commits := []gitrepo.Commit{
		{SHA: "a", Subject: "Release with references", Author: "Fixture author", Date: timestamp, Refs: "feature/international-release-candidate-verification-rollout, tag: next-release, tag: v1"},
		{SHA: "b", Subject: "Commit without references", Author: "Fixture author", Date: timestamp},
	}
	label := formatDateTime(timestamp)
	var dateWidth float32
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		width, _ := c.Size()
		a.historySplit.rowWidth = width
		visibleDate := label
		if width <= 309 {
			visibleDate = label[5:16]
		}
		dateWidth, _ = c.MeasureText(0, ui.Span{Text: visibleDate, Size: scaledFont(11.5)})
		graph := &commitGraph{Width: width * 0.26}
		ui.Column(c).FillWidth().PaddingX(historyListBorder + historyListPadding).Children(func() {
			for i, commit := range commits {
				a.commitRow(c.Key(commit.SHA), commit, i, 0, len(commits), graph)
			}
		})
	}, 260, 130)
	for _, width := range []int{260, 309, 699, 700, 701, 699, 260} {
		tt.SetSize(width, 130)
		tt.Frame()
		visibleDate := label
		if width <= 309 {
			visibleDate = label[5:16]
			for _, node := range tt.AccessAll() {
				if node.Role == ui.RoleText && node.Name == "Fixture author" && node.Bounds.W < 42 {
					t.Errorf("narrow metadata clipped the author to %.1f points", node.Bounds.W)
				}
			}
		}
		found := 0
		for _, node := range tt.AccessAll() {
			if node.Role != ui.RoleText || node.Name != visibleDate {
				continue
			}
			found++
			right := float32(width - historyListBorder - historyListPadding - historyRowPadding)
			end := node.Bounds.X + node.Bounds.W
			if node.Offscreen || node.Bounds.W < dateWidth-.5 || end < right-.5 || end > right+.5 {
				t.Errorf("width %d: the date must show all %.1f points of text and end at %.1f, got %+v", width, dateWidth, right, node.Bounds)
			}
		}
		if found != len(commits) {
			t.Errorf("width %d: found %d full timestamps, expected %d", width, found, len(commits))
		}
	}
}

func TestHistoryMetadataExpandsRefsAcrossResize(t *testing.T) {
	oldScale := uiFontScaleValue()
	setUIFontScale(fontScaleLarge)
	t.Cleanup(func() { setUIFontScale(oldScale) })
	commit := gitrepo.Commit{
		SHA: "a", Subject: "Verify release metadata", Author: "Fixture author", Date: "2026-11-17T12:34:56Z",
		Refs: "feature-a, HEAD -> main, tag: v2, release",
	}
	a := &app{showGraph: true, locale: localeEnglish, selectedSHA: commit.SHA, historyDetailsHidden: true,
		historyRef: historyRef{Kind: historyRefTag, Name: "v2"}}
	var dateWidth float32
	date := formatDateTime(commit.Date)
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		width, _ := c.Size()
		a.historySplit.rowWidth = width
		dateWidth, _ = c.MeasureText(0, ui.Span{Text: date, Size: scaledFont(11.5)})
		ui.Column(c).FillWidth().PaddingX(historyListBorder + historyListPadding).Children(func() {
			a.commitRow(c, commit, 0, 0, 1, &commitGraph{Width: 76})
		})
	}, 800, 100)
	for _, tc := range []struct {
		width int
		shown []string
	}{
		{800, []string{"feature-a", "main", "v2", "+1"}},
		{510, []string{"main", "v2", "+2"}},
		{500, []string{"v2", "+3"}}, // Same slot limit as 510, with less actual room.
		{310, []string{"+4"}},       // The selected tag remains available in the overflow.
		{800, []string{"feature-a", "main", "v2", "+1"}},
	} {
		tt.SetSize(tc.width, 100)
		tt.Frame()
		row, ok := tt.AccessByRole(ui.RoleOption, commit.Subject)
		if !ok || !row.Selected {
			t.Fatalf("%d points: selected commit row missing", tc.width)
		}
		visibleDate, minimumDateWidth := date, dateWidth
		if tc.width == 310 {
			visibleDate, minimumDateWidth = date[5:16], 70
		}
		stamp, ok := tt.AccessByRole(ui.RoleText, visibleDate)
		right := float32(tc.width - historyListBorder - historyListPadding - historyRowPadding)
		if !ok || stamp.Offscreen || stamp.Bounds.W < minimumDateWidth-.5 || stamp.Bounds.X+stamp.Bounds.W < right-.5 || stamp.Bounds.X+stamp.Bounds.W > right+.5 {
			t.Errorf("%d points: timestamp clipped or unaligned: %+v", tc.width, stamp.Bounds)
		}
		last := row.Bounds.X
		for _, label := range tc.shown {
			ref, ok := tt.AccessByRole(ui.RoleText, label)
			if !ok || ref.Offscreen || ref.Bounds.W < 10 || ref.Bounds.X < last-.5 || ref.Bounds.X+ref.Bounds.W > right+.5 {
				t.Errorf("%d points: expected ordered visible ref %q, got %+v", tc.width, label, ref.Bounds)
			}
			last = ref.Bounds.X + ref.Bounds.W
			if tc.width >= 450 && (ref.Bounds.Y+ref.Bounds.H <= stamp.Bounds.Y || stamp.Bounds.Y+stamp.Bounds.H <= ref.Bounds.Y || last > stamp.Bounds.X) {
				t.Errorf("%d points: %q doesn't fit alongside the timestamp", tc.width, label)
			}
		}
	}
}

func TestHistoryWidensWithoutTruncatingReadableAuthor(t *testing.T) {
	oldScale := uiFontScaleValue()
	setUIFontScale(fontScaleStandard)
	t.Cleanup(func() { setUIFontScale(oldScale) })
	type layout struct {
		width    int
		overflow string
	}
	for _, tc := range []struct {
		author  string
		layouts []layout
	}{
		{"Fixture Author", []layout{{1130, "+5"}, {1131, "+5"}, {1320, "+4"}, {1600, "+3"}, {1760, "+2"}, {1130, "+5"}}},
		{"Alexandria Montgomery-Worthington", []layout{{1651, "+4"}, {1652, "+4"}, {1651, "+4"}}},
	} {
		t.Run(tc.author, func(t *testing.T) {
			commits := manyLaneHistory(14)
			commits[0].Subject = "Release readiness"
			commits[0].Author = tc.author
			commits[0].Date = "2026-11-18T22:55:47Z"
			commits[0].Refs = "HEAD -> main, tag: rc-2026-11, tag: v2, origin/main, release-candidate"
			target := commits[0]
			a := &app{page: "history", locale: localeEnglish, themeMode: "light", repo: &gitrepo.Repository{Root: t.TempDir()},
				commits: commits, selectedSHA: target.SHA, showGraph: true, commitFilesLoaded: true}
			a.status.Head, a.status.Branch = target.SHA, "main"
			view := ui.NewTester(a.root, 1130, 640)
			view.SetTitleBar(ui.TitleBar{Height: 38, Left: 72})
			date := formatDateTime(target.Date)
			var authorWidth, dateWidth float32
			ui.NewTester(func(c *ui.Context) {
				authorWidth, _ = c.MeasureText(0, ui.Span{Text: target.Author, Size: 11.5})
				dateWidth, _ = c.MeasureText(0, ui.Span{Text: date, Size: 11.5})
			}, 300, 50).Frame()

			for _, tc := range tc.layouts {
				view.SetSize(tc.width, 640)
				for range 5 {
					view.Frame()
				}
				author, ok := view.AccessByRole(ui.RoleText, target.Author)
				if !ok || author.Offscreen || author.Bounds.W < authorWidth-0.5 {
					t.Fatalf("width %d: author needs %.1f points, got %.1f", tc.width, authorWidth, author.Bounds.W)
				}
				stamp, ok := view.AccessByRole(ui.RoleText, date)
				if !ok || stamp.Offscreen || stamp.Bounds.W < dateWidth-0.5 {
					t.Fatalf("width %d: timestamp needs %.1f points, got %.1f", tc.width, dateWidth, stamp.Bounds.W)
				}
				ref, ok := view.AccessByRole(ui.RoleText, tc.overflow)
				if !ok || ref.Offscreen || ref.Bounds.X+ref.Bounds.W > stamp.Bounds.X {
					t.Fatalf("width %d: expected %s beside the timestamp, got %+v", tc.width, tc.overflow, ref)
				}
				if a.selectedSHA != target.SHA || a.historyScroll.Y != 0 || a.historyGraphScroll.X != 0 {
					t.Fatalf("width %d: resizing changed selection or scroll position", tc.width)
				}
			}
		})
	}
}

func TestHistoryRefsStayWithMetadataAcrossNarrowResize(t *testing.T) {
	commit := gitrepo.Commit{
		SHA: "a", Subject: "Repair cache invalidation for deployment workers and background queue",
		Author: "Alexandria Montgomery", Date: "2026-11-18T20:34:56Z",
		Refs: "HEAD -> main, origin/main, tag: release-2026-prod, feature/long-migration-rollout, tag: release-2026-staging",
	}
	a := &app{locale: localeEnglish, showGraph: true, historyDetailsHidden: true, selectedSHA: commit.SHA}
	var mainWidth float32
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		width, _ := c.Size()
		a.historySplit.rowWidth = width
		mainWidth, _ = c.MeasureText(0, ui.Span{Text: "main", Size: scaledFont(11)})
		ui.Column(c).FillWidth().PaddingX(historyListBorder + historyListPadding).Children(func() {
			a.commitRow(c, commit, 0, 0, 1, &commitGraph{Width: width * 0.26})
		})
	}, 390, 190)
	for _, width := range []int{390, 398, 400, 402, 420, 390} {
		tt.SetSize(width, 190)
		tt.Frame()
		stamp, ok := tt.AccessByRole(ui.RoleText, formatDateTime(commit.Date))
		if !ok || stamp.Offscreen {
			t.Fatalf("%d points: timestamp is missing", width)
		}
		if main, ok := tt.AccessByRole(ui.RoleText, "main"); ok && main.Bounds.W < mainWidth-0.5 {
			t.Errorf("%d points: current branch has %.1f points for %.1f points of text", width, main.Bounds.W, mainWidth)
		}
		var overflow ui.AccessInfo
		for _, label := range []string{"+2", "+3", "+4", "+5"} {
			if node, ok := tt.AccessByRole(ui.RoleText, label); ok {
				overflow = node
				break
			}
		}
		if overflow.Offscreen || overflow.Bounds.W < 10 || overflow.Bounds.Y+overflow.Bounds.H <= stamp.Bounds.Y ||
			stamp.Bounds.Y+stamp.Bounds.H <= overflow.Bounds.Y || overflow.Bounds.X+overflow.Bounds.W > stamp.Bounds.X {
			t.Errorf("%d points: reference overflow should stay beside the timestamp, got %+v", width, overflow.Bounds)
		}
	}
}

func TestHistoryRefPillsKeepDistinguishingSuffixes(t *testing.T) {
	commits := []gitrepo.Commit{
		{SHA: "a", Subject: "One release", Refs: "HEAD -> feature/international-release-candidate-alpha, tag: release"},
		{SHA: "b", Subject: "Another release", Refs: "feature/international-release-candidate-bravo, tag: candidate"},
	}
	for i := range commits {
		commits[i].Author = "Fixture author"
		commits[i].Date = "2026-11-18T20:34:56Z"
	}
	a := &app{locale: localeEnglish, showGraph: true, historyDetailsHidden: true}
	var labels []ui.AccessInfo
	widths := make(map[string]float32)
	view := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		width, _ := c.Size()
		a.historySplit.rowWidth = width
		ui.Column(c).FillWidth().PaddingX(historyListBorder + historyListPadding).Children(func() {
			for i, commit := range commits {
				a.commitRow(c.Key(commit.SHA), commit, i, 0, len(commits), &commitGraph{Width: width * 0.26})
			}
		})
		for _, node := range labels {
			widths[node.Name], _ = c.MeasureText(0, ui.Span{Text: node.Name, Size: scaledFont(11)})
		}
	}, 500, 145) // Room for the author and a shortened branch, but not the full branch name.
	view.Frame()
	labels = view.AccessAll()
	view.Frame()
	for _, suffix := range []string{"pha", "avo"} {
		found := false
		for _, node := range labels {
			if node.Role != ui.RoleText || !strings.HasSuffix(node.Name, suffix) || !strings.Contains(node.Name, "…") {
				continue
			}
			found = true
			if node.Offscreen || node.Bounds.W < widths[node.Name]-0.5 {
				t.Errorf("ref %q is clipped: %+v", node.Name, node.Bounds)
			}
		}
		if !found {
			t.Errorf("no visible ref ends with %q", suffix)
		}
	}
}

func TestHistoryRefMenuStaysOnTheSameTagAcrossResize(t *testing.T) {
	oldScale := uiFontScaleValue()
	setUIFontScale(fontScaleLarge)
	t.Cleanup(func() { setUIFontScale(oldScale) })
	commit := gitrepo.Commit{
		SHA: "a", Subject: "Release metadata", Author: "Fixture author", Date: "2026-11-17T12:34:56Z",
		Refs: "feature-a, HEAD -> main, tag: v2, release",
	}
	a := &app{showGraph: true, locale: localeEnglish, historyDetailsHidden: true,
		historyRef: historyRef{Kind: historyRefTag, Name: "v2"}}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.historySplit.rowWidth, _ = c.Size()
		ui.Column(c).FillWidth().PaddingX(historyListBorder + historyListPadding).Children(func() {
			a.commitRow(c, commit, 0, 0, 1, &commitGraph{Width: 76})
		})
	}, 450, 220)
	tt.Frame()
	tag, ok := tt.AccessByRole(ui.RoleText, "v2")
	if !ok {
		t.Fatal("tag is missing before resize")
	}
	tt.RightClickAt(tag.Bounds.X+tag.Bounds.W/2, tag.Bounds.Y+tag.Bounds.H/2)
	if !tt.HasText(a.t("copyTagName")) {
		t.Fatal("tag menu did not open")
	}
	tt.SetSize(475, 220)
	tt.Frame()
	if !tt.HasText(a.t("copyTagName")) || tt.HasText(a.t("copyName")) {
		t.Fatal("widening moved the tag menu to a different reference")
	}
}

func TestHistoryRefsShareMetadataLine(t *testing.T) {
	a := &app{showGraph: true, locale: localeEnglish}
	commit := gitrepo.Commit{SHA: "a", Subject: "The title has its own line", Author: "Fixture", Refs: "feature", Date: "2026-11-17T12:00:00+08:00"}
	graph := newCommitGraph("", []gitrepo.Commit{commit}, lanePalette)
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.commitRow(c, commit, 0, 0, 1, graph) }, 310, int(a.historyRowHeight()))
	tt.Frame()
	row, ok := tt.AccessByRole(ui.RoleOption, commit.Subject)
	if !ok {
		t.Fatal("commit row missing")
	}
	ref, ok := tt.AccessByRole(ui.RoleText, "feature")
	if !ok {
		t.Fatal("ref badge missing")
	}
	author, ok := tt.AccessByRole(ui.RoleText, "Fixture")
	if !ok {
		t.Fatal("author missing")
	}
	if ref.Bounds.Y < row.Bounds.Y+row.Bounds.H/2-2 {
		t.Fatal("ref badge competes with subject on first line")
	}
	if ref.Bounds.Y+ref.Bounds.H <= author.Bounds.Y || author.Bounds.Y+author.Bounds.H <= ref.Bounds.Y {
		t.Fatal("author and reference do not share metadata line")
	}
	if ref.Bounds.X+ref.Bounds.W > row.Bounds.X+row.Bounds.W || author.Bounds.W <= 0 {
		t.Fatal("metadata overflow hid author or reference")
	}
}
