package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Git dispatches helpers from its exec path and then PATH. GUI launches on
// macOS may omit Homebrew from PATH. Sublime Merge also distributes AVH with
// the companion scripts it sources; keep that installation together.
type extensionBinary struct {
	present bool
	execDir string
}

func findExtension(name string) extensionBinary {
	if _, err := exec.LookPath("git-" + name); err == nil {
		return extensionBinary{present: true}
	}
	execPath, _ := run(context.Background(), "", "", "--exec-path")
	dirs := []string{strings.TrimSpace(execPath)}
	if runtime.GOOS == "darwin" {
		dirs = append(dirs, "/opt/homebrew/bin", "/usr/local/bin")
		if name == "flow" {
			dirs = append(dirs, "/Applications/Sublime Merge.app/Contents/Resources/Git/git/libexec/git-core")
		}
	}
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		if _, err := exec.LookPath(filepath.Join(dir, "git-"+name)); err == nil {
			return extensionBinary{true, dir}
		}
	}
	return extensionBinary{}
}

var flowBinary = sync.OnceValue(func() extensionBinary { return findExtension("flow") })
var lfsBinary = sync.OnceValue(func() extensionBinary { return findExtension("lfs") })

func HasGitFlow() bool { return flowBinary().present }
func HasGitLFS() bool  { return lfsBinary().present }

func (b extensionBinary) run(ctx context.Context, dir, input string, args ...string) (string, error) {
	if !b.present {
		return "", errors.New("Git extension is not installed")
	}
	if b.execDir != "" {
		args = append([]string{"--exec-path=" + b.execDir}, args...)
	}
	return run(ctx, dir, input, args...)
}

func (r *Repository) checkPaletteStatus(ctx context.Context, expected Status) error {
	if err := r.idle(ctx); err != nil {
		return err
	}
	s, err := r.Status(ctx)
	if err != nil {
		return err
	}
	if s.Branch != expected.Branch || s.Head != expected.Head {
		return ErrIntegrationTargetChanged
	}
	return nil
}

func (r *Repository) ApplyPatchFile(ctx context.Context, path string, expected Status) error {
	if !filepath.IsAbs(path) {
		return errors.New("Choose an absolute patch file path")
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("Choose a regular patch file")
	}
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.checkPaletteStatus(ctx, expected); err != nil {
		return err
	}
	// git apply validates the whole patch before changing the working tree.
	// Its ordinary path protections and all-or-nothing behavior stay enabled.
	_, err = run(ctx, r.Root, "", "apply", "--", path)
	return err
}

func (r *Repository) AddPaletteSubmodule(ctx context.Context, address, name, path string, expected Status) error {
	if err := validURL(address); err != nil {
		return err
	}
	if !filepath.IsLocal(path) || filepath.Clean(path) != path || path == "." || strings.ContainsAny(path, "\r\n\x00") {
		return errors.New("Choose a path inside this repository")
	}
	if name == "" || name == "." || name == ".." || strings.HasPrefix(name, "-") || strings.ContainsAny(name, "\\\r\n\x00") {
		return errors.New("Invalid submodule name")
	}
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if strings.EqualFold(part, ".git") {
			return errors.New("Invalid submodule path")
		}
	}
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.checkPaletteStatus(ctx, expected); err != nil {
		return err
	}
	_, err := run(ctx, r.Root, "", "submodule", "add", "--name", name, "--", address, path)
	return err
}

func (r *Repository) OpenPaletteSubmodule(ctx context.Context, expected Submodule) (string, error) {
	modules, err := r.Submodules(ctx)
	if err != nil {
		return "", err
	}
	for _, m := range modules {
		if m.Path != expected.Path || m.Name != expected.Name || m.Index != expected.Index || !m.Initialized || m.Error != "" {
			continue
		}
		dir := filepath.Join(r.Root, m.Path)
		rel, err := filepath.Rel(canonical(r.Root), canonical(dir))
		if err == nil && filepath.IsLocal(rel) && rel != "." {
			return dir, nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrSubmoduleTargetChanged, expected.Path)
}
