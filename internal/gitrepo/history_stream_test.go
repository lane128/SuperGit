package gitrepo

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestHistoryStreamUsesOnePinnedTraversal(t *testing.T) {
	dir := fixture(t)
	importHistory(t, dir, 2105)
	r := openTest(t, dir)
	first, err := r.HistoryPage(context.Background(), nil, 0, 200)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Fields(git(t, dir, "log", "--topo-order", "--all", "--format=%H"))
	trace := filepath.Join(t.TempDir(), "trace.json")
	t.Setenv("GIT_TRACE2_EVENT", trace)
	stream, err := r.OpenHistoryStream(context.Background(), first.Tips, len(first.Commits))
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	var got []string
	for _, commit := range first.Commits {
		got = append(got, commit.SHA)
	}
	for {
		page, err := stream.Next(200)
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Commits) > 200 || !slices.Equal(page.Tips, first.Tips) {
			t.Fatal("stream lost page bound or pinned tips")
		}
		for _, commit := range page.Commits {
			got = append(got, commit.SHA)
		}
		if !page.HasMore {
			break
		}
	}
	if !slices.Equal(got, want) {
		t.Fatal("stream changed history membership/order")
	}
	data, err := os.ReadFile(trace)
	if err != nil {
		t.Fatal(err)
	}
	logs := 0
	for _, line := range strings.Split(string(data), "\n") {
		var event struct {
			Event string `json:"event"`
			Name  string `json:"name"`
		}
		if json.Unmarshal([]byte(line), &event) == nil && event.Event == "cmd_name" && event.Name == "log" {
			logs++
		}
	}
	if logs != 1 {
		t.Fatalf("stream started %d git log traversals; want one", logs)
	}
}

func TestHistoryStreamCancellationReleasesBackpressure(t *testing.T) {
	dir := fixture(t)
	importHistory(t, dir, 600)
	r := openTest(t, dir)
	first, err := r.HistoryPage(context.Background(), nil, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	stream, err := r.OpenHistoryStream(context.Background(), first.Tips, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stream.Next(1); err != nil {
		t.Fatal(err)
	}
	stream.Close()
	select {
	case <-stream.done:
	case <-time.After(3 * time.Second):
		t.Fatal("cancel did not release blocked Git writer")
	}
	if _, err := stream.Next(200); err == nil {
		t.Fatal("closed stream returned a successful page")
	}
}

func TestHistoryStreamSurvivesConsumerPause(t *testing.T) {
	dir := fixture(t)
	importHistory(t, dir, 1000)
	r := openTest(t, dir)
	first, err := r.HistoryPage(context.Background(), nil, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	stream, err := r.OpenHistoryStream(context.Background(), first.Tips, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if _, err := stream.Next(1); err != nil {
		t.Fatal(err)
	}
	// The timeout-overlay run shortens ordinary command deadlines to 50ms.
	// A paused consumer must not spend a whole-traversal deadline.
	time.Sleep(100 * time.Millisecond)
	if _, err := stream.Next(2000); err != nil {
		t.Fatalf("consumer pause killed healthy traversal: %v", err)
	}
}

func TestHistoryStreamDoesNotBufferWholeOutput(t *testing.T) {
	dir := fixture(t)
	importHistoryWithPadding(t, dir, 1000, 5000)
	r := openTest(t, dir)
	first, err := r.HistoryPage(context.Background(), nil, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	stream, err := r.OpenHistoryStream(context.Background(), first.Tips, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	count := 0
	for {
		page, err := stream.Next(100)
		if err != nil {
			t.Fatal(err)
		}
		count += len(page.Commits)
		if !page.HasMore {
			break
		}
	}
	if count != 1000 {
		t.Fatalf("stream stopped after %d records", count)
	}
}
