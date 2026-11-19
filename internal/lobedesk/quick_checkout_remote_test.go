package lobedesk

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"image/png"
	"os"
	"path/filepath"
	"supergit/internal/gitrepo"
	"testing"
)

func TestQuickCheckoutRemoteRenderedBounds(t *testing.T) {
	for _, size := range [][2]int{{360, 360}, {900, 640}} {
		for _, language := range []locale{localeEnglish, localeChinese} {
			for _, mode := range []string{"light", "dark"} {
				for _, step := range []string{"", "targets", "name"} {
					name := fmt.Sprintf("%dx%d-%s-%s-%s", size[0], size[1], language, mode, step)
					t.Run(name, func(t *testing.T) {
						a := &app{repo: &gitrepo.Repository{Root: t.TempDir()}, locale: language, themeMode: mode, status: gitrepo.Status{Branch: "main"}, branches: []gitrepo.Branch{{Name: "main", SHA: "a", Current: true}, {Name: "local-topic", SHA: "b", Upstream: "upstream/feature/long-remote-branch-name"}}, remoteBranches: []gitrepo.Branch{{Name: "upstream/feature/long-remote-branch-name", SHA: "b"}}}
						a.openQuickCheckout()
						a.quickCheckout.step = step
						a.quickCheckout.remoteName = "upstream/feature/long-remote-branch-name"
						a.quickCheckout.remoteSHA = "b"
						a.quickCheckout.localName = "new-topic"
						a.quickCheckout.nameFocus = true
						tt := ui.NewTester(func(c *ui.Context) {
							a.applyTheme(c)
							ui.Column(c).Fill().Background(theme.TokensOf(c).BgLayout)
							a.quickCheckoutDialog(c)
						}, size[0], size[1])
						for _, n := range tt.AccessAll() {
							if n.Role != ui.RoleTextField && n.Role != ui.RoleOption && n.Role != ui.RoleDialog && n.Role != ui.RoleButton {
								continue
							}
							vertical := n.Role != ui.RoleOption && (n.Bounds.Y < 16 || n.Bounds.Y+n.Bounds.H > float32(size[1]-16))
							if n.Bounds.X < 16 || n.Bounds.X+n.Bounds.W > float32(size[0]-16) || vertical {
								t.Errorf("control outside gutters: %+v", n)
							}
						}
						if dir := os.Getenv("SUPERGIT_REMOTE_CHECKOUT_CAPTURE_DIR"); dir != "" {
							f, err := os.Create(filepath.Join(dir, name+".png"))
							if err != nil {
								t.Fatal(err)
							}
							err = png.Encode(f, tt.Image())
							closeErr := f.Close()
							if err != nil {
								t.Fatal(err)
							}
							if closeErr != nil {
								t.Fatal(closeErr)
							}
						}
					})
				}
			}
		}
	}
}

func TestQuickCheckoutRemoteTargetReturnAndTab(t *testing.T) {
	for _, scenario := range []string{"tab", "return"} {
		t.Run(scenario, func(t *testing.T) {
			a, _, target := quickCheckoutFixture(t)
			a.remoteBranches = []gitrepo.Branch{{Name: "origin/topic", SHA: target}}
			a.branches = append(a.branches, gitrepo.Branch{Name: "first", SHA: target, Upstream: "origin/topic"})
			tt := quickCheckoutTester(a)
			tt.Key(ui.Cmd, ui.KeyB)
			tt.Frame()
			tt.Type("origin/topic")
			tt.Frame()
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			if scenario == "tab" {
				tt.Key(0, ui.KeyTab)
				tt.Frame()
				back, _ := tt.AccessByRole(ui.RoleButton, a.t("quickCheckoutBack"))
				for attempt := 0; !back.Focused && attempt < 6; attempt++ {
					for _, node := range tt.AccessAll() {
						if node.Focused {
							t.Logf("Tab %d focused %+v", attempt, node)
						}
					}
					tt.Key(0, ui.KeyTab)
					tt.Frame()
					back, _ = tt.AccessByRole(ui.RoleButton, a.t("quickCheckoutBack"))
				}
				if !back.Focused {
					t.Fatal("target search prevents Tab from reaching Back")
				}
			} else {
				tt.Key(0, ui.KeyDown)
				tt.Frame()
				tt.Key(0, ui.KeyEnter)
				tt.Frame()
				tt.Key(0, ui.KeyEscape)
				tt.Frame()
				tt.Key(0, ui.KeyEnter)
				tt.Frame()
				if a.quickCheckout.step != "name" || a.busy {
					t.Fatal("return lost create selection and Enter selected another destination")
				}
			}
		})
	}
}

