package gitrepo

import (
	"testing"
	"time"
)

// The history search matches the day the commit list prints, so a commit keeps
// the calendar day this machine would show, not the one its author saw. A
// commit made shortly before midnight in Tokyo falls on the previous day here.
func TestParseHistoryKeepsTheDayTheListShows(t *testing.T) {
	when := "2026-11-08T23:30:00+09:00"
	commits, err := parseHistory("abc123\x00\x00subject\x00Ada\x00author@example.invalid\x00" + when + "\x00main")
	if err != nil {
		t.Fatalf("parseHistory: %v", err)
	}
	parsed, err := time.Parse(time.RFC3339, when)
	if err != nil {
		t.Fatalf("parse %q: %v", when, err)
	}
	want := parsed.Local().Format("2006-01-02")
	if commits[0].Day != want {
		t.Errorf("Day = %q; want the local day %q", commits[0].Day, want)
	}
	if commits[0].Date != when {
		t.Errorf("Date = %q; want the raw value %q kept for display", commits[0].Date, when)
	}
}

// A date Git printed in a form this code cannot parse leaves Day empty: the
// day filter then has nothing to offer, rather than matching the wrong text.
func TestParseHistoryLeavesAnUnreadableDateWithoutADay(t *testing.T) {
	commits, err := parseHistory("abc123\x00\x00subject\x00Ada\x00author@example.invalid\x00not-a-date\x00main")
	if err != nil {
		t.Fatalf("parseHistory: %v", err)
	}
	if commits[0].Day != "" {
		t.Errorf("Day = %q; want empty for an unparseable date", commits[0].Day)
	}
}
