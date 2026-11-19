package lobedesk

import (
	"reflect"
	"strings"
	"testing"
)

func TestCommitMessageCommitSHAs(t *testing.T) {
	sha, other := strings.Repeat("ab", 20), strings.Repeat("cd", 20)
	for _, tc := range []struct {
		name, text string
		want       []string
	}{
		{"literal", "Revert " + sha + ".", []string{sha}},
		{"canonical-dedup", strings.ToUpper(sha) + " " + sha + " (" + other + ")", []string{sha, other}},
		{"not-substring", "g" + sha + " " + sha + "_suffix " + sha + strings.Repeat("0", 24), nil},
		{"no-short-or-revision", sha[:12] + " " + sha + "^ " + sha + "~1 " + sha + ":file", nil},
		{"revision-range", sha + ".." + other, nil},
		{"revision-symmetric-range", sha + "..." + other, nil},
		{"unicode-boundaries", "作者" + sha + " " + sha + "作者", nil},
		{"skip-url", "https://example.invalid/" + sha + "?other=" + other + " " + other, []string{other}},
		{"skip-url-host-and-fragment", "https://" + sha + "/x#" + other, nil},
		{"skip-other-uri-tokens", "file:///tmp/" + sha + " mailto:" + sha + "@example.invalid", nil},
		{"skip-uri-with-sha-only", "mailto:" + sha + " custom:" + other, nil},
		{"quoted", "‘" + sha + "’ and [" + other + "]", []string{sha, other}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := commitMessageCommitSHAs(tc.text); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("candidates: %v, want %v", got, tc.want)
			}
		})
	}
}
