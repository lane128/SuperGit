package desktop

import (
	"errors"
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestFailedHistoryRefreshRetainsLoadedRowsAndRequiresRetry(t *testing.T) {
	a := pagedHistoryApp()
	before := a.data
	beforeDiff, beforeSHA, beforePath := a.commitDiff, a.selectedSHA, a.commitPath
	failure := errors.New("history output exceeded limit")
	installed := a.installSnapshot(snapshot{status: gitrepo.Status{Branch: "refreshed-workspace"}, historyErr: failure})
	if !reflect.DeepEqual(a.data.history, before.history) || !reflect.DeepEqual(a.data.historyTips, before.historyTips) || a.data.historyMore != before.historyMore {
		t.Fatal("failed history refresh discarded loaded rows or traversal")
	}
	if installed.status.Branch != "refreshed-workspace" || installed.historyErr != failure || !reflect.DeepEqual(installed.history, before.history) {
		t.Fatal("history failure suppressed valid workspace status or was not returned to reconciliation")
	}
	if a.historyLoadLabel() != "历史读取失败，请刷新" || a.canLoadHistory() {
		t.Fatal("history failure was reported as completion or allowed paging a stale refresh")
	}
	if a.selectedSHA != beforeSHA || a.commitPath != beforePath || !reflect.DeepEqual(a.commitDiff, beforeDiff) {
		t.Fatal("failed refresh changed immutable preview")
	}
	// A successful refresh clears the error and replaces even an empty history.
	a.installSnapshot(before)
	if a.historyLoadLabel() != "继续加载 300 条" || !a.canLoadHistory() {
		t.Fatal("successful retry did not restore continuation")
	}
	a.installSnapshot(snapshot{})
	if len(a.data.history) != 0 || a.historyLoadLabel() != "已到历史末尾" {
		t.Fatal("successful empty refresh retained stale rows")
	}
}

func TestInitialHistoryReadFailureIsNotEmptyRepository(t *testing.T) {
	a := pagedHistoryApp()
	a.data = snapshot{}
	a.installSnapshot(snapshot{historyErr: errors.New("history failed")})
	if a.historyLoadLabel() != "历史读取失败，请刷新" || a.canLoadHistory() {
		t.Fatal("initial read failure was labeled as exhausted history")
	}
}

func TestHistorySearchCallbackDefersNativeReset(t *testing.T) {
	a := pagedHistoryApp()
	a.historyList.ScrollTo(500, ui.Start)
	before := a.historyList
	a.historySearchChanged()
	if !a.historyListReset || !reflect.DeepEqual(a.historyList, before) {
		t.Fatal("search callback destroyed constructed list instead of requesting a view-time reset")
	}
}
