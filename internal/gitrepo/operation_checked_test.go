package gitrepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckedOperationControlRejectsReplacementOfSameKind(t *testing.T) {
	for _, control := range []string{"abort", "skip", "continue"} {
		t.Run(control, func(t *testing.T) {
			ctx := context.Background()
			dir := conflictFixture(t)
			first := strings.TrimSpace(git(t, dir, "rev-parse", "feature"))
			git(t, dir, "switch", "feature")
			write(t, dir, "base.txt", "next theirs\n")
			git(t, dir, "commit", "-am", "next feature")
			git(t, dir, "switch", "main")
			repo := openTest(t, dir)
			if err := repo.Integrate(ctx, "cherry-pick", first); err == nil {
				t.Fatal("fixture lacks original conflict")
			}
			expected, err := repo.Operation(ctx)
			if err != nil {
				t.Fatal(err)
			}
			git(t, dir, "cherry-pick", "--abort")
			if err := repo.Integrate(ctx, "cherry-pick", "feature"); err == nil {
				t.Fatal("fixture lacks replacement conflict")
			}
			before, err := repo.Operation(ctx)
			if err != nil || before == expected || before.Kind != expected.Kind || before.Head != expected.Head {
				t.Fatalf("fixture requires same-kind same-HEAD replacement: %+v %+v %v", expected, before, err)
			}
			if control == "continue" {
				write(t, dir, "base.txt", "replacement resolution\n")
				git(t, dir, "add", "--", "base.txt")
			}
			index, head := git(t, dir, "ls-files", "--stage"), git(t, dir, "rev-parse", "HEAD")
			bytes, err := os.ReadFile(filepath.Join(dir, "base.txt"))
			if err != nil {
				t.Fatal(err)
			}
			switch control {
			case "abort":
				err = repo.AbortChecked(ctx, expected)
			case "skip":
				err = repo.SkipChecked(ctx, expected)
			case "continue":
				err = repo.ContinueChecked(ctx, expected)
			}
			if !errors.Is(err, ErrOperationTargetChanged) {
				t.Fatalf("old operation identity needs changed-target refusal, got %v", err)
			}
			after, err := repo.Operation(ctx)
			if err != nil || before != after || git(t, dir, "ls-files", "--stage") != index || git(t, dir, "rev-parse", "HEAD") != head {
				t.Fatal("refused control changed operation, HEAD or Index")
			}
			afterBytes, err := os.ReadFile(filepath.Join(dir, "base.txt"))
			if err != nil || string(afterBytes) != string(bytes) {
				t.Fatal("refused control changed working bytes")
			}
		})
	}
}

func TestCheckedOperationControlAcceptsOriginalIdentity(t *testing.T) {
	for _, control := range []string{"abort", "skip", "continue"} {
		t.Run(control, func(t *testing.T) {
			ctx := context.Background()
			dir := conflictFixture(t)
			repo := openTest(t, dir)
			head := git(t, dir, "rev-parse", "HEAD")
			if err := repo.Integrate(ctx, "cherry-pick", "feature"); err == nil {
				t.Fatal("fixture lacks conflict")
			}
			expected, err := repo.Operation(ctx)
			if err != nil || expected.Token == "" {
				t.Fatalf("missing operation identity: %v", err)
			}
			if control == "continue" {
				write(t, dir, "base.txt", "resolved original\n")
				git(t, dir, "add", "--", "base.txt")
			}
			switch control {
			case "abort":
				err = repo.AbortChecked(ctx, expected)
			case "skip":
				err = repo.SkipChecked(ctx, expected)
			case "continue":
				err = repo.ContinueChecked(ctx, expected)
			}
			if err != nil {
				t.Fatal(err)
			}
			after, err := repo.Operation(ctx)
			if err != nil || after.Kind != "" {
				t.Fatalf("original control did not finish operation: %+v %v", after, err)
			}
			if moved := git(t, dir, "rev-parse", "HEAD") != head; moved != (control == "continue") {
				t.Fatalf("unexpected HEAD movement for %s: %v", control, moved)
			}
		})
	}
}
