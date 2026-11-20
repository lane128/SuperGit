package lobedesk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func externalPathMessageKey(system, key string) string {
	if system == "darwin" {
		return key
	}
	switch key {
	case "openInFinder":
		return "openInFileManager"
	case "revealInFinder":
		if system == "linux" {
			return "openContainingFolder"
		}
		return "showInFileManager"
	}
	return key
}

// externalOpenCommand keeps paths out of shell source. Linux reveal opens the
// containing folder because xdg-open has no portable select-file operation.
func externalOpenCommand(system, full, kind string) (*exec.Cmd, error) {
	if kind == "sublimeText" {
		if system == "darwin" {
			return exec.Command("open", "-a", "Sublime Text", full), nil
		}
		binary, err := exec.LookPath("subl")
		if err != nil {
			return nil, fmt.Errorf("Sublime Text (subl): %w", err)
		}
		return exec.Command(binary, full), nil
	}
	if kind != "open" && kind != "reveal" && kind != "terminal" {
		return nil, fmt.Errorf("unsupported path action: %s", kind)
	}
	directory := full
	if info, err := os.Stat(full); err != nil || !info.IsDir() {
		directory = filepath.Dir(full)
	}
	switch system {
	case "darwin":
		switch kind {
		case "reveal":
			return exec.Command("open", "-R", full), nil
		case "terminal":
			return exec.Command("open", "-a", "Terminal", directory), nil
		default:
			return exec.Command("open", full), nil
		}
	case "windows":
		switch kind {
		case "terminal":
			cmd := exec.Command("cmd.exe", "/K")
			cmd.Dir = directory
			return cmd, nil
		case "reveal":
			if info, err := os.Stat(full); err != nil || !info.IsDir() {
				return exec.Command("explorer.exe", "/select,"+full), nil
			}
		}
		return exec.Command("explorer.exe", full), nil
	case "linux":
		switch kind {
		case "terminal":
			for _, candidate := range []struct {
				name string
				args []string
			}{
				{"gnome-terminal", []string{"--working-directory", directory}},
				{"konsole", []string{"--workdir", directory}},
				{"xfce4-terminal", []string{"--working-directory", directory}},
				{"x-terminal-emulator", nil},
				{"xterm", nil},
			} {
				if executable, err := exec.LookPath(candidate.name); err == nil {
					cmd := exec.Command(executable, candidate.args...)
					cmd.Dir = directory
					return cmd, nil
				}
			}
			return nil, fmt.Errorf("no supported terminal found (gnome-terminal, konsole, xfce4-terminal, x-terminal-emulator or xterm)")
		case "reveal":
			return exec.Command("xdg-open", directory), nil
		default:
			return exec.Command("xdg-open", full), nil
		}
	}
	return nil, fmt.Errorf("unsupported desktop platform: %s", system)
}
