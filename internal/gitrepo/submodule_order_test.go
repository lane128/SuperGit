package gitrepo

import (
	"context"
	"slices"
	"strings"
	"testing"
)

// The list used to follow .gitmodules, which keeps the order modules were
// added, so a repository whose config was edited over time showed its
// sidebar and Workspaces rows in an arbitrary order. Sorting in the loader
// gives every caller the same predictable name order.
func TestSubmodulesListInNameOrder(t *testing.T) {
	child := fixture(t)
	seed(t, child)
	parent := fixture(t)
	seed(t, parent)
	added := []struct{ name, path string }{
		{"zeta", "vendor/zeta"},
		{"alpha", "libs/alpha"},
		{"Mid", "libs/Mid"},
	}
	for _, module := range added {
		git(t, parent, "-c", "protocol.file.allow=always", "submodule", "add", "--name", module.name, child, module.path)
	}
	// The fixture must start out of order, or the assertion below would pass
	// even without the sort.
	raw := git(t, parent, "config", "--file", ".gitmodules", "--get-regexp", `^submodule\..*\.path$`)
	if !strings.HasPrefix(raw, "submodule.zeta.path") {
		t.Fatalf(".gitmodules did not keep its insertion order:\n%s", raw)
	}
	mods, err := openTest(t, parent).Submodules(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, m := range mods {
		got = append(got, m.Name)
	}
	if want := []string{"alpha", "Mid", "zeta"}; !slices.Equal(got, want) {
		t.Fatalf("submodules listed as %q; want name order %q", got, want)
	}
}
