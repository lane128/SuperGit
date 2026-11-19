package lobedesk

import (
	"testing"
	"time"

	"supergit/internal/gitrepo"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestHistorySubjectTooltipFollowsRefLayout(t *testing.T) {
	const subject = "并行提交记录说明甲乙丙丁戊己庚辛壬癸"
	commit := gitrepo.Commit{SHA: "tip", Subject: subject, Refs: "HEAD -> long-branch-name-for-this-commit, another-long-branch-name"}
	a := &app{hideAuthor: true, hideTime: true, showGraph: true, status: gitrepo.Status{Head: commit.SHA}}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		width, _ := c.Size()
		a.historyDetailsHidden, a.historySplit.rowWidth = true, width
		a.commitRow(c, commit, 0, 0, 1, &commitGraph{Width: 74})
	}, 309, 200)
	count := func() int {
		n := 0
		for _, text := range tt.Texts() {
			if text == subject {
				n++
			}
		}
		return n
	}
	// The row label and the title both report the subject before hovering.
	tt.Frame()
	baseline := count()
	// Start over the title, between the graph and the ref badges.
	tt.Move(123, a.historyRowHeight()/2)
	tt.Frame()
	tt.Advance(time.Second)
	tt.Frame()
	if n := count(); n != baseline+1 {
		t.Fatalf("truncated subject has no full-text tooltip; texts=%q", tt.Texts())
	}
	tt.SetSize(760, 200)
	tt.Frame()
	tt.Advance(time.Second)
	if n := count(); n != baseline {
		t.Fatalf("tooltip remains when the title fits; texts=%q", tt.Texts())
	}
}
