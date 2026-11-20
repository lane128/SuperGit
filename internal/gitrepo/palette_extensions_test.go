package gitrepo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func extensionStatus(t *testing.T, r *Repository) Status {
	t.Helper()
	s, err := r.Status(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestPaletteExtensionPatchAndSubmodule(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	s := extensionStatus(t, r)
	patch := filepath.Join(t.TempDir(), "-example patch")
	if err := os.WriteFile(patch, []byte("diff --git a/base.txt b/base.txt\n--- a/base.txt\n+++ b/base.txt\n@@ -1 +1 @@\n-original\n+patched\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := r.ApplyPatchFile(t.Context(), patch, s); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "base.txt")); string(data) != "patched\n" {
		t.Fatalf("patch result %q", data)
	}
	if got := git(t, dir, "diff", "--cached", "--name-only"); got != "" {
		t.Fatalf("staged %q", got)
	}
	child := fixture(t)
	seed(t, child)
	t.Setenv("GIT_ALLOW_PROTOCOL", "file")
	if err := r.AddPaletteSubmodule(t.Context(), child, "library", "deps/with space", s); err != nil {
		t.Fatal(err)
	}
	mods, err := r.Submodules(t.Context())
	if err != nil || len(mods) != 1 {
		t.Fatal(mods, err)
	}
	path, err := r.OpenPaletteSubmodule(t.Context(), mods[0])
	if err != nil || path != filepath.Join(r.Root, "deps/with space") {
		t.Fatal(path, err)
	}
	for _, dst := range []string{"../outside", ".git/metadata"} {
		if err := r.AddPaletteSubmodule(t.Context(), child, "bad", dst, s); err == nil {
			t.Fatalf("accepted %q", dst)
		}
	}
	if err := r.AddPaletteSubmodule(t.Context(), "ext::invalid", "bad", "bad", s); err == nil {
		t.Fatal("accepted external transport")
	}
}

func TestPaletteExtensionPatternPreservesAttributesAndScope(t *testing.T) {
	dir := fixture(t)
	r := openTest(t, dir)
	status := extensionStatus(t, r)
	write(t, dir, "assets/.gitattributes", "# rules\r\n\"*.big image\" filter=lfs diff=lfs -text lockable custom=value\r\n*.go text\r\n")
	rows, err := r.LFSPatterns(t.Context())
	if err != nil || len(rows) != 2 {
		t.Fatal(rows, err)
	}
	var old, add LFSPattern
	for _, row := range rows {
		if row.Pattern == "" {
			add = row
		} else {
			old = row
		}
	}
	if err := r.EditLFSPattern(t.Context(), old, "*.art file", status); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, old.File)); string(data) != "# rules\r\n\"*.art file\" filter=lfs diff=lfs -text lockable custom=value\r\n*.go text\r\n" {
		t.Fatalf("changed unrelated attributes %q", data)
	}
	if err := r.EditLFSPattern(t.Context(), old, "", status); !errors.Is(err, ErrIntegrationTargetChanged) {
		t.Fatalf("stale rule %v", err)
	}
	if err := r.EditLFSPattern(t.Context(), add, "*.large", status); err != nil {
		t.Fatal(err)
	}
	rows, err = r.LFSPatterns(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range rows {
		if p.Pattern == "*.art file" {
			if err := r.EditLFSPattern(t.Context(), p, "", status); err != nil {
				t.Fatal(err)
			}
		}
	}
	if data, _ := os.ReadFile(filepath.Join(dir, old.File)); string(data) != "# rules\r\n*.go text\r\n" {
		t.Fatalf("removed unrelated lines %q", data)
	}
	attr := git(t, dir, "check-attr", "filter", "--", "file.large")
	if !strings.Contains(attr, "filter: lfs") {
		t.Fatalf("new pattern not active %q", attr)
	}
}

func flowRequest(t *testing.T, r *Repository, action, kind, name string) FlowCommand {
	t.Helper()
	ctx := t.Context()
	conf, err := r.FlowConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	branches, err := r.Branches(ctx)
	if err != nil {
		t.Fatal(err)
	}
	remotes, err := r.Remotes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	q := FlowCommand{Action: action, Kind: kind, Name: name, Config: conf, Expected: extensionStatus(t, r), Branches: branches, Base: conf.DefaultBase(kind)}
	if len(remotes) > 0 {
		q.Remote = remotes[0]
	}
	return q
}

func TestPaletteExtensionFlowWorkflows(t *testing.T) {
	if !HasGitFlow() {
		t.Skip("git-flow unavailable")
	}
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	remote := t.TempDir()
	git(t, remote, "init", "--bare")
	git(t, dir, "remote", "add", "origin", remote)
	run := func(t *testing.T, q FlowCommand) {
		t.Helper()
		if err := r.RunFlow(t.Context(), q); err != nil {
			t.Fatalf("%s %s %s: %v", q.Kind, q.Action, q.Name, err)
		}
	}
	initFlow := flowRequest(t, r, "init", "", "")
	initFlow.Production, initFlow.Development = "main", "develop"
	run(t, initFlow)
	config, err := r.FlowConfig(t.Context())
	if err != nil || config["gitflow.branch.master"] != "main" {
		t.Fatal(config, err)
	}
	if err := r.SetFlowConfig(t.Context(), config, "gitflow.prefix.feature", "topic/"); err != nil {
		t.Fatal(err)
	}
	if err := r.SetFlowConfig(t.Context(), config, "gitflow.prefix.feature", "stale/"); !errors.Is(err, ErrIntegrationTargetChanged) {
		t.Fatal(err)
	}
	git(t, dir, "push", "origin", "main", "develop")
	for _, kind := range []string{"feature", "bugfix", "release", "hotfix", "support"} {
		t.Run(kind, func(t *testing.T) {
			name := "test-" + kind
			q := flowRequest(t, r, "start", kind, name)
			run(t, q)
			branch := q.Config["gitflow.prefix."+kind] + name
			if got := extensionStatus(t, r).Branch; got != branch {
				t.Fatalf("current branch %q", got)
			}
			write(t, dir, kind+".txt", "content\n")
			git(t, dir, "add", "--", kind+".txt")
			git(t, dir, "commit", "-m", kind+" work")
			run(t, flowRequest(t, r, "rebase", kind, name))
			if kind == "support" {
				return
			}
			run(t, flowRequest(t, r, "publish", kind, name))
			q = flowRequest(t, r, "finish", kind, name)
			q.TagMessage = "Completed " + name
			run(t, q)
			if !strings.Contains(git(t, remote, "show-ref"), "refs/heads/"+branch) {
				t.Fatal("local finish removed published branch")
			}
			if strings.Contains(git(t, dir, "for-each-ref", "--format=%(refname)", "refs/heads/"), "refs/heads/"+branch+"\n") {
				t.Fatal("finished local branch still exists")
			}
			if kind == "release" || kind == "hotfix" {
				if got := git(t, dir, "tag", "-l", name); strings.TrimSpace(got) != name {
					t.Fatalf("missing tag %s", name)
				}
			}
			git(t, dir, "push", "origin", "main", "develop")
		})
	}
}

func TestPaletteExtensionLFSLocks(t *testing.T) {
	if !HasGitLFS() {
		t.Skip("git-lfs unavailable")
	}
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	lock := "{\"id\":\"1\",\"path\":\"base.txt\",\"owner\":{\"name\":\"Test Developer\"},\"locked_at\":\"2020-01-01T00:00:00Z\"}"
	var locked atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.git-lfs+json")
		switch {
		case req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/locks/verify"):
			if locked.Load() {
				_, _ = w.Write([]byte("{\"ours\":[" + lock + "],\"theirs\":[]}"))
			} else {
				_, _ = w.Write([]byte("{\"ours\":[],\"theirs\":[]}"))
			}
		case req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/locks/1/unlock"):
			locked.Store(false)
			_, _ = w.Write([]byte("{\"lock\":" + lock + "}"))
		case req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/locks"):
			locked.Store(true)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("{\"lock\":" + lock + "}"))
		default:
			http.Error(w, "unexpected API resource "+req.URL.Path, 404)
		}
	}))
	defer server.Close()
	git(t, dir, "remote", "add", "origin", server.URL+"/repo")
	git(t, dir, "config", "lfs.url", server.URL+"/repo/info/lfs")
	git(t, dir, "config", "lfs.locksverify", "false")
	write(t, dir, ".gitattributes", "base.txt lockable\n")
	remotes, err := r.Remotes(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, unlock := range []bool{false, true} {
		list, err := r.LFSLockFiles(t.Context(), remotes[0], unlock)
		if err != nil || len(list.Files) != 1 {
			t.Fatal(list.Files, err)
		}
		if err := r.ChangeLFSLock(t.Context(), remotes[0], list, list.Files[0], extensionStatus(t, r)); err != nil {
			t.Fatal(err)
		}
	}
	list, err := r.LFSLockFiles(context.Background(), remotes[0], true)
	if err != nil || len(list.Files) != 0 {
		t.Fatal(list.Files, err)
	}
}

