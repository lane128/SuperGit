package lobedesk

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"supergit/internal/gitrepo"
)

func TestFilePreviewPolicy(t *testing.T) {
	for _, tc := range []struct {
		name   string
		binary bool
		want   previewKind
	}{
		{"cover.PnG", true, imagePreview}, {"renamed.jpeg", false, imagePreview}, {"clip.gif", true, imagePreview},
		{"wall.webp", true, imagePreview}, {"scan.bmp", true, imagePreview}, {"bundle.ZIP", false, binaryPreview},
		{"app.exe", true, binaryPreview}, {"manual.pdf", false, binaryPreview}, {"unknown", true, binaryPreview},
		{"drawing.svg", false, textPreview}, {"README.md", false, textPreview}, {"source.go", false, textPreview},
	} {
		if kind := previewFor(tc.name, tc.binary); kind != tc.want {
			t.Errorf("%s: got %v", tc.name, kind)
		}
	}
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 2000, 20))); err != nil {
		t.Fatal(err)
	}
	pic := decodePreview(data.Bytes(), nil)
	if pic.problem != "" || pic.width != 2000 || pic.height != 20 {
		t.Fatalf("%+v", pic)
	}
	if w, _ := pic.bitmap.Size(); w != 1280 {
		t.Fatalf("thumbnail width %d", w)
	}
	if p := decodePreview([]byte("not an image"), nil); p.problem != "decode" {
		t.Fatal(p.problem)
	}
	if p := decodePreview([]byte("version https://git-lfs.github.com/spec/v1\noid sha256:123"), nil); p.problem != "lfs" {
		t.Fatal(p.problem)
	}
}

func TestFilePreviewPairsRespectAddDeleteAndRename(t *testing.T) {
	for _, tc := range []struct{ status, old, before, after string }{
		{"A", "", "", "new"}, {"?", "", "", "new"}, {"D", "", "old", ""}, {"R100", "old.png", "old", "new"}, {"M", "", "old", "new"},
	} {
		p := previewVersions(gitrepo.ChangedFile{Path: "new.jpg", OriginalPath: tc.old, Status: tc.status}, "old", "new")
		if p.before.rev != tc.before || p.after.rev != tc.after || tc.old != "" && p.before.path != tc.old {
			t.Errorf("%s: %+v", tc.status, p)
		}
	}
}
