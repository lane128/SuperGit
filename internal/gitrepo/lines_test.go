package gitrepo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func selectedText(d Diff, texts ...string) []int {
	var indices []int
	for i, line := range d.Lines {
		for _, text := range texts {
			if line.Text == text && (line.Kind == "add" || line.Kind == "del") {
				indices = append(indices, i)
				break
			}
		}
	}
	return indices
}
func TestApplyLinesStagesOnlySelectedReplacementAndReversesAddition(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	ctx := context.Background()
	r := openTest(t, dir)
	base := "one\ntwo\nthree\nfour\n"
	write(t, dir, "base.txt", base)
	git(t, dir, "commit", "-am", "lines")
	working := "ONE\nTWO\nthree\nfour\nfive\n"
	write(t, dir, "base.txt", working)
	d, err := r.Diff(ctx, "base.txt", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ApplyLines(ctx, "base.txt", d, selectedText(d, "-one", "+ONE"), false); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "show", ":base.txt"); got != "ONE\ntwo\nthree\nfour\n" {
		t.Fatalf("wrong index %q", got)
	}
	staged, err := r.Diff(ctx, "base.txt", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ApplyLines(ctx, "base.txt", staged, selectedText(staged, "+ONE"), true); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "show", ":base.txt"); got != "two\nthree\nfour\n" {
		t.Fatalf("reverse changed unselected deletion: %q", got)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "base.txt"))
	if string(b) != working {
		t.Fatal("working file changed")
	}
}
func TestApplyLinesReverseKeepsAdjacentStagedChanges(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	ctx := context.Background()
	r := openTest(t, dir)
	write(t, dir, "base.txt", "a\nb\nc\n")
	git(t, dir, "commit", "-am", "lines")
	write(t, dir, "base.txt", "A\nB\nc\nextra\n")
	git(t, dir, "add", "base.txt")
	d, err := r.Diff(ctx, "base.txt", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ApplyLines(ctx, "base.txt", d, selectedText(d, "-a", "+A"), true); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "show", ":base.txt"); got != "a\nB\nc\nextra\n" {
		t.Fatalf("unselected changes lost: %q", got)
	}
}
func TestApplyLinesRecountsMultipleHunksAndLiteralPaths(t *testing.T) {
	for _, path := range []string{"空 格\n*.txt", "-option.txt"} {
		t.Run(path, func(t *testing.T) {
			dir := fixture(t)
			ctx := context.Background()
			r := openTest(t, dir)
			var base strings.Builder
			for i := 1; i <= 30; i++ {
				fmt.Fprintf(&base, "line %02d\n", i)
			}
			write(t, dir, path, base.String())
			git(t, dir, "add", "--", path)
			git(t, dir, "commit", "-m", "base")
			working := strings.Replace(base.String(), "line 02\n", "line 02\nearly addition\n", 1)
			working = strings.Replace(working, "line 15\n", "middle one\nmiddle two\n", 1)
			working = strings.Replace(working, "line 28\n", "late change\n", 1)
			write(t, dir, path, working)
			d, err := r.Diff(ctx, path, false)
			if err != nil {
				t.Fatal(err)
			}
			if len(d.Hunks) != 3 {
				t.Fatalf("hunks %d", len(d.Hunks))
			}
			choice := selectedText(d, "+early addition", "-line 28", "+late change")
			if err := r.ApplyLines(ctx, path, d, choice, false); err != nil {
				t.Fatal(err)
			}
			expected := strings.Replace(base.String(), "line 02\n", "line 02\nearly addition\n", 1)
			expected = strings.Replace(expected, "line 28\n", "late change\n", 1)
			if got := git(t, dir, "show", ":"+path); got != expected {
				t.Fatalf("wrong partial index %q", got)
			}
			staged, err := r.Diff(ctx, path, true)
			if err != nil {
				t.Fatal(err)
			}
			if err := r.ApplyLines(ctx, path, staged, selectedText(staged, "-line 28", "+late change"), true); err != nil {
				t.Fatal(err)
			}
			expected = strings.Replace(base.String(), "line 02\n", "line 02\nearly addition\n", 1)
			if got := git(t, dir, "show", ":"+path); got != expected {
				t.Fatalf("reverse offsets %q", got)
			}
			b, _ := os.ReadFile(filepath.Join(dir, path))
			if string(b) != working {
				t.Fatal("working file changed")
			}
		})
	}
}
func TestApplyLinesRejectsStaleDiffAndUnsupportedSelection(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	ctx := context.Background()
	r := openTest(t, dir)
	write(t, dir, "base.txt", "changed\n")
	d, _ := r.Diff(ctx, "base.txt", false)
	write(t, dir, "base.txt", "external change\n")
	if err := r.ApplyLines(ctx, "base.txt", d, selectedText(d, "+changed"), false); err == nil {
		t.Fatal("stale Diff applied")
	}
	if got := git(t, dir, "show", ":base.txt"); got != "original\n" {
		t.Fatal("stale write changed Index")
	}
	fresh, _ := r.Diff(ctx, "base.txt", false)
	for _, selection := range [][]int{nil, {-1}, {len(fresh.Lines)}, {0}} {
		if err := r.ApplyLines(ctx, "base.txt", fresh, selection, false); err == nil {
			t.Fatalf("invalid selection accepted %v", selection)
		}
	}
	write(t, dir, "new.txt", "new\n")
	untracked, _ := r.Diff(ctx, "new.txt", false)
	if err := r.ApplyLines(ctx, "new.txt", untracked, selectedText(untracked, "+new"), false); err != nil {
		t.Fatalf("ordinary new text file rejected: %v", err)
	}
	if got := git(t, dir, "show", ":new.txt"); got != "new\n" {
		t.Fatalf("new-file Index = %q", got)
	}
}
func TestApplyLinesRejectsExistingSymlinkInBothDirections(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(fmt.Sprint("reverse-", reverse), func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			ctx := context.Background()
			r := openTest(t, dir)
			if err := os.Symlink("old-target", filepath.Join(dir, "link")); err != nil {
				t.Fatal(err)
			}
			git(t, dir, "add", "--", "link")
			git(t, dir, "commit", "-m", "symlink")
			if err := os.Remove(filepath.Join(dir, "link")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("new-target", filepath.Join(dir, "link")); err != nil {
				t.Fatal(err)
			}
			expectedIndex := "old-target"
			if reverse {
				git(t, dir, "add", "--", "link")
				expectedIndex = "new-target"
			}
			d, err := r.Diff(ctx, "link", reverse)
			if err != nil {
				t.Fatal(err)
			}
			if d.CanStageHunks {
				t.Error("existing symlink exposes line/hunk controls")
			}
			var selection []int
			for i, l := range d.Lines {
				if l.Kind == "add" || l.Kind == "del" {
					selection = append(selection, i)
				}
			}
			if len(selection) == 0 {
				t.Fatal("test did not load changed symlink target")
			}
			if err := r.ApplyLines(ctx, "link", d, selection, reverse); err == nil {
				t.Error("symlink accepted as ordinary text")
			}
			if got := git(t, dir, "show", ":link"); got != expectedIndex {
				t.Errorf("Index target changed: %q", got)
			}
			if target, err := os.Readlink(filepath.Join(dir, "link")); err != nil || target != "new-target" {
				t.Fatal("working symlink changed", target, err)
			}
		})
	}
}
func TestApplyLinesEmptyRangesAndUnevenReplacement(t *testing.T) {
	for _, tc := range []struct {
		name, base, working string
		choose              []string
		expected            string
	}{
		{"empty insertion", "", "A\nB\n", []string{"+A"}, "A\n"},
		{"empty target", "A\nB\n", "", []string{"-A", "-B"}, ""},
		{"partial removal", "A\nB\n", "", []string{"-A"}, "B\n"},
		{"long replacement", "a\nb\nend\n", "A\nB\nC\nend\n", []string{"-b", "+B", "+C"}, "a\nB\nC\nend\n"},
		{"short replacement", "a\nb\nc\nend\n", "A\nend\n", []string{"-b"}, "a\nc\nend\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			r := openTest(t, dir)
			ctx := context.Background()
			write(t, dir, "base.txt", tc.base)
			git(t, dir, "commit", "-am", "base")
			write(t, dir, "base.txt", tc.working)
			d, err := r.Diff(ctx, "base.txt", false)
			if err != nil {
				t.Fatal(err)
			}
			if err := r.ApplyLines(ctx, "base.txt", d, selectedText(d, tc.choose...), false); err != nil {
				t.Fatal(err)
			}
			if got := git(t, dir, "show", ":base.txt"); got != tc.expected {
				t.Fatalf("index %q, want %q", got, tc.expected)
			}
			staged, err := r.Diff(ctx, "base.txt", true)
			if err != nil {
				t.Fatal(err)
			}
			var all []int
			for i, l := range staged.Lines {
				if l.Kind == "add" || l.Kind == "del" {
					all = append(all, i)
				}
			}
			if err := r.ApplyLines(ctx, "base.txt", staged, all, true); err != nil {
				t.Fatal(err)
			}
			if got := git(t, dir, "show", ":base.txt"); got != tc.base {
				t.Fatalf("reverse %q, want %q", got, tc.base)
			}
			b, err := os.ReadFile(filepath.Join(dir, "base.txt"))
			if err != nil || string(b) != tc.working {
				t.Fatal("working file changed", err)
			}
		})
	}
}
func TestDiscardLinesRevertsOnlySelectedWorktreeChanges(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	ctx := context.Background()
	r := openTest(t, dir)
	write(t, dir, "base.txt", "one\ntwo\nthree\nfour\n")
	git(t, dir, "commit", "-am", "discard base")
	write(t, dir, "base.txt", "ONE\ntwo\nthree\nfour\nfive\n")
	d, err := r.Diff(ctx, "base.txt", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.DiscardLines(ctx, "base.txt", d, selectedText(d, "+five")); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "base.txt"))
	if string(b) != "ONE\ntwo\nthree\nfour\n" {
		t.Fatalf("unselected change lost: %q", string(b))
	}
	if _, err := r.DiscardLines(ctx, "base.txt", d, selectedText(d, "+five")); err == nil {
		t.Fatal("stale diff accepted")
	}
	d, _ = r.Diff(ctx, "base.txt", false)
	if _, err := r.DiscardLines(ctx, "base.txt", d, selectedText(d, "-one", "+ONE")); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(filepath.Join(dir, "base.txt"))
	if string(b) != "one\ntwo\nthree\nfour\n" {
		t.Fatalf("replacement not discarded: %q", string(b))
	}
	if got := git(t, dir, "show", ":base.txt"); got != "one\ntwo\nthree\nfour\n" {
		t.Fatalf("index changed: %q", got)
	}
}
func TestDiscardLinesRejectsUntrackedAndDeletedFiles(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	ctx := context.Background()
	r := openTest(t, dir)
	write(t, dir, "gone.txt", "bye\n")
	git(t, dir, "add", "gone.txt")
	git(t, dir, "commit", "-m", "gone")
	if err := os.Remove(filepath.Join(dir, "gone.txt")); err != nil {
		t.Fatal(err)
	}
	d, err := r.Diff(ctx, "gone.txt", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.DiscardLines(ctx, "gone.txt", d, selectedText(d, "-bye")); err == nil {
		t.Fatal("discard on deleted file accepted")
	}
	write(t, dir, "fresh.txt", "new\n")
	d, err = r.Diff(ctx, "fresh.txt", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Lines) > 0 {
		if _, err := r.DiscardLines(ctx, "fresh.txt", d, selectedText(d, "+new")); err == nil {
			t.Fatal("discard on untracked file accepted")
		}
	}
}

func TestApplyLinesPreservesCRLFAndHandlesNoFinalNewline(t *testing.T) {
	for _, ending := range []string{"\r\n", ""} {
		t.Run(fmt.Sprintf("ending-%q", ending), func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			ctx := context.Background()
			r := openTest(t, dir)
			write(t, dir, "base.txt", "before"+ending)
			git(t, dir, "commit", "-am", "line ending")
			working := "after" + ending
			write(t, dir, "base.txt", working)
			d, _ := r.Diff(ctx, "base.txt", false)
			if ending == "" {
				if err := r.ApplyLines(ctx, "base.txt", d, selectedText(d, "+after"), false); err == nil {
					t.Fatal("partial no-newline change accepted")
				}
			}
			choice := selectedText(d, "-before"+strings.TrimSuffix(ending, "\n"), "+after"+strings.TrimSuffix(ending, "\n"))
			if err := r.ApplyLines(ctx, "base.txt", d, choice, false); err != nil {
				t.Fatal(err)
			}
			if got := git(t, dir, "show", ":base.txt"); got != working {
				t.Fatalf("line ending changed %q", got)
			}
			staged, _ := r.Diff(ctx, "base.txt", true)
			choice = selectedText(staged, "-before"+strings.TrimSuffix(ending, "\n"), "+after"+strings.TrimSuffix(ending, "\n"))
			if err := r.ApplyLines(ctx, "base.txt", staged, choice, true); err != nil {
				t.Fatal(err)
			}
			if got := git(t, dir, "show", ":base.txt"); got != "before"+ending {
				t.Fatalf("reverse line ending %q", got)
			}
		})
	}
}
func TestUndoDiscardRestoresSelectedLines(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	ctx := context.Background()
	r := openTest(t, dir)
	write(t, dir, "base.txt", "one\ntwo\n")
	git(t, dir, "commit", "-am", "undo base")
	write(t, dir, "base.txt", "ONE\ntwo\nfive\n")
	d, err := r.Diff(ctx, "base.txt", false)
	if err != nil {
		t.Fatal(err)
	}
	patch, err := r.DiscardLines(ctx, "base.txt", d, selectedText(d, "+five"))
	if err != nil {
		t.Fatal(err)
	}
	if patch == "" {
		t.Fatal("discard returned no patch for undo")
	}
	b, _ := os.ReadFile(filepath.Join(dir, "base.txt"))
	if string(b) != "ONE\ntwo\n" {
		t.Fatalf("discard result: %q", string(b))
	}
	if err := r.UndoDiscard(ctx, "base.txt", patch); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(filepath.Join(dir, "base.txt"))
	if string(b) != "ONE\ntwo\nfive\n" {
		t.Fatalf("undo result: %q", string(b))
	}
	if got := git(t, dir, "show", ":base.txt"); got != "one\ntwo\n" {
		t.Fatalf("index changed by undo: %q", got)
	}
	write(t, dir, "base.txt", "ONE\ntwo\nsix\n")
	if err := r.UndoDiscard(ctx, "base.txt", patch); err == nil {
		t.Fatal("undo accepted after the file changed")
	}
	if err := r.UndoDiscard(ctx, "../outside.txt", patch); err == nil {
		t.Fatal("path outside repository accepted")
	}
	if err := r.UndoDiscard(ctx, "base.txt", ""); err == nil {
		t.Fatal("empty patch accepted")
	}
}
