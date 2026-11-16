package lobedesk

import "testing"

func TestRepositorySwitchPreservesIndependentWindowNotices(t *testing.T) {
	a, done, _ := worktreeApp(t)
	_, _, nextRoot := worktreeApp(t)
	stale := &toastNotice{root: a.repo.Root, kind: "error", title: "old repository result"}
	independentMain := &toastNotice{kind: "error", title: "main preferences save failure"}
	settingsFirst := &toastNotice{kind: "error", title: "settings save failure"}
	settingsSecond := &toastNotice{kind: "success", title: "settings saved"}
	a.notices[toastMain] = []*toastNotice{stale, independentMain}
	a.notices[toastSettings] = []*toastNotice{settingsFirst, settingsSecond}
	a.openRepository(nextRoot, true)
	runContinuation(t, done)
	if a.repo == nil || a.repo.Root != nextRoot || a.loadErr != "" {
		t.Fatalf("repository switch failed: %q", a.loadErr)
	}
	if got := a.notices[toastSettings]; len(got) != 2 || got[0] != settingsFirst || got[1] != settingsSecond {
		t.Fatal("repository switch lost or reordered independent Settings results")
	}
	if got := a.notices[toastMain]; len(got) != 1 || got[0] != independentMain {
		t.Fatal("repository switch lost independent main result or retained stale repository result")
	}
}

func TestSameRepositoryReopenPreservesItsQueuedResult(t *testing.T) {
	a, done, root := worktreeApp(t)
	notice := &toastNotice{root: root, kind: "error", title: "same repository result"}
	a.notices[toastMain] = []*toastNotice{notice}
	a.openRepository(root, true)
	runContinuation(t, done)
	if a.repo == nil || a.repo.Root != root || a.loadErr != "" {
		t.Fatalf("reopen failed: %q", a.loadErr)
	}
	if got := a.notices[toastMain]; len(got) != 1 || got[0] != notice {
		t.Fatal("harmless same-root reopen discarded valid repository result")
	}
}
