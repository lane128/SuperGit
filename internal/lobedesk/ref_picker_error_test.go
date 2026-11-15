package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func refPickerTester(a *app) *ui.Tester {
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(600).Height(500).Children(func() {
			a.historyRefMenu(c)
		})
	}, 600, 500)
	anchor, ok := tt.Find(a.t("allRefs"))
	if !ok {
		return tt
	}
	tt.ClickAt(anchor.X+anchor.W/2, anchor.Y+anchor.H/2)
	tt.Frame()
	return tt
}

// A failed ref read used to leave the picker looking like a repository with
// nothing in it. The error and a retry have to show, and a query that matches
// nothing must not blame itself for the failure.
func TestRefPickerShowsALoadFailure(t *testing.T) {
	a := &app{locale: localeChinese, branchesErr: "fatal: cannot read refs"}
	tt := refPickerTester(a)
	if !a.historyRefOpen {
		t.Fatal("the ref picker did not open")
	}
	if _, ok := tt.Find("fatal: cannot read refs"); !ok {
		t.Error("the picker hides the error that left the list short")
	}
	if _, ok := tt.Find(a.t("retry")); !ok {
		t.Error("the picker offers no retry")
	}
	a.historyRefQuery = "zzz"
	tt.Frame()
	if _, ok := tt.Find(a.t("refsUnavailable")); !ok {
		t.Error("an empty list under a load failure does not say the refs are missing")
	}
	if _, ok := tt.Find(a.t("searchBranchesNoMatch")); ok {
		t.Error("the picker blames the query for a list it could not load")
	}
}

// Without a failure the empty result keeps saying the query matched nothing,
// and no retry is offered for a list that loaded fine.
func TestRefPickerKeepsTheNoMatchTextWhenNothingFailed(t *testing.T) {
	a := &app{locale: localeChinese}
	tt := refPickerTester(a)
	a.historyRefQuery = "zzz"
	tt.Frame()
	if _, ok := tt.Find(a.t("searchBranchesNoMatch")); !ok {
		t.Error("a query that matches nothing does not say so")
	}
	if _, ok := tt.Find(a.t("retry")); ok {
		t.Error("the picker offers a retry although nothing failed")
	}
	if _, ok := tt.Find(a.t("refsUnavailable")); ok {
		t.Error("the picker claims the refs did not load")
	}
}
