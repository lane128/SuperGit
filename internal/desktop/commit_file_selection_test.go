package desktop

import (
	"supergit/internal/gitrepo"
	"testing"
)

func commitFileSelectionApp() *app {
	a := selectionHistoryApp()
	a.selectedSHA = a.data.history[0].SHA
	a.replaceCommitFiles([]gitrepo.ChangedFile{{Path: "a.txt", Status: "M"}, {Path: " odd\nfile.txt ", OriginalPath: "old name", Status: "R100"}})
	a.commitPath = "a.txt"
	return a
}
func TestCommitFilePresentationCopiesRecordsAndCachesIdentity(t *testing.T) {
	a := commitFileSelectionApp()
	m := a.commitFileModel()
	if m != a.commitFileModel() {
		t.Fatal("unchanged frame rebuilt file presentation")
	}
	f, ok := a.commitFileSelection(m, 1)
	if !ok || f.Path != " odd\nfile.txt " || f.OriginalPath != "old name" || f.Status != "R100" {
		t.Fatal("raw rename identity lost")
	}
	a.commitFiles[1].Path = "mutated"
	if m.files[1].Path != " odd\nfile.txt " {
		t.Fatal("model borrowed mutable records")
	}
	if _, ok := a.commitFileSelection(m, 1); ok {
		t.Fatal("mutated source accepted stale record")
	}
}
func TestCommitFileCursorPreservesDeliveredInputAndResetsAcrossCommits(t *testing.T) {
	a := commitFileSelectionApp()
	m := a.commitFileModel()
	var cursor commitFileCursor
	cursor.sync(m, a.commitPath)
	if cursor.row != 0 {
		t.Fatal("initial selection lost")
	}
	cursor.row = 1
	cursor.sync(m, a.commitPath)
	if cursor.row != 1 {
		t.Fatal("view overwrote pending row input")
	}
	a.commitPath = " odd\nfile.txt "
	cursor.sync(m, a.commitPath)
	if cursor.row != 1 {
		t.Fatal("raw path selection lost")
	}
	a.selectedSHA = a.data.history[1].SHA
	a.replaceCommitFiles(nil)
	a.commitPath = ""
	cursor.sync(a.commitFileModel(), a.commitPath)
	if cursor.row != -1 {
		t.Fatal("new empty commit selected an old file")
	}
	a.replaceCommitFiles([]gitrepo.ChangedFile{{Path: "b.txt", Status: "A"}})
	a.commitPath = "b.txt"
	cursor.sync(a.commitFileModel(), a.commitPath)
	if cursor.row != 0 {
		t.Fatal("replacement at same row lost")
	}
}
func TestCommitFileSelectionRejectsSupersededIdentityAndWrites(t *testing.T) {
	for _, change := range []string{"sha", "revision", "repository", "record", "writing", "opening", "page", "no repo"} {
		a := commitFileSelectionApp()
		m := a.commitFileModel()
		switch change {
		case "sha":
			a.selectedSHA = a.data.history[1].SHA
		case "revision":
			a.replaceCommitFiles(a.commitFiles)
		case "repository":
			a.guard.Switch()
		case "record":
			a.commitFiles[0].Status = "D"
		case "writing":
			a.guard.writing = true
		case "opening":
			a.opening = true
		case "page":
			a.page = "action"
		case "no repo":
			a.repo = nil
		}
		if _, ok := a.commitFileSelection(m, 0); ok {
			t.Fatalf("accepted %s", change)
		}
	}
	a := commitFileSelectionApp()
	a.loading = true
	a.commitLoading = true
	m := a.commitFileModel()
	if _, ok := a.commitFileSelection(m, 0); !ok {
		t.Fatal("read-only file browsing blocked by reads")
	}
	for _, row := range []int{-1, 2, 99} {
		if _, ok := a.commitFileSelection(m, row); ok {
			t.Fatal("invalid row accepted")
		}
	}
	if _, ok := a.commitFileSelection(nil, 0); ok {
		t.Fatal("nil presentation accepted")
	}
}
