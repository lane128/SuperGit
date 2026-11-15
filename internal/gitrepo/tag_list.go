package gitrepo

import (
	"context"
	"fmt"
	"strings"
)

// Tag preserves the reference object separately from its peeled target.
// Git recursively peels annotated tags, including nested tags, in one query.
// Commit is populated only for commit targets; tree/blob tags stay visible.
type Tag struct {
	Name, ObjectSHA, ObjectType string
	TargetSHA, TargetType       string
	SymbolicRef                 string
	// Message is the tag object's own message, including its body. Only
	// annotated tags carry one; a lightweight tag points straight at its
	// target, so Message stays empty rather than repeating the commit's
	// subject.
	Message string
	Commit  Commit
}

// Tags returns refs exposed by Git's resolved enumeration. Git silently skips
// dangling symbolic refs; callers must not present this as a raw ref inventory.
func (r *Repository) Tags(ctx context.Context) ([]Tag, error) {
	format := "%(refname:strip=2)%00%(objectname)%00%(objecttype)%00%(*objectname)%00%(*objecttype)%00%(symref)%00%(parent)%00%(subject)%00%(authorname)%00%(authordate:iso-strict)%00%(*parent)%00%(*subject)%00%(*authorname)%00%(*authordate:iso-strict)%00%(contents)%00"
	out, err := run(ctx, r.Root, "", "for-each-ref", "--sort=refname", "--format="+format, "refs/tags/")
	if err != nil {
		return nil, err
	}
	return parseTags(out)
}

func parseTags(out string) ([]Tag, error) {
	if out == "" {
		return nil, nil
	}
	if !strings.HasSuffix(out, "\x00\n") {
		return nil, fmt.Errorf("无效的标签记录")
	}
	var tags []Tag
	for _, record := range strings.Split(strings.TrimSuffix(out, "\x00\n"), "\x00\n") {
		f := strings.Split(record, "\x00")
		if len(f) != 15 || f[0] == "" || !oidRE.MatchString(f[1]) {
			return nil, fmt.Errorf("无效的标签记录")
		}
		tag := Tag{Name: f[0], ObjectSHA: f[1], ObjectType: f[2], TargetSHA: f[1], TargetType: f[2], SymbolicRef: f[5]}
		metadata := f[6:10]
		if tag.ObjectType == "tag" {
			tag.TargetSHA, tag.TargetType = f[3], f[4]
			tag.Message = strings.TrimSpace(f[14])
			metadata = f[10:14]
		}
		if !oidRE.MatchString(tag.TargetSHA) || !gitObjectType(tag.ObjectType) || !gitObjectType(tag.TargetType) {
			return nil, fmt.Errorf("无效的标签目标")
		}
		if tag.TargetType == "commit" {
			tag.Commit = Commit{SHA: tag.TargetSHA, Parents: strings.Fields(metadata[0]), Subject: metadata[1], Author: metadata[2], Date: metadata[3], Day: localDay(metadata[3])}
			for _, parent := range tag.Commit.Parents {
				if !oidRE.MatchString(parent) {
					return nil, fmt.Errorf("无效的标签提交父 SHA")
				}
			}
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func gitObjectType(value string) bool {
	return value == "commit" || value == "tag" || value == "tree" || value == "blob"
}
