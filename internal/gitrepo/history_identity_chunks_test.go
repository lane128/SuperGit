package gitrepo

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestHistoryIdentityFramingAtEveryByteBoundary(t *testing.T) {
	date := "2026-11-18T12:00:00+08:00"
	first := []string{strings.Repeat("a", 40), "", "root", "作者 A", "MiXeD@example.invalid", date, "tag: v1"}
	second := []string{strings.Repeat("b", 40), strings.Repeat("a", 40) + " " + strings.Repeat("c", 40), "merge 名", "作者 B", "", date, ""}
	encode := func(fields []string) string { return strings.Join(fields, "\x00") + "\x00" }
	assertRecord := func(t *testing.T, c Commit, want []string) {
		t.Helper()
		got := []string{c.SHA, strings.Join(c.Parents, " "), c.Subject, c.Author, c.AuthorEmail, c.Date, c.Refs}
		if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
			t.Fatalf("record boundary changed identity or metadata: %+v", c)
		}
	}
	for pairIndex, pair := range [][2][]string{{first, second}, {second, first}} {
		first, second := pair[0], pair[1]
		raw := encode(first) + encode(second)
		for split := 0; split <= len(raw); split++ {
			t.Run(fmt.Sprintf("pair-%d/split-%d", pairIndex, split), func(t *testing.T) {
				chunks := [][]byte{[]byte(raw[:split]), []byte(raw[split:])}
				commits := make(chan Commit, 2)
				stream := historyStreamOutput{ctx: context.Background(), commits: commits}
				page := fileHistoryPageOutput{skipFields: 7, page: limitedBuffer{limit: outputLimit}}
				for _, chunk := range chunks {
					if n, err := stream.Write(chunk); err != nil || n != len(chunk) {
						t.Fatalf("stream write: %d %v", n, err)
					}
					if n, err := page.Write(chunk); err != nil || n != len(chunk) {
						t.Fatalf("page write: %d %v", n, err)
					}
				}
				if len(commits) != 2 || len(stream.record) != 0 || stream.fields != 0 {
					t.Fatal("stream lost a boundary or retained a partial record")
				}
				assertRecord(t, <-commits, first)
				assertRecord(t, <-commits, second)
				if page.skipFields != 0 || page.page.String() != encode(second) {
					t.Fatal("skip crossed into the next record or retained old bytes")
				}
				parsed, err := parseHistory(page.page.String())
				if err != nil || len(parsed) != 1 {
					t.Fatalf("skipped page is not exactly one record: %v", err)
				}
				assertRecord(t, parsed[0], second)
			})
		}
		// Also cross many boundaries inside a single UTF-8 name and between adjacent
		// NULs for empty email/refs, with one-byte subprocess writes throughout.
		commits := make(chan Commit, 2)
		stream := historyStreamOutput{ctx: context.Background(), commits: commits}
		page := fileHistoryPageOutput{skipFields: 7, page: limitedBuffer{limit: outputLimit}}
		for i := 0; i < len(raw); i++ {
			chunk := []byte(raw[i : i+1])
			if _, err := stream.Write(chunk); err != nil {
				t.Fatal(err)
			}
			if _, err := page.Write(chunk); err != nil {
				t.Fatal(err)
			}
		}
		if page.skipFields != 0 || page.page.String() != encode(second) {
			t.Fatal("single-byte skip lost the retained record boundary")
		}
		if len(commits) != 2 {
			t.Fatal("single-byte stream lost records")
		}
		assertRecord(t, <-commits, first)
		assertRecord(t, <-commits, second)
	}
}
