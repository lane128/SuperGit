package lobedesk

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

const recentRepositoryLimit = 12

type repositoryRecents struct {
	Roots []string `json:"roots"`
}

func recentRepositoriesPath(sessionPath string) string {
	return filepath.Join(filepath.Dir(sessionPath), "repository-recents.json")
}

func readRecentRepositories(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var recents repositoryRecents
	if err := json.Unmarshal(data, &recents); err != nil {
		return nil, err
	}
	return validRecentRoots(recents.Roots), nil
}

func legacyRecentRepositories(sessionPath string) []string {
	// Both applications store their files under the same OS config root.
	data, err := os.ReadFile(legacyDesktopPreferencesPath(sessionPath))
	if err != nil {
		return nil
	}
	var legacy struct{ Recent []string }
	if json.Unmarshal(data, &legacy) != nil {
		return nil
	}
	return validRecentRoots(legacy.Recent)
}

func validRecentRoots(candidates []string) []string {
	roots := make([]string, 0, min(len(candidates), recentRepositoryLimit))
	for _, root := range candidates {
		if filepath.IsAbs(root) && filepath.Clean(root) == root && !slices.Contains(roots, root) {
			roots = append(roots, root)
			if len(roots) == recentRepositoryLimit {
				break
			}
		}
	}
	return roots
}

func (tabs *repositoryTabs) saveRecentRepository(root string, local []string) []string {
	state := &tabs.sessionState
	roots, err := updateStoredRecentRepositories(recentRepositoriesPath(state.path), root, local)
	if err != nil {
		state.savedRecentHead = ""
		log.Printf("could not save recent repositories: %v", err)
		return local
	}
	state.savedRecentHead = root
	return roots
}

// recentStoreMu serializes saves from this process. flock is taken per open
// file description, so two windows saving at once can see each other as
// foreign holders even inside one app.
var recentStoreMu sync.Mutex

// A save waits briefly for another window's write to finish. The lock is held
// for one small file, so the wait is short; a genuinely stuck writer still
// surfaces as an error after these attempts.
const (
	recentLockAttempts = 5
	recentLockWait     = 20 * time.Millisecond
)

func updateStoredRecentRepositories(path, root string, local []string) ([]string, error) {
	recentStoreMu.Lock()
	defer recentStoreMu.Unlock()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	// Serialize read, merge and rename on a stable sibling of the JSON file.
	// The lock is released by the OS if SuperGit exits unexpectedly.
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	defer lock.Close()
	var lockErr error
	for attempt := 0; attempt < recentLockAttempts; attempt++ {
		if lockErr = lockRepositoryRecents(lock); lockErr == nil {
			break
		}
		time.Sleep(recentLockWait)
	}
	if lockErr != nil {
		return nil, lockErr
	}
	saved, err := readRecentRepositories(path)
	if err != nil {
		return nil, err
	}
	roots := appendRecentRepositories([]string{root}, saved)
	roots = appendRecentRepositories(roots, local)
	if !slices.Equal(roots, saved) {
		if err := writeLocalJSON(path, "repository-recents-*.json", repositoryRecents{Roots: roots}); err != nil {
			return nil, err
		}
	}
	return roots, nil
}

func appendRecentRepositories(roots, candidates []string) []string {
	for _, root := range candidates {
		if len(roots) == recentRepositoryLimit {
			break
		}
		if !slices.Contains(roots, root) {
			roots = append(roots, root)
		}
	}
	return roots
}
