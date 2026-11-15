package lobedesk

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

// docs/graph-colors.md explains the capsule and node measurements in prose and
// repeats the machine-checked values in a keyed table (section 7). The tests
// read the table, not the sentences: rewords used to fail a check while the
// app still drew the right thing, and a number in prose could drift while the
// check kept matching an unrelated mention of it.
//
// A value the code computes changes first; the failure then names the expected
// text, which is what goes into the table.
var docNumberRow = regexp.MustCompile(`(?m)^\|\s*([a-z][a-z0-9-]*)\s*\|\s*([^|]+?)\s*\|\s*$`)

func graphColorDocNumbers(t *testing.T) map[string]string {
	t.Helper()
	doc, err := os.ReadFile("../../docs/graph-colors.md")
	if err != nil {
		t.Fatalf("read docs/graph-colors.md: %v", err)
	}
	values := map[string]string{}
	for _, match := range docNumberRow.FindAllStringSubmatch(string(doc), -1) {
		values[match[1]] = match[2]
	}
	// The table is the only place with lowercase keys; a shape change would
	// otherwise read as "every value is missing".
	if len(values) < 10 {
		t.Fatalf("docs/graph-colors.md section 7 lists %d keyed values; the table is missing or changed shape", len(values))
	}
	return values
}

func checkDocNumber(t *testing.T, values map[string]string, key, want string) {
	t.Helper()
	got, ok := values[key]
	if !ok {
		t.Errorf("docs/graph-colors.md has no %q row; add it to the section 7 table", key)
		return
	}
	if got != want {
		t.Errorf("docs/graph-colors.md says %s = %q; the code computes %q", key, got, want)
	}
}

// The capsule constants live in ref_capsule.go and are explained in
// docs/graph-colors.md. A tint or a height changed in one place and not the
// other leaves the document describing a capsule the app no longer draws, and
// the next round reads the document as truth.
func TestGraphColorDocMatchesTheCapsuleConstants(t *testing.T) {
	values := graphColorDocNumbers(t)
	checkDocNumber(t, values, "capsule-tint", fmt.Sprintf("%.2f", refCapsuleTint))
	checkDocNumber(t, values, "capsule-border", fmt.Sprintf("%.2f", refCapsuleBorder))
	checkDocNumber(t, values, "capsule-height", fmt.Sprintf("%d", refCapsuleHeight))
	checkDocNumber(t, values, "capsule-corner", fmt.Sprintf("%d", refCapsuleHeight/2))
	checkDocNumber(t, values, "capsule-row-width", fmt.Sprintf("%d", refCapsuleRowWidth))
	checkDocNumber(t, values, "capsule-detail-width", fmt.Sprintf("%d", refCapsuleDetailWidth))
}

// The node geometry and the light-theme line contrast are computed, not
// constants someone can search for: the dot and the merge square scale with
// the node preset, and the selected node's ring and halo are derived from
// them. Writing the expected text from the functions keeps the table honest
// even after the formulas change, which is when a prose-only description goes
// stale.
func TestGraphColorDocMatchesTheGraphGeometry(t *testing.T) {
	values := graphColorDocNumbers(t)
	inner, halo := graphNodeRing(graphNodeStandard, graphLineStandard)
	checkDocNumber(t, values, "node-dot-radius", fmt.Sprintf("%d", int(graphNodeDot(graphNodeStandard, graphLineStandard))))
	checkDocNumber(t, values, "node-merge-half", fmt.Sprintf("%d", int(graphNodeMerge(graphNodeStandard, graphLineStandard))))
	checkDocNumber(t, values, "node-ring", fmt.Sprintf("%d", int(inner*2)))
	checkDocNumber(t, values, "node-halo", fmt.Sprintf("%d", int(halo*2)))
	checkDocNumber(t, values, "lane-ink-min", fmt.Sprintf("%.1f:1", laneInkMin))
}

// The table must not accumulate rows the tests never read: a stale key reads
// as a checked value while nothing compares it.
func TestEveryDocNumberRowIsChecked(t *testing.T) {
	values := graphColorDocNumbers(t)
	checked := map[string]bool{
		"capsule-tint": true, "capsule-border": true, "capsule-height": true,
		"capsule-corner": true, "capsule-row-width": true, "capsule-detail-width": true,
		"node-dot-radius": true, "node-merge-half": true, "node-ring": true,
		"node-halo": true, "lane-ink-min": true,
	}
	for key := range values {
		if strings.TrimSpace(key) != "" && !checked[key] {
			t.Errorf("docs/graph-colors.md lists %q, but no test reads it", key)
		}
	}
}
