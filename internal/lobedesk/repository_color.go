package lobedesk

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

var repositoryColorPresets = []string{"default", "blue", "green", "amber", "rose", "violet", "gray"}

func normalizedRepositoryColor(color string) string {
	if slices.Contains(repositoryColorPresets, color) {
		return color
	}
	return "default"
}

// Callers use Repository.Root, canonicalized by Git open, just as aliases do.
// Independent files keep windows editing different checkouts from losing data.
func repositoryColorPath(root string) (string, error) {
	settings, err := settingsPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(settings), "repository-colors", fmt.Sprintf("%x.json", sha256.Sum256([]byte(root)))), nil
}

func readRepositoryColor(root string) string {
	path, err := repositoryColorPath(root)
	if err != nil {
		return "default"
	}
	data, err := os.ReadFile(path)
	var color string
	if err != nil || json.Unmarshal(data, &color) != nil {
		return "default"
	}
	return normalizedRepositoryColor(color)
}

func (tabs *repositoryTabs) canColorRepository(target *app, root string) bool {
	return target != nil && target.repo != nil && target.repo.Root == root &&
		tabs.findCheckout(root) == target && tabs.canChange() && tabs.canClose(target) && !target.loading
}

func (tabs *repositoryTabs) setRepositoryColor(target *app, root, color string) {
	if !slices.Contains(repositoryColorPresets, color) || !tabs.canColorRepository(target, root) {
		return
	}
	path, err := repositoryColorPath(root)
	if err == nil {
		err = writeLocalJSON(path, "color-*.json", color)
	}
	if err != nil {
		tabs.active.queue("error", tabs.active.t("saveRepositoryColorFailed"), err.Error())
		tabs.active.invalidate()
		return
	}
	target.repositoryColor = color
	tabs.active.invalidate()
}
