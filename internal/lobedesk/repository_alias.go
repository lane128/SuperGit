package lobedesk

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// A separate file per canonical checkout lets windows rename different tabs
// without rewriting each other's aliases or the window session.
func repositoryAliasPath(root string) (string, error) {
	settings, err := settingsPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(settings), "repository-aliases", fmt.Sprintf("%x.json", sha256.Sum256([]byte(root)))), nil
}

func readRepositoryAlias(root string) string {
	path, err := repositoryAliasPath(root)
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ""
	}
	var alias string
	if err == nil {
		err = json.Unmarshal(data, &alias)
	}
	if err != nil {
		log.Printf("could not read repository alias: %v", err)
		return ""
	}
	return strings.TrimSpace(checkoutCaptionText(alias))
}

func (tabs *repositoryTabs) askAlias(target *app) {
	if !tabs.canChange() || !tabs.canClose(target) || target.loading || target.repo == nil {
		return
	}
	tabs.activate(target, false)
	a := target
	root := target.repo.Root
	a.askTextPrompt(a.t("setRepositoryAlias"), a.t("repositoryAliasPlaceholder"), target.repositoryAlias, a.t("saveRepositoryAlias"), true, func(value string) {
		if tabs.findCheckout(root) != target {
			return
		}
		value = strings.TrimSpace(checkoutCaptionText(value))
		path, err := repositoryAliasPath(root)
		if err == nil {
			err = writeLocalJSON(path, "alias-*.json", value)
		}
		if err != nil {
			a.queue("error", a.t("saveRepositoryAliasFailed"), err.Error())
			a.invalidate()
			return
		}
		target.repositoryAlias = value
		tabs.reveal = true
		a.invalidate()
	})
}
