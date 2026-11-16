package lobedesk

import (
	"strings"
	"testing"

	"supergit/internal/gitrepo"
)

func TestRebaseConflictMenusNameTheActualSides(t *testing.T) {
	for _, language := range []locale{localeEnglish, localeChinese} {
		a := &app{locale: language, operation: gitrepo.Operation{Kind: "rebase"}}
		ours, theirs := menuItem(t, a.conflictFileMenuItems(), "ours"), menuItem(t, a.conflictFileMenuItems(), "theirs")
		if language == localeEnglish {
			if !strings.Contains(ours.Label, "rebased base") || !strings.Contains(theirs.Label, "replayed commit") {
				t.Fatalf("rebase sides imply ordinary branch merge: ours=%q theirs=%q", ours.Label, theirs.Label)
			}
		} else if !strings.Contains(ours.Label, "变基基底") || !strings.Contains(theirs.Label, "重放提交") {
			t.Fatalf("rebase sides lack localized identities: ours=%q theirs=%q", ours.Label, theirs.Label)
		}
	}
}
