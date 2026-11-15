package lobedesk

import (
	"testing"

	"supergit/internal/gitrepo"
)

// Every letter Git can answer with gets a label and a color: a verified
// signature reads as success, a wrong or unusable one as an error, an unknown
// trust as information, and a check Git could not run as neutral.
func TestSignatureBadgesCoverEveryStatus(t *testing.T) {
	a := &app{locale: localeChinese}
	cases := []struct {
		status, label, color string
	}{
		{"G", "已验证签名", "success"},
		{"U", "签名有效（信任未知）", "info"},
		{"B", "签名无效", "error"},
		{"X", "签名已过期", "error"},
		{"Y", "签名密钥已过期", "error"},
		{"R", "签名密钥已吊销", "error"},
		{"E", "无法验证签名", "gray"},
	}
	for _, tt := range cases {
		if got := a.signatureLabel(tt.status); got != tt.label {
			t.Errorf("signatureLabel(%q) = %q; want %q", tt.status, got, tt.label)
		}
		if got := signatureTagColor(tt.status); got != tt.color {
			t.Errorf("signatureTagColor(%q) = %q; want %q", tt.status, got, tt.color)
		}
	}
	// An unsigned commit has no badge, and no letter Git did not define.
	if (gitrepo.Signature{Status: "N"}).Signed() {
		t.Error("an unsigned commit asked for a badge")
	}
}

// The verification answer belongs to one commit. Moving the selection clears
// it, so the next commit never shows the previous one's badge while its own
// answer is on the way.
func TestSignatureClearsWhenTheSelectionChanges(t *testing.T) {
	a := &app{selectedSHA: "aaaaaaa", commitSig: gitrepo.Signature{Status: "G", Signer: "Ada"}}
	a.setSelectedCommit("bbbbbbb")
	if a.commitSig.Signed() {
		t.Errorf("signature = %#v after the selection moved; want nothing", a.commitSig)
	}
	a.commitSig = gitrepo.Signature{Status: "G"}
	a.setSelectedCommit("bbbbbbb")
	if !a.commitSig.Signed() {
		t.Error("selecting the same commit again dropped its signature")
	}
}
