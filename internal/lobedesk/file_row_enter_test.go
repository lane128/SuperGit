package lobedesk

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// Sublime Merge Build 2125 binds Enter to stage_file/unstage_file in its
// table_of_contents controls. The focused row supplies the raw path/version.
func TestFileRowEnterStagesFocusedVersion(t *testing.T) {
	for _, staged := range []bool{false, true} {
		name := "stage"
		if staged {
			name = "unstage"
		}
		t.Run(name, func(t *testing.T) {
			a, callbacks := overviewFixture(t)
			a.page, a.locale = "history", localeEnglish
			// Avoid launching another application when testing the old Enter behavior.
			bin := t.TempDir()
			for _, command := range []string{"open", "xdg-open"} {
				if err := os.WriteFile(filepath.Join(bin, command), []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			readGit := func(args ...string) string {
				t.Helper()
				cmd := exec.Command("git", args...)
				cmd.Dir = a.repo.Root
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("git %v: %v: %s", args, err, out)
				}
				return string(out)
			}
			head := readGit("rev-parse", "HEAD")
			// Preview the other version, then give keyboard focus to the target
			// row without activating it, as roving focus in the Files pane does.
			a.selectFile("same.go", !staged)
			tt := pageTester(a, func(c *ui.Context) {
				a.registerShortcuts(c)
				ui.Column(c).Height(500).FillWidth().Children(func() {
					for _, version := range []bool{false, true} {
						row := a.fileRow(c.Key(version), a.status.Entries[0], version)
						if version == staged {
							row.Focus()
						}
					}
				})
			})
			if a.selectedStaged != !staged {
				t.Fatal("fixture preview must differ from focused version")
			}
			tt.Key(0, ui.KeyEnter)
			if !a.busy {
				t.Fatal("Enter did not start the focused file stage action")
			}
			timer := time.NewTimer(10 * time.Second)
			defer timer.Stop()
			for a.busy || a.loading {
				select {
				case fn := <-callbacks:
					fn()
				case <-timer.C:
					t.Fatal("Enter file action did not finish")
				}
			}
			expected := "working version\n"
			if staged {
				expected = "original\n"
			}
			if got := readGit("show", ":same.go"); got != expected {
				t.Fatalf("wrong Index version: %q want %q", got, expected)
			}
			if got := readGit("rev-parse", "HEAD"); got != head {
				t.Fatal("file action changed HEAD")
			}
			data, err := os.ReadFile(filepath.Join(a.repo.Root, "same.go"))
			if err != nil || string(data) != "working version\n" {
				t.Fatalf("file action changed worktree: %q %v", data, err)
			}
		})
	}
}
