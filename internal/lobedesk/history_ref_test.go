package lobedesk

import (
	"testing"

	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func refTestCommits() []gitrepo.Commit {
	return []gitrepo.Commit{
		{SHA: "c", Parents: []string{"b"}, Subject: "main tip"},
		{SHA: "d", Parents: []string{"b"}, Subject: "side tip"},
		{SHA: "b", Parents: []string{"a"}, Subject: "shared"},
		{SHA: "a", Subject: "root"},
	}
}

func shaList(commits []gitrepo.Commit) string {
	out := ""
	for _, commit := range commits {
		out += commit.SHA
	}
	return out
}

func TestReachableCommitsScopesToRef(t *testing.T) {
	commits := refTestCommits()
	for _, test := range []struct {
		tip  string
		want string
	}{
		{"c", "cba"},
		{"d", "dba"},
		{"a", "a"},
		{"", "cdba"},
		{"missing", "cdba"},
	} {
		if got := shaList(reachableCommits(commits, test.tip)); got != test.want {
			t.Fatalf("tip %q reachable = %q; want %q", test.tip, got, test.want)
		}
	}
}

func TestHistoryRefTipResolvesEachKind(t *testing.T) {
	a := &app{
		branches:       []gitrepo.Branch{{Name: "main", SHA: "c"}},
		remoteBranches: []gitrepo.Branch{{Name: "origin/main", SHA: "d"}},
		tags: []gitrepo.Tag{
			{Name: "v1", TargetType: "commit", TargetSHA: "b"},
			{Name: "tree-tag", TargetType: "tree", TargetSHA: "t"},
		},
	}
	for _, test := range []struct {
		ref  historyRef
		want string
	}{
		{historyRef{Kind: historyRefLocal, Name: "main"}, "c"},
		{historyRef{Kind: historyRefRemote, Name: "origin/main"}, "d"},
		{historyRef{Kind: historyRefTag, Name: "v1"}, "b"},
		{historyRef{Kind: historyRefTag, Name: "tree-tag"}, ""},
		{historyRef{Kind: historyRefLocal, Name: "gone"}, ""},
		{historyRef{}, ""},
	} {
		if got := a.historyRefTip(test.ref); got != test.want {
			t.Fatalf("tip for %+v = %q; want %q", test.ref, got, test.want)
		}
	}
}

func TestHistoryRefChoicesListKindsInOrder(t *testing.T) {
	a := &app{
		locale:         localeChinese,
		status:         gitrepo.Status{Branch: "main"},
		branches:       []gitrepo.Branch{{Name: "main"}, {Name: "feature"}},
		remoteBranches: []gitrepo.Branch{{Name: "origin/main"}},
		tags: []gitrepo.Tag{
			{Name: "v1", TargetType: "commit"},
			{Name: "tree-tag", TargetType: "tree"},
		},
	}
	choices := a.historyRefChoices(theme.Dark())
	want := []struct{ kind, label string }{
		{"", "所有引用"},
		{historyRefLocal, "main"},
		{historyRefLocal, "feature"},
		{historyRefRemote, "origin/main"},
		{historyRefTag, "v1"},
	}
	if len(choices) != len(want) {
		t.Fatalf("selector lists %d refs; want %d (tree tags are not commits)", len(choices), len(want))
	}
	for i, w := range want {
		if choices[i].ref.Kind != w.kind || choices[i].label != w.label {
			t.Errorf("choice %d = %q %q; want %q %q", i, choices[i].ref.Kind, choices[i].label, w.kind, w.label)
		}
	}
	// Only the checked-out branch carries the head marker.
	for i, choice := range choices {
		if wantHead := choice.label == "main"; choice.head != wantHead {
			t.Errorf("choice %d (%q) head = %v; want %v", i, choice.label, choice.head, wantHead)
		}
	}
	palette := lanePaletteOf(theme.Dark())
	for _, choice := range choices[1:] {
		wantColor := laneColor(palette, laneColorIndex(choice.label, len(palette)))
		if choice.color != wantColor {
			t.Errorf("%q color = %v; want its graph lane color %v", choice.label, choice.color, wantColor)
		}
	}
}

func TestHistorySourceUsesPinnedWalk(t *testing.T) {
	a := &app{
		commits:    refTestCommits(),
		branches:   []gitrepo.Branch{{Name: "main", SHA: "c"}},
		historyRef: historyRef{Kind: historyRefLocal, Name: "main"},
	}
	if got := shaList(a.historySource()); got != "cba" {
		t.Fatalf("filtered source = %q; want the loaded commits the ref reaches", got)
	}
	// Pinned: the loaded rows came from a walk at the ref's tip, so filtering
	// the window again would hide commits the walk already chose.
	a.historyPinned = "c"
	if got := shaList(a.historySource()); got != "cdba" {
		t.Fatalf("pinned source = %q; want every loaded commit", got)
	}
}

func TestHistoryRefHeaderIconShowsFilterColor(t *testing.T) {
	tk := theme.Dark()
	a := &app{}
	if _, color := a.historyRefHeaderIcon(tk); color != tk.TextSecondary {
		t.Errorf("all refs header color = %v; want the neutral secondary color", color)
	}
	for _, ref := range []historyRef{
		{Kind: historyRefLocal, Name: "main"},
		{Kind: historyRefRemote, Name: "origin/main"},
		{Kind: historyRefTag, Name: "v1"},
	} {
		a.historyRef = ref
		_, color := a.historyRefHeaderIcon(tk)
		want := laneInk(tk, refLaneColor(lanePaletteOf(tk), ref.Name))
		if color != want {
			t.Errorf("%s header color = %v; want its lane color %v", ref.Name, color, want)
		}
	}
}

func TestHistoryRefKeyRoundTrip(t *testing.T) {
	for _, kind := range []string{historyRefLocal, historyRefRemote, historyRefTag} {
		ref := historyRef{Kind: kind, Name: "refs/with:colon"}
		gotKind, gotName, ok := splitHistoryRefKey(ref.key())
		if !ok || gotKind != kind || gotName != ref.Name {
			t.Fatalf("key %q split = %q %q %v", ref.key(), gotKind, gotName, ok)
		}
	}
	if _, _, ok := splitHistoryRefKey("all"); ok {
		t.Fatal("all key parsed as a ref")
	}
}

func TestCommitRefNamesTheRefItOpens(t *testing.T) {
	a := &app{remotes: []gitrepo.Remote{{Name: "origin"}}}
	for _, test := range []struct {
		ref  string
		kind string
		name string
		ok   bool
	}{
		{"master", historyRefLocal, "master", true},
		{"feature/login", historyRefLocal, "feature/login", true},
		{"origin/master", historyRefRemote, "origin/master", true},
		{"HEAD -> master", historyRefLocal, "master", true},
		{"tag: v1.2.0", historyRefTag, "v1.2.0", true},
		{"HEAD", "", "", false},
		{"", "", "", false},
	} {
		kind, name, ok := a.parseCommitRef(test.ref)
		if kind != test.kind || name != test.name || ok != test.ok {
			t.Errorf("parseCommitRef(%q) = %q %q %v; want %q %q %v",
				test.ref, kind, name, ok, test.kind, test.name, test.ok)
		}
	}
}

func TestHistoryRefDotsMatchLaneInkOnLight(t *testing.T) {
	a := &app{
		locale:         localeChinese,
		branches:       []gitrepo.Branch{{Name: "main"}, {Name: "feature"}},
		remoteBranches: []gitrepo.Branch{{Name: "origin/main"}},
		tags:           []gitrepo.Tag{{Name: "v1", TargetType: "commit"}},
	}
	light := theme.Light()
	choices := a.historyRefChoices(light)
	ground := light.Blue.Alpha(0.14).Over(light.BgElevated)
	for _, choice := range choices[1:] {
		want := laneInk(light, refLaneColor(lanePaletteOf(light), choice.label))
		if choice.color != want {
			t.Errorf("%q dot = %v; want laneInk %v", choice.label, choice.color, want)
		}
		if ratio := contrastRatio(choice.color, ground); ratio < 3.0 {
			t.Errorf("%q dot on the selected row = %.2f; want at least 3:1", choice.label, ratio)
		}
	}
}

// A row keeps its two-line height while either column is visible; hiding
// both columns lets it become the compact single-line row.
func TestHistoryRowHeightFollowsColumns(t *testing.T) {
	before := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(before) })
	for _, scale := range []float32{fontScaleSmall, fontScaleStandard, fontScaleLarge} {
		setUIFontScale(scale)
		a := &app{}
		want := 44 * scale
		if scale == fontScaleSmall {
			want = 41.6
		}
		if got := a.historyRowHeight(); !nearlyEqual(got, want) {
			t.Fatalf("default row height = %v; want %v", got, want)
		}
		a.hideAuthor = true
		if got := a.historyRowHeight(); !nearlyEqual(got, want) {
			t.Fatalf("row height with only the time = %v; want %v", got, want)
		}
		a.hideTime = true
		if got := a.historyRowHeight(); !nearlyEqual(got, 34*scale) {
			t.Fatalf("compact row height = %v; want %v", got, 34*scale)
		}
	}
}
