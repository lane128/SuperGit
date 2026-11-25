//go:build e2e

package lobedesk

import (
	"github.com/egoist/mygo"
	"os"
	"path/filepath"
)

// Native fixtures isolate app preferences without changing the process home.
func init() {
	if dir := os.Getenv("SUPERGIT_TEST_CONFIG_DIR"); dir != "" {
		settingsConfigDir = func() (string, error) { return dir, nil }
		// Window geometry is stored by MyGo separately from app preferences.
		// Keep native fixture windows out of the installed app's state too.
		mygo.App.SetPath(mygo.PathUserData, filepath.Join(dir, "mygo"))
	}
}
