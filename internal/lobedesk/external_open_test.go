package lobedesk

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestExternalOpenPreservesPathsAsArguments(t *testing.T) {
	file := filepath.Join(t.TempDir(), "目录 with spaces;$(touch nope).txt")
	if err := os.WriteFile(file, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		os, kind string
		want     []string
	}{
		{"darwin", "open", []string{"open", file}},
		{"darwin", "reveal", []string{"open", "-R", file}},
		{"darwin", "terminal", []string{"open", "-a", "Terminal", filepath.Dir(file)}},
		{"linux", "open", []string{"xdg-open", file}},
		{"linux", "reveal", []string{"xdg-open", filepath.Dir(file)}},
		{"windows", "open", []string{"explorer.exe", file}},
		{"windows", "reveal", []string{"explorer.exe", "/select," + file}},
		{"windows", "terminal", []string{"cmd.exe", "/K"}},
	} {
		t.Run(tc.os+"/"+tc.kind, func(t *testing.T) {
			cmd, err := externalOpenCommand(tc.os, file, tc.kind)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(cmd.Args, tc.want) {
				t.Fatalf("launch arguments=%q want %q", cmd.Args, tc.want)
			}
			if tc.kind == "terminal" && tc.os == "windows" && cmd.Dir != filepath.Dir(file) {
				t.Fatalf("terminal directory=%q", cmd.Dir)
			}
		})
	}
	for _, system := range []string{"linux", "windows"} {
		cmd, err := externalOpenCommand(system, filepath.Dir(file), "reveal")
		if err != nil || cmd.Args[len(cmd.Args)-1] != filepath.Dir(file) {
			t.Fatalf("%s: directory reveal selected its parent: cmd=%v err=%v", system, cmd, err)
		}
	}
}

func TestExternalPathLabelsMatchPlatformBehavior(t *testing.T) {
	for _, tc := range []struct{ system, key, want string }{
		{"darwin", "revealInFinder", "revealInFinder"},
		{"windows", "revealInFinder", "showInFileManager"},
		{"linux", "revealInFinder", "openContainingFolder"},
		{"linux", "openInFinder", "openInFileManager"},
		{"windows", "openInTerminal", "openInTerminal"},
	} {
		if got := externalPathMessageKey(tc.system, tc.key); got != tc.want {
			t.Errorf("%s/%s label=%s want %s", tc.system, tc.key, got, tc.want)
		}
		for _, language := range []locale{localeEnglish, localeChinese} {
			if messages[language][tc.want] == "" {
				t.Errorf("missing %s translation for %s", language, tc.want)
			}
		}
	}
}

func TestExternalOpenRejectsUnsupportedActions(t *testing.T) {
	for _, tc := range []struct{ os, kind string }{{"plan9", "open"}, {"linux", "unexpected"}} {
		if _, err := externalOpenCommand(tc.os, t.TempDir(), tc.kind); err == nil {
			t.Fatalf("unsupported launch silently accepted: %+v", tc)
		}
	}
}

func TestLinuxTerminalUsesInstalledEmulatorWithoutShellInterpolation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Linux executable discovery uses a POSIX host fixture")
	}
	bin := t.TempDir()
	terminal := filepath.Join(bin, "gnome-terminal")
	if err := os.WriteFile(terminal, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	dir := filepath.Join(t.TempDir(), "目录;$(touch nope)")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	cmd, err := externalOpenCommand("linux", dir, "terminal")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Path != terminal || !reflect.DeepEqual(cmd.Args[1:], []string{"--working-directory", dir}) || cmd.Dir != dir {
		t.Fatalf("terminal command lost literal working directory: %+v", cmd)
	}
	if err := os.Remove(terminal); err != nil {
		t.Fatal(err)
	}
	if _, err := externalOpenCommand("linux", dir, "terminal"); err == nil {
		t.Fatal("missing terminal did not produce a recoverable launch error")
	}
}
