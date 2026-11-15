package lobedesk

import (
	"os"
	"strconv"
	"strings"
)

const (
	// defaultWindowWidth and defaultWindowHeight are the size a normal launch
	// opens at.
	defaultWindowWidth  = 1320
	defaultWindowHeight = 840
	// minWindowWidth and minWindowHeight bound the window, so a layout check
	// below them would test a size no reader can reach.
	minWindowWidth  = 980
	minWindowHeight = 640
)

// windowSizeEnv names the variable an automated run uses to choose its window
// size. The automation harness sends pointer events to the page and cannot
// drag the window's edge, so this is the only way to check layouts narrower
// than the default without a person resizing the window by hand. Unset or
// malformed leaves the default size alone, so a normal launch never sees it.
const windowSizeEnv = "SUPERGIT_WINDOW_SIZE"

// windowSize is the size the window opens at, taking windowSizeEnv into
// account.
func windowSize() (int, int) {
	if w, h, ok := parseWindowSize(os.Getenv(windowSizeEnv)); ok {
		return w, h
	}
	return defaultWindowWidth, defaultWindowHeight
}

// parseWindowSize reads a size written as "980x640". Sizes below the window's
// minimum are refused rather than clamped: a check meant for a narrow window
// would otherwise silently run at a different width than it asked for.
func parseWindowSize(raw string) (width, height int, ok bool) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(raw)), "x")
	if len(parts) != 2 {
		return 0, 0, false
	}
	width, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, false
	}
	height, err = strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, false
	}
	if width < minWindowWidth || height < minWindowHeight {
		return 0, 0, false
	}
	return width, height, true
}
