//go:build !darwin

package updater

import "errors"

// Eligibility describes whether the running SuperGit instance can perform an
// in-place atomic replacement.
type Eligibility struct {
	Eligible   bool
	BundlePath string
	Reason     string
}

const (
	ReasonNotAppBundle  = "notAppBundle"
	ReasonTranslocated  = "translocated"
	ReasonNotWritable   = "notWritable"
	ReasonUnsupportedOS = "unsupportedOS"
)

func CheckEligibility(string) Eligibility {
	return Eligibility{Reason: ReasonUnsupportedOS}
}

func StageAndLaunchApply(string, string, int) error {
	return errors.New("in-place automatic update is currently supported on macOS only")
}

func RunApplyHelper([]string) error {
	return errors.New("in-place automatic update is currently supported on macOS only")
}
