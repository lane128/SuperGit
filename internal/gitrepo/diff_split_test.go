package gitrepo

import "testing"

const multiFilePatch = `diff --git a/alpha.go b/alpha.go
index 1111111..2222222 100644
--- a/alpha.go
+++ b/alpha.go
@@ -1,2 +1,2 @@ func main() {
-old
+new
 ctx
diff --git a/dir/space name.txt b/dir/space name.txt
new file mode 100644
index 0000000..3333333
--- /dev/null
+++ b/dir/space name.txt
@@ -0,0 +1 @@
+hello
diff --git a/gone.txt b/gone.txt
deleted file mode 100644
index 4444444..0000000
--- a/gone.txt
+++ /dev/null
@@ -1 +0,0 @@
-x
diff --git a/old.go b/new.go
similarity index 90%
rename from old.go
rename to new.go
index 5555555..6666666 100644
--- a/old.go
+++ b/new.go
@@ -1 +1 @@
-a
+b
diff --git a/bin.dat b/bin.dat
index 7777777..8888888 100644
Binary files a/bin.dat and b/bin.dat differ
`

func TestSplitFileDiffsReadsEveryFileInOrder(t *testing.T) {
	files := splitFileDiffs(multiFilePatch)
	want := []string{"alpha.go", "dir/space name.txt", "gone.txt", "new.go", "bin.dat"}
	if len(files) != len(want) {
		t.Fatalf("split into %d files; want %d: %+v", len(files), len(want), files)
	}
	for i, path := range want {
		if files[i].Path != path {
			t.Errorf("file %d is %q; want %q", i, files[i].Path, path)
		}
	}
	// Each section keeps its own hunks, so a file's lines still know their
	// patch, and the deleted and binary files keep the rows that explain them.
	if hunks := files[0].Diff.Hunks; len(hunks) != 1 {
		t.Errorf("alpha.go has %d hunks; want 1", len(hunks))
	}
	if kinds := kindsOf(files[2].Diff); kinds != "meta,meta,meta,meta,meta,hunk,del" {
		t.Errorf("gone.txt lines are %q; want its deletion row", kinds)
	}
	if hunks := files[4].Diff.Hunks; len(hunks) != 0 || kindsOf(files[4].Diff) != "meta,meta,meta" {
		t.Errorf("bin.dat = %q with %d hunks; want the binary note only", kindsOf(files[4].Diff), len(files[4].Diff.Hunks))
	}
}

func TestSplitFileDiffsQuotedPath(t *testing.T) {
	patch := "diff --git \"a/tab\\tname\" \"b/tab\\tname\"\n" +
		"index 1111111..2222222 100644\n" +
		"--- \"a/tab\\tname\"\n" +
		"+++ \"b/tab\\tname\"\n" +
		"@@ -1 +1 @@\n-a\n+b\n"
	files := splitFileDiffs(patch)
	if len(files) != 1 || files[0].Path != "tab\tname" {
		t.Fatalf("files = %+v; want one file named tab\\tname", files)
	}
}

func TestSplitFileDiffsLeavesAnEmptyPatchEmpty(t *testing.T) {
	if files := splitFileDiffs(""); len(files) != 0 {
		t.Errorf("split an empty patch into %+v; want none", files)
	}
}

func kindsOf(d Diff) string {
	out := ""
	for i, line := range d.Lines {
		if i > 0 {
			out += ","
		}
		out += line.Kind
	}
	return out
}
