package lobedesk

import (
	"context"
	"fmt"
	"sync"

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
	// The three identities are independent Git reads; together with the rest
	// of the open they would otherwise add three subprocess rounds in a row.
	var reads sync.WaitGroup
	reads.Go(func() {
		identity, err := repo.AuthorIdentity(ctx)
		if err != nil {
			state.AuthorError = err.Error()
			return
		}
		state.Author = identity
	})
	reads.Go(func() {
		identity, err := repo.CommitterIdentity(ctx)
		if err != nil {
			state.CommitterError = err.Error()
			return
		}
		state.Committer = identity
	})
	if hasHead {
		reads.Go(func() {
			identity, err := repo.HeadAuthorIdentity(ctx)
			if err != nil {
				state.HeadAuthorError = err.Error()
				return
			}
			state.HeadAuthor = identity
		})
	}
	reads.Wait()
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

func (a *app) commitIdentityHasExtraLines() bool {
	s := a.commitIdentity
	if a.loading || a.repo == nil || !s.Loaded || s.Root != a.repo.Root || !a.commitIdentityReady() {
		return true
	}
	id := s.Author
	if a.amend {
		id = s.HeadAuthor
	}
	return a.amend && s.CommitterError == "" && s.Committer != id
}

func (a *app) commitIdentityView(c *ui.Context) {
	a.commitIdentityViewInline(c, false)
}

func (a *app) commitIdentityViewInline(c *ui.Context, inline bool) {
	s := a.commitIdentity
	if a.loading || a.repo == nil || !s.Loaded || s.Root != a.repo.Root {
		display.Text(c, a.t("commitIdentityLoading"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11.5)})
		return
	}
	id, identityError := s.Author, s.AuthorError
	if a.amend {
		id, identityError = s.HeadAuthor, s.HeadAuthorError
	}
	if inline && identityError == "" && id.Name != "" && id.Email != "" && a.commitIdentityReady() {
		full := fmt.Sprintf("%s: %s <%s>", a.t("author"), id.Name, id.Email)
		ui.Row(c).Grow(1).Basis(0).MinWidth(0).AlignItems(ui.Center).Selectable().Children(func() {
			display.Text(c, full, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11.5)}).
				SingleLine().MinWidth(0).Shrink(1).Tooltip(full)
		})
		return
	}
	ui.Column(c).FillWidth().Gap(4).Selectable().Children(func() {
		if identityError == "" && id.Name != "" && id.Email != "" {
			full := fmt.Sprintf("%s: %s <%s>", a.t("author"), id.Name, id.Email)
			display.Text(c, full, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11.5)}).
				SingleLine().MinWidth(0).Tooltip(full)
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
