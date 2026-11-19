//go:build legacydesktop

package main

import "supergit/internal/desktop"

func runApp(path string) error { return desktop.Run(path) }
