//go:build !legacydesktop

package main

import "supergit/internal/lobedesk"

func runApp(path string) error { return lobedesk.Run(path) }
