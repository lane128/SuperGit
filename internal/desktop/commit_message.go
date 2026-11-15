package desktop

import (
	"context"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type commitMessageState struct {
	sha, message    string
	loading, loaded bool
	err             error
}

func (s commitMessageState) parts(sha, fallback string) (string, string) {
	if s.sha != sha || !s.loaded || s.err != nil {
		return fallback, ""
	}
	title, body, _ := strings.Cut(s.message, "\n")
	// The layout supplies the subject/body separator; leave other message
	// whitespace intact, including a continued first paragraph and trailers.
	return title, strings.TrimPrefix(body, "\n")
}

func (a *app) cancelCommitMessageRead() {
	if a.commitTextCancel != nil {
		a.commitTextCancel()
		a.commitTextCancel = nil
	}
	a.commitText.loading = false
	a.guard.Begin("commit-message")
}

func (a *app) beginCommitMessage(sha string) (ticket, context.Context, bool) {
	if a.closed || a.repo == nil || a.guard.writing || a.opening || sha == "" || sha != a.selectedSHA {
		return ticket{}, nil, false
	}
	a.cancelCommitMessageRead()
	ctx, cancel := context.WithCancel(a.ctx)
	a.commitTextCancel = cancel
	a.commitText = commitMessageState{sha: sha, loading: true}
	return a.guard.Begin("commit-message"), ctx, true
}

func (a *app) finishCommitMessage(req ticket, sha, message string, err error) bool {
	if a.closed || !a.guard.Accept(req) || sha != a.selectedSHA || a.commitText.sha != sha || !a.commitText.loading {
		return false
	}
	if a.commitTextCancel != nil {
		a.commitTextCancel()
		a.commitTextCancel = nil
	}
	if err != nil {
		message = ""
		a.record("读取提交说明失败", err)
	}
	a.commitText = commitMessageState{sha: sha, message: message, loaded: true, err: err}
	return true
}

func (a *app) loadCommitMessage(sha string) {
	req, ctx, ok := a.beginCommitMessage(sha)
	if !ok {
		return
	}
	r, cancel := a.repo, a.commitTextCancel
	go func() {
		defer cancel()
		message, err := r.CommitMessage(ctx, sha)
		mygo.RunOnMain(func() {
			if a.finishCommitMessage(req, sha, message, err) {
				a.invalidate()
			}
		})
	}()
}

func (a *app) commitMessageView(c *ui.Context, sha, body string) {
	t := c.Theme()
	s := a.commitText
	if s.sha == sha && s.loading {
		ui.Text(c, "正在读取提交说明…").FontSize(12).TextColor(t.TextMuted)
		return
	}
	if s.sha != sha || !s.loaded || s.err != nil {
		if s.err != nil && s.sha == sha {
			ui.Text(c, "提交说明读取失败："+s.err.Error()).FontSize(12).TextColor(t.Danger).FillWidth()
		}
		if ui.Button(c, "重试读取提交说明").Disabled(a.busy()).Clicked() {
			a.loadCommitMessage(sha)
		}
		return
	}
	if body != "" {
		ui.Text(c, body).FontSize(13).Selectable().FillWidth()
	}
}
