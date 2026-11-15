package gitrepo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestTagsExposeExactObjectsAndCommitMetadataWithoutWrites(t *testing.T) {
	dir := fixture(t)
	r := openTest(t, dir)
	ctx := context.Background()
	if tags, err := r.Tags(ctx); err != nil || len(tags) != 0 {
		t.Fatalf("empty tags: %+v %v", tags, err)
	}
	seed(t, dir)
	parent := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	git(t, dir, "commit", "--allow-empty", "-m", "Selected release α\n\nBody remains separately readable")
	selected := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	git(t, dir, "tag", "release/候选", selected)
	git(t, dir, "tag", "-a", "annotated", "-m", "Tag annotation differs from commit subject\n\nSecond paragraph stays readable", selected)
	annotated := strings.TrimSpace(git(t, dir, "rev-parse", "refs/tags/annotated"))
	git(t, dir, "tag", "-a", "nested", "-m", "Nested tag", "refs/tags/annotated")
	git(t, dir, "tag", "tree", "HEAD^{tree}")
	git(t, dir, "tag", "blob", "HEAD:base.txt")
	git(t, dir, "branch", "annotated", parent) // No short-name ambiguity.
	git(t, dir, "commit", "--allow-empty", "-m", "Later HEAD")
	git(t, dir, "symbolic-ref", "refs/tags/symbolic", "refs/heads/main")
	write(t, dir, "base.txt", "index version\n")
	git(t, dir, "add", "--", "base.txt")
	write(t, dir, "base.txt", "worktree version\n")
	head := git(t, dir, "rev-parse", "HEAD")
	index := git(t, dir, "ls-files", "--stage")
	refs := git(t, dir, "for-each-ref", "--format=%(refname) %(objectname) %(symref)")
	tags, err := r.Tags(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	byName := map[string]Tag{}
	for _, tag := range tags {
		names = append(names, tag.Name)
		byName[tag.Name] = tag
	}
	if !reflect.DeepEqual(names, []string{"annotated", "blob", "nested", "release/候选", "symbolic", "tree"}) {
		t.Fatalf("wrong exact names/order: %v", names)
	}
	for _, name := range []string{"release/候选", "annotated", "nested"} {
		tag := byName[name]
		if tag.Commit.SHA != selected || tag.Commit.Subject != "Selected release α" || !reflect.DeepEqual(tag.Commit.Parents, []string{parent}) || tag.Commit.Author == "" || tag.Commit.Date == "" || tag.TargetType != "commit" || tag.TargetSHA != selected {
			t.Fatalf("wrong commit metadata for %s: %+v", name, tag)
		}
	}
	if tag := byName["annotated"]; tag.ObjectSHA != annotated || tag.ObjectSHA == selected || tag.ObjectType != "tag" || tag.Message != "Tag annotation differs from commit subject\n\nSecond paragraph stays readable" {
		t.Fatalf("annotation identity/body lost: %+v", tag)
	}
	if tag := byName["nested"]; tag.Message != "Nested tag" {
		t.Fatalf("nested annotation message = %q; want its own subject", tag.Message)
	}
	if tag := byName["release/候选"]; tag.ObjectSHA != selected || tag.ObjectType != "commit" {
		t.Fatalf("lightweight identity lost: %+v", tag)
	}
	if tag := byName["release/候选"]; tag.Message != "" {
		t.Fatalf("lightweight tag took the target's subject as a message: %q", tag.Message)
	}
	for _, name := range []string{"symbolic", "tree", "blob"} {
		if tag := byName[name]; tag.Message != "" {
			t.Fatalf("%s carries a tag message: %q", name, tag.Message)
		}
	}
	for _, name := range []string{"blob", "tree"} {
		if byName[name].Commit.SHA != "" {
			t.Fatalf("%s was treated as a commit", name)
		}
	}
	if byName["blob"].TargetType != "blob" || byName["tree"].TargetType != "tree" || byName["nested"].TargetType != "commit" || byName["nested"].TargetSHA != selected || byName["nested"].ObjectSHA == annotated {
		t.Fatal("noncommit target type/OID lost")
	}
	if byName["symbolic"].SymbolicRef != "refs/heads/main" || byName["symbolic"].Commit.SHA != strings.TrimSpace(head) {
		t.Fatal("symbolic tag not labeled/resolved correctly")
	}
	if git(t, dir, "rev-parse", "HEAD") != head || git(t, dir, "ls-files", "--stage") != index || git(t, dir, "for-each-ref", "--format=%(refname) %(objectname) %(symref)") != refs {
		t.Fatal("tag inspection changed HEAD/Index/refs")
	}
	b, err := os.ReadFile(filepath.Join(dir, "base.txt"))
	if err != nil || string(b) != "worktree version\n" {
		t.Fatal("working bytes changed", err)
	}
}

func TestTagsDanglingSymbolicReferenceInventory(t *testing.T) {
	for _, withValid := range []bool{false, true} {
		t.Run(fmt.Sprint(withValid), func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			if withValid {
				git(t, dir, "tag", "valid")
			}
			git(t, dir, "symbolic-ref", "refs/tags/dangling", "refs/heads/missing")
			r := openTest(t, dir)
			tags, err := r.Tags(context.Background())
			want := 0
			if withValid {
				want = 1
			}
			// Characterize Git's enumeration boundary: it does not expose
			// dangling symbolic refs. UI must promise resolvable tags only.
			if err != nil || len(tags) != want || withValid && tags[0].Name != "valid" {
				t.Fatalf("unexpected resolvable inventory: %+v %v", tags, err)
			}
			if got := strings.TrimSpace(git(t, dir, "symbolic-ref", "refs/tags/dangling")); got != "refs/heads/missing" {
				t.Fatal("tag read changed dangling symbolic ref")
			}
		})
	}
}

func TestTagsRejectCancelledAndUnavailableObjectReads(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	git(t, dir, "tag", "valid")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if tags, err := r.Tags(ctx); err == nil || len(tags) != 0 {
		t.Fatalf("cancelled tags read accepted: %+v %v", tags, err)
	}
	bad := strings.Repeat("f", 40) + "\n"
	path := filepath.Join(dir, ".git", "refs", "tags", "broken")
	if err := os.WriteFile(path, []byte(bad), 0600); err != nil {
		t.Fatal(err)
	}
	if tags, err := r.Tags(context.Background()); err == nil || len(tags) != 0 {
		t.Fatalf("unavailable tag object became partial/empty success: %+v %v", tags, err)
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != bad {
		t.Fatal("read repaired/altered broken ref", err)
	}
}
