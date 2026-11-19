//go:build e2e

package desktop

import "os"

func init() {
	if dir := os.Getenv("SUPERGIT_TEST_CONFIG_DIR"); dir != "" {
		desktopConfigDir = func() (string, error) { return dir, nil }
	}
}
