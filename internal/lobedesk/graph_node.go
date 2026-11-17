package lobedesk

import "supergit/internal/gitrepo"

// HEAD is repository identity, independent of the selected history row.
func graphNodeShape(commit gitrepo.Commit, head string, row commitGraphRow) string {
	if head != "" && commit.SHA == head {
		return "head"
	}
	if len(commit.Parents) > 1 {
		first := commit.Parents[0]
		for _, parent := range commit.Parents[1:] {
			if parent != first {
				return "merge"
			}
		}
	}
	if row.Fork {
		return "fork"
	}
	return "dot"
}