func TestQuickCheckoutRemoteFormTabAndEscape(t *testing.T) {
	a, _, target := quickCheckoutFixture(t)
	a.remoteBranches = []gitrepo.Branch{{Name: "origin/topic", SHA: target}}
	tt := quickCheckoutTester(a)
	tt.Key(ui.Cmd, ui.KeyB)
	tt.Frame()
	tt.Type("origin/topic")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Key(0, ui.KeyTab)
	tt.Frame()
	input, _ := tt.AccessByRole(ui.RoleTextField, a.t("branchName"))
	if input.Focused {
		t.Fatal("name field stole Tab focus")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if !a.quickCheckout.open || a.quickCheckout.step != "" {
		t.Fatal("Escape from form button must return to search")
	}
}

func TestQuickCheckoutRemoteNameAndReturn(t *testing.T) {
	a, callbacks, target := quickCheckoutFixture(t)
	pullFixtureGit(t, a.repo.Root, "remote", "add", "origin", a.repo.Root)
	pullFixtureGit(t, a.repo.Root, "update-ref", "refs/remotes/origin/topic", target)
	a.remoteBranches = []gitrepo.Branch{{Name: "origin/topic", SHA: target}}
	tt := quickCheckoutTester(a)
	body, _ := tt.AccessByRole(ui.RoleTextField, a.t("commitDescription"))
	tt.ClickAt(body.Bounds.X+10, body.Bounds.Y+10)
	tt.Key(ui.Cmd, ui.KeyB)
	tt.Frame()
	tt.Type("origin/topic")
	tt.Frame()
	if !tt.HasText("origin/topic") {
		t.Fatal("remote candidate absent")
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleTextField, a.t("branchName")); !ok {
		t.Fatal("remote without tracker must request a local name")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if !a.quickCheckout.open || a.quickCheckout.query != "origin/topic" {
		t.Fatal("Back lost query or closed overlay")
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("my-topic")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	waitQuickCheckout(t, a, callbacks)
	if got := pullFixtureGit(t, a.repo.Root, "branch", "--show-current"); got != "my-topic" {
		t.Fatalf("wrong local branch %q", got)
	}
	if got := pullFixtureGit(t, a.repo.Root, "rev-parse", "--abbrev-ref", "@{upstream}"); got != "origin/topic" {
		t.Fatalf("wrong upstream %q", got)
	}
	if a.commitText != "draft\n\nbody" || !a.amend {
		t.Fatal("draft changed")
	}
}

func TestQuickCheckoutRemoteGuardsAndRejection(t *testing.T) {
	for _, scenario := range []string{"name collision", "cached tip", "external tip", "external deleted", "external dirty", "generation", "root", "epoch", "busy", "read error"} {
		t.Run(scenario, func(t *testing.T) {
			a, callbacks, target := quickCheckoutFixture(t)
			pullFixtureGit(t, a.repo.Root, "remote", "add", "origin", a.repo.Root)
			pullFixtureGit(t, a.repo.Root, "update-ref", "refs/remotes/origin/topic", target)
			a.remoteBranches = []gitrepo.Branch{{Name: "origin/topic", SHA: target}}
			tt := quickCheckoutTester(a)
			tt.Key(ui.Cmd, ui.KeyB)
			tt.Frame()
			tt.Type("origin/topic")
			tt.Frame()
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			generation := a.quickCheckout.generation
			a.quickCheckout.localName = "new-topic"
			switch scenario {
			case "name collision":
				a.quickCheckout.localName = "topic"
			case "cached tip":
				a.remoteBranches[0].SHA = "changed"
			case "external tip":
				pullFixtureGit(t, a.repo.Root, "update-ref", "refs/remotes/origin/topic", "HEAD")
			case "external deleted":
				pullFixtureGit(t, a.repo.Root, "update-ref", "-d", "refs/remotes/origin/topic")
			case "external dirty":
				if err := os.WriteFile(filepath.Join(a.repo.Root, "same.go"), []byte("keep external bytes\n"), 0600); err != nil {
					t.Fatal(err)
				}
			case "generation":
				a.quickCheckout = quickCheckoutState{}
				a.openQuickCheckout()
			case "root":
				a.quickCheckout.root = "old-root"
			case "epoch":
				a.loadSeq++
			case "busy":
				a.busy = true
			case "read error":
				a.remoteBranchesErr = "read failed"
			}
			snapshot := func() string {
				config, err := os.ReadFile(filepath.Join(a.repo.Root, ".git/config"))
				if err != nil {
					t.Fatal(err)
				}
				working, err := os.ReadFile(filepath.Join(a.repo.Root, "same.go"))
				if err != nil {
					t.Fatal(err)
				}
				return pullFixtureGit(t, a.repo.Root, "rev-parse", "HEAD") + pullFixtureGit(t, a.repo.Root, "show-ref") + pullFixtureGit(t, a.repo.Root, "ls-files", "--stage") + string(config) + string(working)
			}
			before := snapshot()
			a.submitQuickCheckoutRemote(generation)
			if scenario != "busy" {
				waitQuickCheckout(t, a, callbacks)
			}
			if after := snapshot(); after != before {
				t.Fatal("rejection mutated repository")
			}
			if a.commitText != "draft\n\nbody" || !a.amend {
				t.Fatal("rejection changed draft")
			}
			if scenario == "external tip" || scenario == "external deleted" || scenario == "external dirty" {
				if len(a.notices[toastMain]) == 0 {
					t.Fatal("backend rejection lacks feedback")
				}
			}
		})
	}
}

func TestQuickCheckoutRemoteTrackerExactDestination(t *testing.T) {
	for _, changed := range []string{"", "upstream", "remote tip"} {
		t.Run(changed, func(t *testing.T) {
			a, callbacks, target := quickCheckoutFixture(t)
			pullFixtureGit(t, a.repo.Root, "remote", "add", "origin", a.repo.Root)
			pullFixtureGit(t, a.repo.Root, "update-ref", "refs/remotes/origin/topic", target)
			pullFixtureGit(t, a.repo.Root, "branch", "first", target)
			pullFixtureGit(t, a.repo.Root, "branch", "second", "HEAD")
			for _, name := range []string{"first", "second"} {
				pullFixtureGit(t, a.repo.Root, "branch", "--set-upstream-to=origin/topic", name)
			}
			var err error
			a.branches, err = a.repo.Branches(a.ctx)
			if err != nil {
				t.Fatal(err)
			}
			a.remoteBranches = []gitrepo.Branch{{Name: "origin/topic", SHA: target}}
			tt := quickCheckoutTester(a)
			tt.Key(ui.Cmd, ui.KeyB)
			tt.Frame()
			tt.Type("origin/topic")
			tt.Frame()
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			tt.Key(0, ui.KeyDown)
			tt.Frame()
			if changed == "upstream" {
				pullFixtureGit(t, a.repo.Root, "config", "branch.second.merge", "refs/heads/main")
			} else if changed == "remote tip" {
				pullFixtureGit(t, a.repo.Root, "update-ref", "refs/remotes/origin/topic", "HEAD")
			}
			beforeHead := pullFixtureGit(t, a.repo.Root, "rev-parse", "HEAD")
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			waitQuickCheckout(t, a, callbacks)
			want := "second"
			if changed != "" {
				want = "main"
			}
			if got := pullFixtureGit(t, a.repo.Root, "branch", "--show-current"); got != want {
				t.Fatalf("target %q want %q", got, want)
			}
			if got := pullFixtureGit(t, a.repo.Root, "rev-parse", "HEAD"); got != beforeHead {
				t.Fatal("tracker switch moved destination tip to remote tip")
			}
		})
	}
}

func TestQuickCheckoutRemoteFullIdentity(t *testing.T) {
	a, _, target := quickCheckoutFixture(t)
	a.remoteBranches = []gitrepo.Branch{{Name: "origin/topic", SHA: target}, {Name: "upstream/topic", SHA: target}, {Name: "origin/feature/HEAD", SHA: target}}
	tt := quickCheckoutTester(a)
	tt.Key(ui.Cmd, ui.KeyB)
	tt.Frame()
	for _, name := range []string{"origin/topic", "upstream/topic", "origin/feature/HEAD"} {
		if !tt.HasText(name) {
			t.Fatalf("full remote identity hidden: %s", name)
		}
	}
	tt.Type("upstream/topic")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.quickCheckout.remoteName != "upstream/topic" {
		t.Fatal("same short name redirected remote")
	}
}

func TestQuickCheckoutRemoteCancelRestoresBody(t *testing.T) {
	a, _, target := quickCheckoutFixture(t)
	a.remoteBranches = []gitrepo.Branch{{Name: "origin/topic", SHA: target}}
	tt := quickCheckoutTester(a)
	body, ok := tt.AccessByRole(ui.RoleTextField, a.t("commitDescription"))
	if !ok {
		t.Fatal("Body absent")
	}
	tt.ClickAt(body.Bounds.X+10, body.Bounds.Y+10)
	tt.Key(ui.Cmd, ui.KeyB)
	tt.Frame()
	tt.Type("origin/topic")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	tt.Key(0, ui.KeyEnd)
	tt.Type("X")
	tt.Frame()
	if a.quickCheckout.open || a.commitText != "draft\n\nbodyX" || !a.amend {
		t.Fatalf("cancel lost opener/draft: open=%v draft=%q", a.quickCheckout.open, a.commitText)
	}
}

func TestQuickCheckoutRemoteMouseBackRestoresSearch(t *testing.T) {
 a,_,target:=quickCheckoutFixture(t)
 a.remoteBranches=[]gitrepo.Branch{{Name:"origin/topic",SHA:target}}
 tt:=quickCheckoutTester(a);tt.Key(ui.Cmd,ui.KeyB);tt.Frame();tt.Type("origin/topic");tt.Frame();tt.Key(0,ui.KeyEnter);tt.Frame()
 back,ok:=tt.AccessByRole(ui.RoleButton,a.t("quickCheckoutBack"));if !ok{t.Fatal("Back missing")}
 tt.ClickAt(back.Bounds.X+back.Bounds.W/2,back.Bounds.Y+back.Bounds.H/2);tt.Frame();tt.Frame();tt.Key(0,ui.KeyEnd);tt.Type("X");tt.Frame()
 if a.quickCheckout.query!="origin/topicX" {t.Fatalf("mouse Back lost search input ownership: %q",a.quickCheckout.query)}
}

func TestQuickCheckoutRemoteExplicitTrackers(t *testing.T) {
	a, _, target := quickCheckoutFixture(t)
	a.remoteBranches = []gitrepo.Branch{{Name: "origin/topic", SHA: target}, {Name: "upstream/topic", SHA: target}}
	a.branches = append(a.branches, gitrepo.Branch{Name: "first", SHA: target, Upstream: "origin/topic"}, gitrepo.Branch{Name: "second", SHA: target, Upstream: "origin/topic"})
	tt := quickCheckoutTester(a)
	tt.Key(ui.Cmd, ui.KeyB)
	tt.Frame()
	tt.Type("origin/topic")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	for _, name := range []string{"first", "second"} {
		if !tt.HasText(name) {
			t.Fatalf("explicit tracker missing %s", name)
		}
	}
	if a.busy || !a.quickCheckout.open {
		t.Fatal("auto-selected first tracker")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.quickCheckout.query != "origin/topic" || !a.quickCheckout.open {
		t.Fatal("lost search return state")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.quickCheckout.open {
		t.Fatal("search Escape did not close")
	}
}
