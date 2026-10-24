package content

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var pngBytes = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00")

func TestSaveAsset(t *testing.T) {
	root, dir := openTestRoot(t)
	got, err := SaveAsset(root, "static", "My Photo.PNG", pngBytes)
	if err != nil {
		t.Fatal(err)
	}
	if got != "/images/my-photo.png" {
		t.Fatalf("url = %q", got)
	}
	data, err := os.ReadFile(filepath.Join(dir, "static/images/my-photo.png"))
	if err != nil || !bytes.Equal(data, pngBytes) {
		t.Fatalf("file not written: %v", err)
	}
	assertNoTemps(t, filepath.Join(dir, "static/images"))
}

func TestSaveAssetCollision(t *testing.T) {
	root, dir := openTestRoot(t)
	other := append(append([]byte{}, pngBytes...), 0)
	for i, want := range []string{"/images/a.png", "/images/a-1.png", "/images/a-2.png"} {
		data := pngBytes
		if i > 0 {
			data = other
		}
		got, err := SaveAsset(root, "static", "a.png", data)
		if err != nil || got != want {
			t.Fatalf("save %d = %q, %v; want %q", i, got, err, want)
		}
	}
	first, _ := os.ReadFile(filepath.Join(dir, "static/images/a.png"))
	if !bytes.Equal(first, pngBytes) {
		t.Fatal("original was overwritten")
	}
}

func TestSaveAssetEmptyStem(t *testing.T) {
	root, dir := openTestRoot(t)
	for _, name := range []string{"照片.png", "...png"} {
		got, err := SaveAsset(root, "static", name, pngBytes)
		if err != nil || !strings.HasPrefix(got, "/images/image-") || !strings.HasSuffix(got, ".png") {
			t.Fatalf("%s: got %q, %v", name, got, err)
		}
		if _, err := os.Stat(filepath.Join(dir, "static", got)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSaveAssetRefusals(t *testing.T) {
	root, dir := openTestRoot(t)
	if _, err := SaveAsset(root, "static", "a.png", pngBytes); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "linked"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "linked/images")); err != nil {
		t.Fatal(err)
	}
	big := append(append([]byte{}, pngBytes...), make([]byte, MaxAssetSize)...)

	tests := []struct {
		name, assetDir, file string
		data                 []byte
		want                 error
	}{
		{"too large", "static", "b.png", big, ErrAssetTooLarge},
		{"text as png", "static", "b.png", []byte("hello"), ErrAssetType},
		{"html as png", "static", "b.png", []byte("<html><script>"), ErrAssetType},
		{"svg", "static", "b.svg", []byte("<svg xmlns='http://www.w3.org/2000/svg'/>"), ErrAssetType},
		{"ext mismatch", "static", "b.gif", pngBytes, ErrAssetType},
		{"no ext", "static", "b", pngBytes, ErrAssetName},
		{"symlinked folder", "linked", "b.png", pngBytes, ErrNotEditable},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := SaveAsset(root, tc.assetDir, tc.file, tc.data); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Fatalf("wrote outside project: %v", entries)
	}
}

func TestSaveAssetTraversalName(t *testing.T) {
	root, dir := openTestRoot(t)
	got, err := SaveAsset(root, "static", "../../../evil.png", pngBytes)
	if err != nil {
		t.Fatal(err)
	}
	if got != "/images/evil.png" {
		t.Fatalf("url = %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "static/images/evil.png")); err != nil {
		t.Fatal(err)
	}
}
