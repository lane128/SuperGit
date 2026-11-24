package gitrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBulkSubmoduleGitlinkCommands(t *testing.T) {
	root := fixture(t)
	seed(t, root)
	var modules strings.Builder
	for i := range 75 {
		fmt.Fprintf(&modules, "[submodule \"child-%d\"]\npath = deps/child-%d\n", i, i)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitmodules"), []byte(modules.String()), 0600); err != nil {
		t.Fatal(err)
	}
	trace := filepath.Join(t.TempDir(), "trace")
	t.Setenv("GIT_TRACE2_EVENT", trace)
	rows, err := openTest(t, root).Submodules(t.Context())
	if err != nil || len(rows) != 75 {
		t.Fatalf("inventory: %d %v", len(rows), err)
	}
	data, err := os.ReadFile(trace)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var event struct{ Event, Name string }
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatal(err)
		}
		if event.Event == "cmd_name" {
			counts[event.Name]++
		}
	}
	if counts["ls-files"] != 1 || counts["ls-tree"] != 1 {
		t.Fatalf("parent reads scaled with children: %v", counts)
	}
}

func BenchmarkChildInspectionWorkers(b *testing.B) {
	root := fixture(b)
	source := fixture(b)
	seed(b, source)
	r := openTest(b, root)
	var rows []Submodule
	for i := range 75 {
		path := fmt.Sprintf("child-%02d", i)
		git(b, root, "clone", "-q", source, path)
		rows = append(rows, Submodule{Path: path})
	}
	for _, workers := range []int{2, 4} {
		b.Run(fmt.Sprint(workers), func(b *testing.B) {
			for b.Loop() {
				r.inspectSubmodules(b.Context(), rows, workers, func(m Submodule) {
					if m.Error != "" || !m.Initialized {
						b.Errorf("%s: %s", m.Path, m.Error)
					}
				})
			}
		})
	}
}
func TestBulkGitlinksUnbornAndUnmerged(t *testing.T) {
	src := fixture(t)
	seed(t, src)
	sha := strings.TrimSpace(git(t, src, "rev-parse", "HEAD"))
	root := fixture(t)
	path := "nested/tab\t[*] child"
	git(t, root, "config", "-f", ".gitmodules", "submodule.odd.path", path)
	git(t, root, "update-index", "--add", "--cacheinfo", "160000,"+sha+","+path)
	repo := openTest(t, root)
	check := func(index, recorded string) {
		t.Helper()
		rows, e := repo.SubmoduleInventory(t.Context())
		if e != nil || len(rows) != 1 || rows[0].Path != path || rows[0].Index != index || rows[0].Recorded != recorded {
			t.Fatalf("unexpected inventory: %+v %v", rows, e)
		}
	}
	check(sha, "")
	git(t, root, "add", ".gitmodules")
	git(t, root, "commit", "-m", "links")
	check(sha, sha)
	input := fmt.Sprintf("0 %s\t%s\x00160000 %s 2\t%s\x00160000 %s 3\t%s\x00", strings.Repeat("0", len(sha)), path, sha, path, sha, path)
	if _, err := run(t.Context(), root, input, "update-index", "-z", "--index-info"); err != nil {
		t.Fatal(err)
	}
	check("", sha)
}

func TestChildWorkersShareCheckoutLimitAndCancel(t *testing.T) {
	root := fixture(t)
	child := fixture(t)
	seed(t, child)
	var groups [2][]Submodule
	for i := range 8 {
		path := fmt.Sprintf("module-%d", i)
		git(t, root, "clone", "-q", child, path)
		groups[i%2] = append(groups[i%2], Submodule{Path: path}, Submodule{Path: path})
	}
	realGit, e := exec.LookPath("git")
	if e != nil {
		t.Fatal(e)
	}
	bin := t.TempDir()
	log := filepath.Join(bin, "active")
	t.Setenv("CHILD_LOG", log)
	t.Setenv("CHILD_GIT", realGit)
	if e = os.WriteFile(filepath.Join(bin, "git"), []byte(`#!/bin/sh
case " $* " in *" status "*) printf '%s\n' "$*" >> "$CHILD_LOG";exec sleep 30;; esac
exec "$CHILD_GIT" "$@"
`), 0700); e != nil {
		t.Fatal(e)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var wg sync.WaitGroup
	for _, rows := range groups {
		repo := openTest(t, root)
		wg.Go(func() {
			repo.InspectSubmodules(ctx, rows, func(m Submodule) { t.Errorf("cancelled result published: %+v", m) })
		})
	}
	defer wg.Wait()
	defer cancel()
	deadline := time.Now().Add(5 * time.Second)
	for {
		data, _ := os.ReadFile(log)
		if strings.Count(string(data), "\n") >= 4 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("workers did not start")
		}
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)
	cancel()
	wg.Wait()
	data, e := os.ReadFile(log)
	if e != nil || strings.Count(string(data), "\n") != 4 {
		t.Fatalf("shared checkout exceeded four blocked children: %s %v", data, e)
	}
}
