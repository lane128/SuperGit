//go:build e2e

package lobedesk

import "os"

// Native fixtures isolate app preferences without changing the process home.
func init() {
	if dir := os.Getenv("SUPERGIT_TEST_CONFIG_DIR"); dir != "" {
		settingsConfigDir = func() (string, error) { return dir, nil }
	}
}
