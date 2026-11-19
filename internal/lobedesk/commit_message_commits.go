package lobedesk

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"supergit/internal/gitrepo"
)

var commitMessageSHAPattern = regexp.MustCompile(`(?i)[0-9a-f]{40}`)
var commitMessageURIPattern = regexp.MustCompile("(?i)[a-z][a-z0-9+.-]*:[^\\s\\p{Z}<>\"`\\x00-\\x1f\\x7f]+")

// Only standalone complete SHA-1 tokens qualify. Revision expressions and
// hashes embedded in URLs remain plain selectable message text.
func commitMessageCommitSHAs(text string) []string {
	urls := commitMessageURIPattern.FindAllStringIndex(text, -1)
	var result []string
	seen := map[string]bool{}
	nextURL := 0
	for _, span := range commitMessageSHAPattern.FindAllStringIndex(text, -1) {
		for nextURL < len(urls) && urls[nextURL][1] <= span[0] {
			nextURL++
		}
		if nextURL < len(urls) && urls[nextURL][0] <= span[0] && span[0] < urls[nextURL][1] {
			continue
		}
		if span[0] > 0 {
			r, _ := utf8.DecodeLastRuneInString(text[:span[0]])
			if !unicode.IsSpace(r) && !strings.ContainsRune("([{\"'‘“:;,", r) {
				continue
			}
		}
		if strings.HasPrefix(text[span[1]:], "..") {
			continue
		}
		if span[1] < len(text) {
			r, _ := utf8.DecodeRuneInString(text[span[1]:])
			if !unicode.IsSpace(r) && !strings.ContainsRune(")]}\"'’”.,;!", r) {
				continue
			}
		}
		sha := strings.ToLower(text[span[0]:span[1]])
		if !seen[sha] {
			seen[sha] = true
			result = append(result, sha)
		}
	}
	return result
}

type messageCommitOrigin struct {
	repo                  *gitrepo.Repository
	root                  string
	epoch                 uint64
	reload                int
	source, query, pinned string
	ref                   historyRef
	menu                  uint64
}

type messageCommitRequest struct {
	origin messageCommitOrigin
	target string
}

func (a *app) messageCommitOrigin() messageCommitOrigin {
	o := messageCommitOrigin{repo: a.repo, epoch: a.loadSeq, reload: a.historyReload, source: a.selectedSHA, query: a.historyQuery, ref: a.historyRef, pinned: a.historyPinned, menu: a.messageMenuGeneration}
	if a.repo != nil {
		o.root = a.repo.Root
	}
	return o
}

func (a *app) messageCommitReady(o messageCommitOrigin) bool {
	return o.repo != nil && o == a.messageCommitOrigin() && a.page == "history" && !a.historyNavigationBlocked() && a.historyNavigationVisible() && a.pendingJumpSHA == "" && a.historyNavigation.pending == nil
}

// jumpToCommit clears the source search while paging. Every other captured
// identity must remain current until this message transaction selects a row.
func (a *app) messageCommitPagingCurrent(o *messageCommitOrigin) bool {
	if o == nil {
		return true
	}
	current := a.messageCommitOrigin()
	current.query = o.query
	return *o == current && a.page == "history"
}

func (a *app) requestMessageCommit(o messageCommitOrigin, sha string) {
	if !a.messageCommitReady(o) || len(sha) != 40 || commitMessageSHAPattern.FindString(sha) != sha {
		return
	}
	request := &messageCommitRequest{origin: o, target: sha}
	a.messageCommitRequest = request
	ctx := a.historyTraversalContext()
	go func() {
		resolved, err := o.repo.ResolveRef(ctx, sha)
		a.hopToMain(func() {
			if a.messageCommitRequest != request {
				return
			}
			a.messageCommitRequest = nil
			if !a.messageCommitReady(o) || ctx.Err() != nil || errors.Is(err, context.Canceled) {
				return
			}
			if err != nil || resolved != sha {
				a.queue("error", a.t("jumpMessageCommitFailed"), a.tf("messageCommitUnavailable", sha))
				a.invalidate()
				return
			}
			a.jumpWithHistoryTarget(sha, &request.origin)
		})
	}()
}

// Keep object-resolution errors separate from scope exhaustion and interrupted
// reads. A message transaction retains its full target while paging.
func (a *app) commitJumpNotice(sha string, blocked bool) {
	p := a.historyNavigation.pending
	if p != nil && p.target == sha && p.messageOrigin != nil {
		description := a.tf("messageCommitOutsideHistory", sha)
		if blocked {
			description = sha + "\n" + a.t("commitNavigationBlocked")
		}
		a.historyQuery = p.source.query
		a.historyRowsSet = false
		a.queue("warning", a.t("jumpMessageCommit"), description)
		a.cancelPendingHistoryNavigation()
		return
	}
	if blocked {
		a.queue("warning", a.t("jumpToCommit"), a.t("commitNavigationBlocked"))
	} else {
		a.queue("warning", a.t("commitNotLoaded"), shortSHA(sha))
	}
}
