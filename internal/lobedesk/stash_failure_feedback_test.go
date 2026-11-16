package lobedesk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStashRestoreFailureExplainsPartialChangesAndKeepsGitOutput(t *testing.T) {
	for _, lang := range []locale{localeEnglish, localeChinese} {
		for _, pop := range []bool{false, true} {
			name := string(lang) + "/apply"
			if pop {
				name = string(lang) + "/pop"
			}
			t.Run(name, func(t *testing.T) {
				a, callbacks := overviewFixture(t)
				// MyGo RunOnMain waits for the callback. Match that contract so the
				// worker's deferred cancel does not precede result classification.
				a.runOnMain = func(fn func()) {
					returned := make(chan struct{})
					callbacks <- func() { defer close(returned); fn() }
					select {
					case <-returned:
					case <-a.ctx.Done():
					}
				}
				a.locale = lang
				if err := a.repo.SaveStash(a.ctx, "feedback fixture", true); err != nil {
					t.Fatal(err)
				}
				stashes, err := a.repo.Stashes(a.ctx)
				if err != nil || len(stashes) != 1 {
					t.Fatalf("stash fixture: %v %v", stashes, err)
				}
				path := filepath.Join(a.repo.Root, "same.go")
				const dirty = "preserve dirty bytes\n"
				if err := os.WriteFile(path, []byte(dirty), 0600); err != nil {
					t.Fatal(err)
				}
				if pop {
					a.popStash(stashes[0])
				} else {
					a.applyStash(stashes[0])
				}
				select {
				case fn := <-callbacks:
					fn()
				case <-time.After(10 * time.Second):
					t.Fatal("stash completion unavailable")
				}
				if len(a.notices[toastMain]) != 1 {
					t.Fatalf("notices: %v", a.notices[toastMain])
				}
				notice := a.notices[toastMain][0]
				if notice.kind != "error" {
					t.Fatalf("expected failure: %+v", notice)
				}
				detail := notice.description
				if lang == localeEnglish {
					if !strings.HasPrefix(detail, "Check the working tree") || !strings.Contains(detail, "untracked") || strings.Contains(detail, "Stash 与工作区") {
						t.Fatalf("missing localized partial-restoration guidance: %q", detail)
					}
				} else if !strings.HasPrefix(detail, "请检查工作区") || !strings.Contains(detail, "未跟踪") {
					t.Fatalf("missing Chinese partial-restoration guidance: %q", detail)
				}
				if !strings.Contains(detail, "git stash:") || !strings.Contains(detail, "would be overwritten by merge") {
					t.Fatalf("original Git output missing: %q", detail)
				}
				after, err := a.repo.Stashes(a.ctx)
				if err != nil || len(after) != 1 || after[0].SHA != stashes[0].SHA {
					t.Fatalf("failed restore lost stash: %v %v", after, err)
				}
				if bytes, err := os.ReadFile(path); err != nil || string(bytes) != dirty {
					t.Fatalf("failure replaced existing bytes: %q %v", bytes, err)
				}
			})
		}
	}
}
