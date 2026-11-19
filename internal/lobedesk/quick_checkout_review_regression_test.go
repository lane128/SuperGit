package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
	"testing"
)

func TestQuickCheckoutRemoteReviewAmbiguousIdentity(t *testing.T) {
	a, _, target := quickCheckoutFixture(t)
	a.branches = append(a.branches, gitrepo.Branch{Name: "origin/topic", SHA: a.status.Head})
	a.remoteBranches = []gitrepo.Branch{{Name: "origin/topic", SHA: target}}
	tt := quickCheckoutTester(a)
	tt.Key(ui.Cmd, ui.KeyB)
	tt.Frame()
	tt.Type("origin/topic")
	tt.Frame()
	var names, descriptions []string
	for _, node := range tt.AccessAll() {
		if node.Role == ui.RoleOption {
			names = append(names, node.Name)
			descriptions = append(descriptions, node.Description)
		}
	}
	if len(names) != 2 {
		t.Fatalf("expected both candidates, got %v", names)
	}
	if names[0] == names[1] && descriptions[0] == descriptions[1] {
		t.Fatal("local and remote destinations are indistinguishable")
	}
}

func TestQuickCheckoutRemoteReviewInvalidNameRetainsForm(t *testing.T) {
	for _, name := range []string{"bad..name", "feature test", "-option", "HEAD", "bad@{name", ".hidden", "topic.lock", "topic/", "a//b", "bad?name", "bad\\name"} {
		t.Run(name, func(t *testing.T) {
			a, callbacks, target := quickCheckoutFixture(t)
			a.remoteBranches = []gitrepo.Branch{{Name: "origin/topic", SHA: target}}
			tt := quickCheckoutTester(a)
			tt.Key(ui.Cmd, ui.KeyB)
			tt.Frame()
			tt.Type("origin/topic")
			tt.Frame()
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			tt.Key(ui.Cmd, ui.KeyA)
			tt.Type(name)
			tt.Frame()
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			waitQuickCheckout(t, a, callbacks)
			if !a.quickCheckout.open || a.quickCheckout.step != "name" || a.quickCheckout.localName != name {
				t.Fatal("invalid branch name lost form and editable value")
			}
			if a.quickCheckoutNameReason() == "" {
				t.Fatal("invalid name lacks inline feedback")
			}
			if a.commitText != "draft\n\nbody" || !a.amend {
				t.Fatal("invalid name changed draft")
			}
			tt.Key(ui.Cmd, ui.KeyA)
			tt.Type("corrected-topic")
			tt.Frame()
			if a.quickCheckoutNameReason() != "" || a.quickCheckout.localName != "corrected-topic" {
				t.Fatal("cannot correct invalid name in place")
			}
		})
	}
}
