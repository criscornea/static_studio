package publish

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAddress(t *testing.T) {
	cases := []struct {
		kind, owner, repo, domain, want string
		fail                            bool
	}{
		{SiteProject, "Alice", "blog", "", "https://alice.github.io/blog/", false},
		{SiteUser, "alice", "alice.github.io", "", "https://alice.github.io/", false},
		{SiteUser, "alice", "blog", "", "", true},
		{SiteCustom, "", "", "www.example.com", "https://www.example.com/", false},
		{SiteCustom, "", "", "Example.com", "", true},
		{SiteProject, "", "", "", "", true},
		{"", "a", "b", "", "", true},
	}
	for _, c := range cases {
		got, err := Address(c.kind, c.owner, c.repo, c.domain)
		if (err != nil) != c.fail || got != c.want {
			t.Errorf("Address(%+v) = %q, %v", c, got, err)
		}
	}
}

func TestValidDomain(t *testing.T) {
	for d, want := range map[string]bool{
		"example.com": true, "www.my-site.example.org": true,
		"localhost": false, "https://example.com": false, "example.com:80": false,
		"example.com/x": false, "*.example.com": false, "Example.com": false,
		"example.com.": false, "-a.com": false, "": false,
	} {
		if ValidDomain(d) != want {
			t.Errorf("ValidDomain(%q) != %v", d, want)
		}
	}
}

func TestSiteSettingsValidate(t *testing.T) {
	base := Settings{Remote: "git@github.com:a/b.git"}
	for name, mod := range map[string]func(*Settings){
		"bad kind":      func(s *Settings) { s.SiteKind = "weird" },
		"custom empty":  func(s *Settings) { s.SiteKind = SiteCustom },
		"bad domain":    func(s *Settings) { s.CustomDomain = "http://x.com" },
		"bad cname set": func(s *Settings) { s.CNAMEChoice = "both" },
	} {
		s := base
		mod(&s)
		if _, err := s.Validate(); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := base.Validate(); err != nil {
		t.Fatalf("legacy settings: %v", err)
	}
}

func TestParseGitHubRemote(t *testing.T) {
	for _, r := range []string{"https://github.com/a/b.git", "git@github.com:a/b", "ssh://git@github.com/a/b.git"} {
		if o, n, err := ParseGitHubRemote(r); err != nil || o != "a" || n != "b" {
			t.Errorf("%s: %q %q %v", r, o, n, err)
		}
	}
	if _, _, err := ParseGitHubRemote("https://gitlab.com/a/b"); err == nil {
		t.Error("gitlab accepted")
	}
}

func staticRoot(t *testing.T, cname string) *os.Root {
	t.Helper()
	dir := t.TempDir()
	if cname != "" {
		if err := os.MkdirAll(filepath.Join(dir, "static"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "static", "CNAME"), []byte(cname), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return openRoot(t, dir)
}

func TestResolveCNAME(t *testing.T) {
	custom := Settings{SiteKind: SiteCustom, CustomDomain: "a.example.com"}

	if _, c := custom.ResolveCNAME("a.example.com"); c != nil {
		t.Error("equal values conflict")
	}
	if _, c := custom.ResolveCNAME(""); c != nil {
		t.Error("absent file conflicts")
	}
	if _, c := custom.ResolveCNAME("b.example.com"); c == nil || c.Setting != "a.example.com" || c.Static != "b.example.com" {
		t.Errorf("conflict = %+v", c)
	}

	keep := custom
	keep.CNAMEChoice, keep.CNAMEChoiceFor = CNAMESetting, "b.example.com"
	if got, c := keep.ResolveCNAME("b.example.com"); c != nil || got.CustomDomain != "a.example.com" {
		t.Errorf("setting choice: %+v %+v", got, c)
	}
	adopt := custom
	adopt.CNAMEChoice, adopt.CNAMEChoiceFor = CNAMEStatic, "b.example.com"
	if got, c := adopt.ResolveCNAME("b.example.com"); c != nil || got.CustomDomain != "b.example.com" {
		t.Errorf("static choice: %+v %+v", got, c)
	}
	// The file changed after the choice: ask again.
	if _, c := adopt.ResolveCNAME("c.example.com"); c == nil {
		t.Error("stale choice was honoured")
	}
}

func TestCheckSiteAndReadStatic(t *testing.T) {
	root := staticRoot(t, " b.example.com\n")
	if got := ReadStaticCNAME(root); got != "b.example.com" {
		t.Fatalf("static = %q", got)
	}
	if ReadStaticCNAME(staticRoot(t, "not a host")) != "" {
		t.Error("invalid CNAME accepted")
	}
	s := Settings{SiteKind: SiteCustom, CustomDomain: "a.example.com"}
	if _, err := CheckSite(root, s); !errors.Is(err, ErrCNAMEConflict) || !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v", err)
	}
}

func TestUpdateSite(t *testing.T) {
	cur := Settings{Remote: "git@github.com:a/b.git"}
	in := SiteInput{SiteKind: SiteCustom, CustomDomain: "a.example.com"}

	// No static/CNAME, or equal: saved as is.
	if got, c, err := UpdateSite(cur, in, "a.example.com"); err != nil || c != nil || got.CustomDomain != "a.example.com" {
		t.Fatalf("equal: %+v %v %v", got, c, err)
	}
	// Different: conflict, nothing changes.
	got, c, err := UpdateSite(cur, in, "b.example.com")
	if err != nil || c == nil || got.SiteKind != "" {
		t.Fatalf("conflict: %+v %v %v", got, c, err)
	}
	// Explicit choices are stored against the static value.
	in.CNAMEChoice = CNAMEStatic
	got, c, err = UpdateSite(cur, in, "b.example.com")
	if err != nil || c != nil || got.CustomDomain != "b.example.com" || got.CNAMEChoiceFor != "b.example.com" {
		t.Fatalf("static: %+v %v %v", got, c, err)
	}
	in.CNAMEChoice = CNAMESetting
	got, c, err = UpdateSite(cur, in, "b.example.com")
	if err != nil || c != nil || got.CustomDomain != "a.example.com" || got.CNAMEChoice != CNAMESetting {
		t.Fatalf("setting: %+v %v %v", got, c, err)
	}
	// Later edit of static/CNAME asks again.
	in.CNAMEChoice = ""
	if _, c, _ := UpdateSite(got, in, "c.example.com"); c == nil {
		t.Fatal("stale choice honoured")
	}
	// Choice without a file, bad choice.
	in.CNAMEChoice = CNAMEStatic
	if _, _, err := UpdateSite(cur, in, ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v", err)
	}
}
