package lobedesk

import (
	"context"
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"supergit/internal/gitrepo"
)

type commitIdentityState struct {
	Root                                         string
	Loaded                                       bool
	Author, Committer, HeadAuthor                gitrepo.Identity
	AuthorError, CommitterError, HeadAuthorError string
}

func readCommitIdentity(ctx context.Context, repo *gitrepo.Repository, hasHead bool) commitIdentityState {
	state := commitIdentityState{Root: repo.Root, Loaded: true}
	var err error
	state.Author, err = repo.AuthorIdentity(ctx)
	if err != nil {
		state.AuthorError = err.Error()
	}
	state.Committer, err = repo.CommitterIdentity(ctx)
	if err != nil {
		state.CommitterError = err.Error()
	}
	if hasHead {
		state.HeadAuthor, err = repo.HeadAuthorIdentity(ctx)
		if err != nil {
			state.HeadAuthorError = err.Error()
		}
	}
	return state
}

func (a *app) applyCommitIdentity(seq uint64, root string, state commitIdentityState) {
	if a.repo == nil || a.repo.Root != root || seq != a.loadSeq || state.Root != root {
		return
	}
	a.commitIdentity = state
}

func (a *app) commitIdentityReady() bool {
	s := a.commitIdentity
	if a.repo == nil || !s.Loaded || s.Root != a.repo.Root || s.CommitterError != "" || s.Committer.Name == "" || s.Committer.Email == "" {
		return false
	}
	id, err := s.Author, s.AuthorError
	if a.amend {
		id, err = s.HeadAuthor, s.HeadAuthorError
	}
	return err == "" && id.Name != "" && id.Email != ""
}

func (a *app) commitIdentityView(c *ui.Context) {
	s := a.commitIdentity
	if a.loading || a.repo == nil || !s.Loaded || s.Root != a.repo.Root {
		display.Text(c, a.t("commitIdentityLoading"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)})
		return
	}
	id, identityError := s.Author, s.AuthorError
	if a.amend {
		id, identityError = s.HeadAuthor, s.HeadAuthorError
	}
	ui.Column(c).FillWidth().Gap(3).Selectable().Children(func() {
		if identityError == "" && id.Name != "" && id.Email != "" {
			display.Text(c, fmt.Sprintf("%s: %s <%s>", a.t("author"), id.Name, id.Email), display.TextProps{FontSize: scaledFont(11)})
		}
		if a.amend && s.CommitterError == "" && s.Committer != id {
			display.Text(c, fmt.Sprintf("%s: %s <%s>", a.t("committer"), s.Committer.Name, s.Committer.Email), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)})
		}
		if !a.commitIdentityReady() {
			detail := identityError
			if detail == "" {
				detail = s.CommitterError
			}
			display.Text(c, a.t("commitIdentityFailed"), display.TextProps{Type: display.TextWarning, FontSize: scaledFont(11)}).Tooltip(detail)
			display.Text(c, a.t("commitIdentityConfigHint"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(10)})
		}
	})
}
