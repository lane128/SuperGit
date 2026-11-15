package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// The checkout picker paints its keyboard highlight as a background tint, and
// a tint says nothing to a screen reader. The rows have to carry the list
// semantics too: an option, its place among the choices, and whether it is the
// one the arrows are on.
func TestCheckoutPickerExposesOptionsToAssistiveTechnology(t *testing.T) {
	a := &app{
		locale: localeChinese,
		branches: []gitrepo.Branch{
			{Name: "main", SHA: "aaaa", Current: true},
			{Name: "feat", SHA: "aaaa"},
			{Name: "fix", SHA: "aaaa"},
		},
	}
	a.openCheckoutBranchDialog(gitrepo.Commit{SHA: "aaaa", Subject: "pick one"})
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(520).Height(600).Children(func() {
			a.checkoutBranchDialog(c)
		})
	}, 520, 600)

	option := func(name string) ui.AccessInfo {
		t.Helper()
		info, ok := tt.AccessByRole(ui.RoleOption, name)
		if !ok {
			t.Fatalf("%q is not an option of the list", name)
		}
		return info
	}
	if main := option("main"); main.PosInSet != 1 || main.SetSize != 3 {
		t.Errorf("main is %d of %d; want 1 of 3", main.PosInSet, main.SetSize)
	}
	// The current branch is dimmed and not a choice: it has to say so, or a
	// screen reader offers an option that does nothing when activated.
	if main := option("main"); main.Selected || !main.Disabled || main.Description != a.t("current") {
		t.Errorf("main is %+v; want disabled and explained as the current branch", main)
	}
	if feat := option("feat"); feat.PosInSet != 2 || !feat.Selectable || feat.Selected || !feat.Focused {
		t.Errorf("feat is %+v; want the second of three, selectable, and the active descendant", feat)
	}

	tt.Key(0, ui.KeyDown)
	tt.Frame()
	if feat := option("feat"); feat.Focused {
		t.Error("the active descendant stayed on feat after Down")
	}
	if fix := option("fix"); fix.PosInSet != 3 || !fix.Focused {
		t.Errorf("fix is %+v; want the third of three and the active descendant", fix)
	}

	foundList := false
	for _, info := range tt.AccessAll() {
		if info.Role == ui.RoleList {
			foundList = true
			break
		}
	}
	if !foundList {
		t.Error("the rows have no list around them")
	}
}

// The ref picker has the same highlight and the same gap. Its first row is not
// a ref (it clears the filter), so it is still an option of the set.
func TestRefPickerExposesOptionsToAssistiveTechnology(t *testing.T) {
	a := &app{
		locale: localeChinese,
		branches: []gitrepo.Branch{
			{Name: "feat", SHA: "aaaa"},
			{Name: "fix", SHA: "aaaa"},
		},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(600).Height(500).Children(func() {
			a.historyRefMenu(c)
		})
	}, 600, 500)
	anchor, ok := tt.Find(a.t("allRefs"))
	if !ok {
		t.Fatal("the ref anchor is not laid out")
	}
	tt.ClickAt(anchor.X+anchor.W/2, anchor.Y+anchor.H/2)
	tt.Frame()

	if all, ok := tt.AccessByRole(ui.RoleOption, a.t("allRefs")); !ok {
		t.Fatal("the all-refs row is not an option of the list")
	} else if all.PosInSet != 1 || all.SetSize != 3 || !all.Selected || !all.Focused {
		t.Errorf("all-refs is %+v; want the first of three, the ref in use and the active descendant", all)
	}
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	// The row in use keeps Selected; only the active descendant moves, so a
	// screen reader never hears two options chosen.
	if all, _ := tt.AccessByRole(ui.RoleOption, a.t("allRefs")); all.Focused {
		t.Errorf("all-refs is still the active descendant after Down: %+v", all)
	}
	if feat, ok := tt.AccessByRole(ui.RoleOption, "feat"); !ok {
		t.Fatal("feat is not an option of the list")
	} else if feat.PosInSet != 2 || feat.SetSize != 3 || feat.Selected || !feat.Focused {
		t.Errorf("feat is %+v; want the second of three, not selected, and the active descendant", feat)
	}
}
