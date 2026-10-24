package publish

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	s := NewStore(nil, t.TempDir())
	proj := t.TempDir()

	if got := s.Get(proj); got.Remote != "" || got.Branch != DefaultBranch {
		t.Fatalf("default = %+v", got)
	}
	saved, err := s.Set(proj, Settings{Remote: " git@github.com:a/b.git "})
	if err != nil {
		t.Fatal(err)
	}
	if saved.Branch != DefaultBranch || saved.Remote != "git@github.com:a/b.git" {
		t.Fatalf("saved = %+v", saved)
	}
	if got := s.Get(proj); got != saved {
		t.Fatalf("get = %+v", got)
	}
	if got := s.Get(t.TempDir()); got.Remote != "" {
		t.Fatalf("other project leaked: %+v", got)
	}
}

func TestValidate(t *testing.T) {
	bad := []Settings{
		{},
		{Remote: "--upload-pack=evil"},
		{Remote: "https://user:token@example.com/a.git"},
		{Remote: "relative/path.git"},
		{Remote: "/srv/x.git", Branch: "main"},
		{Remote: "/srv/x.git", Branch: "master"},
		{Remote: "/srv/x.git", Branch: "-b"},
		{Remote: "/srv/x.git", Branch: "a..b"},
		{Remote: "/srv/x.git", Branch: "a b"},
		{Remote: "/srv/x.git", Branch: "a.lock"},
	}
	for _, b := range bad {
		if _, err := b.Validate(); !errors.Is(err, ErrInvalid) {
			t.Errorf("%+v: err = %v", b, err)
		}
	}
	if _, err := (Settings{Remote: "/srv/x.git", Branch: "release/pages"}).Validate(); err != nil {
		t.Fatal(err)
	}
	for _, r := range []string{"git@github.com:a/b.git", "https://github.com/a/b.git", "https://user@example.com/a.git", "ssh://git@host/a.git", "file:///srv/x.git"} {
		if _, err := (Settings{Remote: r}).Validate(); err != nil {
			t.Errorf("%s: %v", r, err)
		}
	}
}

func TestStoreCorruptAndDisabled(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(nil, dir)
	if err := os.MkdirAll(filepath.Join(dir, "static-studio"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "static-studio", "publish.json"), []byte("{nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := s.Get("/p"); got.Branch != DefaultBranch {
		t.Fatalf("got %+v", got)
	}
	if _, err := NewStore(nil, "").Set("/p", Settings{Remote: "/srv/x.git"}); err != nil {
		t.Fatal(err)
	}
}
