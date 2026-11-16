package gitrepo

import (
	"context"
	"errors"
	"os"
	"strings"
)

// Identity is the name and email Git uses, without dates or a display mailmap.
// These reads share commit's environment and repository routing; they never
// change configuration or supply an --author override to a later write.
type Identity struct {
	Name  string
	Email string
}

func (r *Repository) AuthorIdentity(ctx context.Context) (Identity, error) {
	// A plain commit finishing a conflicted cherry-pick reuses its source
	// author. git var alone only reports the environment's ordinary author.
	path, err := r.gitPath(ctx, "CHERRY_PICK_HEAD")
	if err != nil {
		return Identity{}, err
	}
	if _, err := os.Stat(path); err == nil {
		sha, err := r.resolve(ctx, "CHERRY_PICK_HEAD")
		if err != nil {
			return Identity{}, err
		}
		return r.gitIdentity(ctx, "show", "-s", "--format=%an <%ae>", sha)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Identity{}, err
	}
	return r.gitIdentity(ctx, "var", "GIT_AUTHOR_IDENT")
}

func (r *Repository) CommitterIdentity(ctx context.Context) (Identity, error) {
	return r.gitIdentity(ctx, "var", "GIT_COMMITTER_IDENT")
}

// HeadAuthorIdentity reads the raw author preserved by commit --amend, even
// when the reader is browsing another ref or GIT_AUTHOR_* has changed.
func (r *Repository) HeadAuthorIdentity(ctx context.Context) (Identity, error) {
	return r.gitIdentity(ctx, "show", "-s", "--format=%an <%ae>", "HEAD")
}

func (r *Repository) gitIdentity(ctx context.Context, args ...string) (Identity, error) {
	out, err := run(ctx, r.Root, "", args...)
	if err != nil {
		return Identity{}, err
	}
	// Git's ident suffix may contain a timestamp and timezone. The final angle
	// brackets delimit the email; names may contain spaces and punctuation.
	end := strings.LastIndex(out, ">")
	if end < 0 {
		return Identity{}, errors.New("Git identity has no email delimiter")
	}
	start := strings.LastIndex(out[:end], "<")
	if start < 0 {
		return Identity{}, errors.New("Git identity has no name delimiter")
	}
	identity := Identity{Name: strings.TrimSpace(out[:start]), Email: out[start+1 : end]}
	if identity.Name == "" || identity.Email == "" {
		return Identity{}, errors.New("Git identity requires a name and email")
	}
	return identity, nil
}