func TestPaletteExtensionFlowInitBranchesAndStaleSelection(t *testing.T) {
	if !HasGitFlow() {
		t.Skip("git-flow unavailable")
	}
	for _, empty := range []bool{false, true} {
		dir := fixture(t)
		if !empty {
			seed(t, dir)
			git(t, dir, "branch", "-m", "production-app")
			git(t, dir, "branch", "integration-app")
		}
		r := openTest(t, dir)
		q := flowRequest(t, r, "init", "", "")
		q.Production, q.Development = "production-app", "integration-app"
		if err := r.RunFlow(t.Context(), q); err != nil {
			t.Fatal(err)
		}
		c, err := r.FlowConfig(t.Context())
		if err != nil || c["gitflow.branch.develop"] != "integration-app" || c["gitflow.branch.master"] != "production-app" {
			t.Fatal(c, err)
		}
		q = flowRequest(t, r, "start", "feature", "stale")
		git(t, dir, "checkout", "integration-app")
		git(t, dir, "commit", "--allow-empty", "-m", "external update")
		if err := r.RunFlow(t.Context(), q); !errors.Is(err, ErrIntegrationTargetChanged) {
			t.Fatalf("stale branch accepted: %v", err)
		}
		if got := git(t, dir, "branch", "--list", "feature/stale"); got != "" {
			t.Fatalf("stale command created %s", got)
		}
	}
}
