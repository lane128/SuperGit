package lobedesk

import (
	"testing"

	"supergit/internal/gitrepo"
)

func TestGraphNodeShapesFollowLoadedDAGAndActualHEAD(t *testing.T) {
	commits := []gitrepo.Commit{
		{SHA: "merge", Parents: []string{"left", "right"}},
		{SHA: "left", Parents: []string{"fork"}},
		{SHA: "right", Parents: []string{"fork"}},
		{SHA: "fork", Parents: []string{"outside"}},
	}
	graph := newCommitGraph("", commits, nil)
	for i, want := range []string{"merge", "dot", "dot", "fork"} {
		if got := graphNodeShape(commits[i], "outside", graph.Rows[i]); got != want {
			t.Errorf("%s: shape=%s want %s", commits[i].SHA, got, want)
		}
	}
	if got := graphNodeShape(commits[0], "merge", graph.Rows[0]); got != "head" {
		t.Fatalf("actual HEAD must retain a hollow ring even on a merge: %s", got)
	}
	if got := graphNodeShape(commits[1], "left", graph.Rows[1]); got != "head" {
		t.Fatalf("detached HEAD not recognized by SHA: %s", got)
	}
	if !graph.Rows[3].Boundary {
		t.Fatal("fork shape removed the unloaded-parent boundary")
	}
	filtered := newCommitGraph("", []gitrepo.Commit{commits[1], commits[3]}, nil)
	if got := graphNodeShape(commits[3], "outside", filtered.Rows[1]); got != "dot" {
		t.Fatalf("filtered graph invented an unloaded second child: %s", got)
	}
	if got := graphNodeShape(commits[0], "", newCommitGraph("", commits[:1], nil).Rows[0]); got != "merge" {
		t.Fatalf("unloaded merge parents erased known merge identity: %s", got)
	}
}

func TestGraphForkNeedsDistinctLoadedChildren(t *testing.T) {
	commits := []gitrepo.Commit{
		{SHA: "child", Parents: []string{"parent", "parent"}},
		{SHA: "parent"},
	}
	graph := newCommitGraph("", commits, nil)
	if got := graphNodeShape(commits[1], "", graph.Rows[1]); got != "dot" {
		t.Fatalf("duplicate parent headers invented a fork: %s", got)
	}
	if got := graphNodeShape(commits[0], "", graph.Rows[0]); got != "dot" {
		t.Fatalf("duplicate parent headers invented a merge: %s", got)
	}
}

func TestSelectedGraphRingClearsIdentityRingAtEveryPreset(t *testing.T) {
	for _, node := range []float32{graphNodeSmall, graphNodeStandard, graphNodeLarge} {
		for _, line := range []float32{graphLineThin, graphLineStandard, graphLineThick} {
			outer, _ := graphNodeRing(node, line)
			// Painter.Stroke paints inside the rectangle: the selection
			// ring's 2-DIP stroke needs 1.5 DIP of visible ground beyond
			// the outermost merge/HEAD/fork mark.
			if gap := outer - 2 - graphNodeMerge(node, line); gap < 1.5 {
				t.Errorf("node=%v line=%v: identity/selection gap=%v", node, line, gap)
			}
		}
	}
}
