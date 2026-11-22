//go:build darwin

package updater

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Eligibility describes whether the running SuperGit instance can perform an
// in-place atomic .app replacement.
type Eligibility struct {
	Eligible   bool
	BundlePath string
	Reason     string // machine-readable reason code when Eligible is false
}

const (
	ReasonNotAppBundle  = "notAppBundle"
	ReasonTranslocated  = "translocated"
	ReasonNotWritable   = "notWritable"
	ReasonUnsupportedOS = "unsupportedOS"
)

// CheckEligibility inspects exePath to verify it runs from a writable,
// non-translocated macOS .app bundle.
func CheckEligibility(exePath string) Eligibility {
	if strings.TrimSpace(exePath) == "" {
		return Eligibility{Reason: ReasonNotAppBundle}
	}
	resolved, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		resolved = exePath
	}
	resolved = filepath.Clean(resolved)
	if strings.Contains(resolved, "/AppTranslocation/") {
		return Eligibility{Reason: ReasonTranslocated}
	}
	bundlePath, ok := enclosingAppBundle(resolved)
	if !ok {
		return Eligibility{Reason: ReasonNotAppBundle}
	}
	parent := filepath.Dir(bundlePath)
	if !dirWritable(parent) {
		return Eligibility{BundlePath: bundlePath, Reason: ReasonNotWritable}
	}
	return Eligibility{Eligible: true, BundlePath: bundlePath}
}

func enclosingAppBundle(exePath string) (string, bool) {
	macosDir := filepath.Dir(exePath)
	if filepath.Base(macosDir) != "MacOS" {
		return "", false
	}
	contentsDir := filepath.Dir(macosDir)
	if filepath.Base(contentsDir) != "Contents" {
		return "", false
	}
	bundleDir := filepath.Dir(contentsDir)
	if !strings.HasSuffix(bundleDir, ".app") {
		return "", false
	}
	return bundleDir, true
}

func dirWritable(dir string) bool {
	f, err := os.CreateTemp(dir, ".supergit-update-probe-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}

// StageAndLaunchApply moves or copies stagedApp into a hidden sibling next to
// targetApp and launches the new binary in detached --apply-update mode so it
// can atomically swap the bundles once the current process exits.
func StageAndLaunchApply(stagedApp, targetApp string, waitPID int) error {
	if err := VerifyAppBundleMetadata(stagedApp, ExpectedBundleID); err != nil {
		return err
	}
	parent := filepath.Dir(targetApp)
	base := filepath.Base(targetApp)
	sibling := filepath.Join(parent, fmt.Sprintf(".%s.update-staged-%d", base, waitPID))
	_ = os.RemoveAll(sibling)

	if err := moveAppBundle(stagedApp, sibling); err != nil {
		return fmt.Errorf("stage update beside target app: %w", err)
	}
	exeName, err := bundleExecutableName(sibling)
	if err != nil {
		_ = os.RemoveAll(sibling)
		return err
	}
	helperBin := filepath.Join(sibling, "Contents", "MacOS", exeName)
	cmd := exec.Command(helperBin,
		"--apply-update",
		"--pid", fmt.Sprintf("%d", waitPID),
		"--staged", sibling,
		"--target", targetApp,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(sibling)
		return fmt.Errorf("launch update helper: %w", err)
	}
	return nil
}

func moveAppBundle(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	cmd := exec.Command("ditto", src, dst)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ditto failed: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	_ = os.RemoveAll(src)
	return nil
}

func bundleExecutableName(appPath string) (string, error) {
	data, err := os.ReadFile(filepath.Join(appPath, "Contents", "Info.plist"))
	if err != nil {
		return "", err
	}
	m := plistExecutableRegexp.FindSubmatch(data)
	if m == nil {
		return "", errors.New("missing CFBundleExecutable")
	}
	name := strings.TrimSpace(string(m[1]))
	if name == "" || strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("invalid CFBundleExecutable %q", name)
	}
	return name, nil
}

// RunApplyHelper executes the detached --apply-update workflow.
func RunApplyHelper(args []string) error {
	fs := flag.NewFlagSet("apply-update", flag.ContinueOnError)
	pid := fs.Int("pid", 0, "PID of the running app to wait for")
	staged := fs.String("staged", "", "staged .app bundle path")
	target := fs.String("target", "", "target .app bundle path to replace")
	noRelaunch := fs.Bool("no-relaunch", false, "skip launching the replaced app (for tests)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return applyBundleUpdate(*pid, *staged, *target, !*noRelaunch, 30*time.Second, relaunchApp)
}

func applyBundleUpdate(pid int, staged, target string, relaunch bool, waitTimeout time.Duration, launch func(string) error) error {
	staged = filepath.Clean(staged)
	target = filepath.Clean(target)
	if !filepath.IsAbs(staged) || !filepath.IsAbs(target) {
		return errors.New("staged and target paths must be absolute")
	}
	if !strings.HasSuffix(target, ".app") {
		return fmt.Errorf("target %q is not a .app bundle", target)
	}
	if filepath.Dir(staged) != filepath.Dir(target) {
		return errors.New("staged bundle must reside in the same directory as target")
	}
	if staged == target {
		return errors.New("staged and target paths must differ")
	}
	if err := VerifyAppBundleMetadata(staged, ExpectedBundleID); err != nil {
		return err
	}
	if pid > 0 {
		if err := waitForProcessExit(pid, waitTimeout); err != nil {
			return err
		}
	}

	backup := filepath.Join(filepath.Dir(target), "."+filepath.Base(target)+".update-backup")
	_ = os.RemoveAll(backup)

	targetExisted := false
	if _, err := os.Stat(target); err == nil {
		targetExisted = true
		if err := os.Rename(target, backup); err != nil {
			return fmt.Errorf("backup current bundle: %w", err)
		}
	}
	if err := os.Rename(staged, target); err != nil {
		if targetExisted {
			_ = os.Rename(backup, target)
			if relaunch && launch != nil {
				_ = launch(target)
			}
		}
		return fmt.Errorf("swap staged bundle into place: %w", err)
	}
	if targetExisted {
		_ = os.RemoveAll(backup)
	}
	if relaunch && launch != nil {
		return launch(target)
	}
	return nil
}

func waitForProcessExit(pid int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		err := syscall.Kill(pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for process %d to exit", pid)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func relaunchApp(appPath string) error {
	return exec.Command("open", "-n", appPath).Start()
}
