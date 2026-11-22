package lobedesk

import (
	"context"
	"path/filepath"
	"testing"

	"supergit/internal/gitrepo"

	"github.com/egoist/mygo"
)

func TestRepositoryFileMenuOffersSwitchRepository(t *testing.T) {
	oldMenu := mygo.App.Menu()
	t.Cleanup(func() { mygo.App.SetMenu(oldMenu) })
	a := &app{locale: localeEnglish, window: &mygo.Window{}}
	tabs := newRepositoryTabs(context.Background(), a)
	tabs.installRepositoryMenu()
	for _, item := range mygo.App.Menu().Items() {
		if item.Role != mygo.RoleFileMenu {
			continue
		}
		for _, child := range item.Submenu {
			if child.Label == a.t("quickRepositorySwitchCommand") && child.Click != nil {
				return
			}
		}
	}
	t.Fatal("File has no interactive Switch repository entry")
}

func TestRepositoryFileMenuOffersNewAndClone(t *testing.T) {
	oldMenu := mygo.App.Menu()
	t.Cleanup(func() { mygo.App.SetMenu(oldMenu) })
	cases := []struct {
		name   string
		locale locale
		labels []string
	}{
		{name: "english", locale: localeEnglish, labels: []string{"New Repository", "Clone Repository"}},
		{name: "chinese", locale: localeChinese, labels: []string{"创建仓库", "克隆仓库"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := &app{locale: tc.locale, window: &mygo.Window{}}
			tabs := newRepositoryTabs(t.Context(), a)
			t.Cleanup(tabs.shutdown)
			tabs.installRepositoryMenu()
			want := make(map[string]bool, len(tc.labels))
			for _, label := range tc.labels {
				want[label] = false
			}
			for _, item := range mygo.App.Menu().Items() {
				if item.Role != mygo.RoleFileMenu {
					continue
				}
				for _, child := range item.Submenu {
					if _, ok := want[child.Label]; ok && child.Click != nil && !child.Disabled {
						want[child.Label] = true
					}
				}
			}
			for name, found := range want {
				if !found {
					t.Errorf("File has no enabled %q action", name)
				}
			}
		})
	}
}

func TestRepositoryFileMenuShowsSessionRecents(t *testing.T) {
	oldMenu := mygo.App.Menu()
	t.Cleanup(func() { mygo.App.SetMenu(oldMenu) })
	a := &app{locale: localeEnglish, window: &mygo.Window{}}
	tabs := newRepositoryTabs(context.Background(), a)
	tabs.installRepositoryMenu()
	recent := func() *mygo.MenuItem {
		t.Helper()
		for _, item := range mygo.App.Menu().Items() {
			if item.Role == mygo.RoleFileMenu {
				for _, child := range item.Submenu {
					if child.Label == "Open Recent" {
						return child
					}
				}
			}
		}
		t.Fatal("File has no Open Recent entry")
		return nil
	}
	if item := recent(); item.Disabled || len(item.Submenu) != 1 || item.Submenu[0].Label != a.t("reopenClosedTab") || !item.Submenu[0].Disabled {
		t.Fatal("Open Recent should expose a disabled Reopen Closed Tab in an empty session")
	}
	root := t.TempDir()
	first := filepath.Join(root, "first", "Same Name")
	second := filepath.Join(root, "second", "Same Name")
	a.rememberRepository(first)
	a.repo = &gitrepo.Repository{Root: second}
	a.rememberRepository(second)
	tabs.rememberClosedRepository(first, 0)
	items := recent().Submenu
	if len(items) != 4 || items[0].Label != a.t("reopenClosedTab") || items[0].Disabled || items[0].Click == nil ||
		items[2].Label != second || !items[2].Disabled || items[3].Label != first || items[3].Disabled || items[3].Click == nil {
		t.Fatalf("want Reopen Closed Tab and distinct recent paths, with the current checkout disabled, got %+v", items)
	}
	tabs.forgetClosedRepository(first)
	if !recent().Submenu[0].Disabled {
		t.Fatal("Reopen Closed Tab stayed enabled after consuming the last closed tab")
	}
}
