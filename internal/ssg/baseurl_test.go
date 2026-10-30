package ssg

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func baseRoot(t *testing.T, files map[string]string) *os.Root {
	t.Helper()
	dir := t.TempDir()
	for name, data := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	r, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r
}

func noEnv(string) string { return "" }

func setBase(t *testing.T, root *os.Root, file, url string) *BaseURLConfig {
	t.Helper()
	c, err := ReadBaseURL(root, file, noEnv)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Write(root, url); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestBaseURLWriter(t *testing.T) {
	const u = "https://a.github.io/blog/"
	cases := []struct{ name, file, in, want string }{
		{"toml replace", "hugo.toml",
			"# site\ntitle = 'x'\nbaseURL = \"http://old/\" # keep me\n\n[params]\nbaseURL = \"nested\"\n",
			"# site\ntitle = 'x'\nbaseURL = \"" + u + "\" # keep me\n\n[params]\nbaseURL = \"nested\"\n"},
		{"toml insert", "hugo.toml",
			"title = 'x'\n[params]\nbaseURL = \"nested\"\n",
			"baseURL = \"" + u + "\"\ntitle = 'x'\n[params]\nbaseURL = \"nested\"\n"},
		{"toml crlf mixed case key", "config.toml",
			"BaseURL = 'http://old/'\r\ntitle = 'x'\r\n",
			"BaseURL = \"" + u + "\"\r\ntitle = 'x'\r\n"},
		{"yaml replace bare", "hugo.yaml",
			"title: x\nbaseURL: http://old/  # c\nparams:\n  baseURL: nested\n",
			"title: x\nbaseURL: \"" + u + "\"  # c\nparams:\n  baseURL: nested\n"},
		{"yaml insert after marker", "hugo.yaml",
			"---\n# top\ntitle: x\nparams:\n  baseURL: nested\n",
			"---\nbaseURL: \"" + u + "\"\n# top\ntitle: x\nparams:\n  baseURL: nested\n"},
		{"json replace", "hugo.json",
			"{\n  \"title\": \"x\",\n  \"params\": {\"baseURL\": \"n\"},\n  \"baseURL\": \"http://old/\"\n}\n",
			"{\n  \"title\": \"x\",\n  \"params\": {\"baseURL\": \"n\"},\n  \"baseURL\": \"" + u + "\"\n}\n"},
		{"json insert", "hugo.json",
			"{\n  \"title\": \"x\"\n}\n",
			"{\n  \"baseURL\": \"" + u + "\",\n  \"title\": \"x\"\n}\n"},
		{"json insert empty", "hugo.json", "{}", "{\n  \"baseURL\": \"" + u + "\"}"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := baseRoot(t, map[string]string{c.file: c.in})
			setBase(t, root, c.file, u)
			got, _ := os.ReadFile(filepath.Join(root.Name(), c.file))
			if string(got) != c.want {
				t.Fatalf("got:\n%q\nwant:\n%q", got, c.want)
			}
			entries, _ := os.ReadDir(root.Name())
			if len(entries) != 1 {
				t.Fatalf("temp file left behind: %v", entries)
			}
		})
	}
}

func TestBaseURLFindsDefiningFile(t *testing.T) {
	root := baseRoot(t, map[string]string{
		"hugo.toml":                   "title = 'x'\n",
		"config/_default/config.toml": "baseURL = 'http://old/'\n",
	})
	c, err := ReadBaseURL(root, "hugo.toml", noEnv)
	if err != nil || c.File != "config/_default/config.toml" || c.Current != "http://old/" {
		t.Fatalf("%+v %v", c, err)
	}
	// Nothing defines it: the primary config gets the key.
	root = baseRoot(t, map[string]string{"hugo.toml": "title = 'x'\n"})
	c, err = ReadBaseURL(root, "hugo.toml", noEnv)
	if err != nil || c.File != "hugo.toml" || c.Current != "" {
		t.Fatalf("%+v %v", c, err)
	}
}

func TestBaseURLStaleFileRefused(t *testing.T) {
	root := baseRoot(t, map[string]string{"hugo.toml": "baseURL = 'a'\n"})
	c, err := ReadBaseURL(root, "hugo.toml", noEnv)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root.Name(), "hugo.toml"), []byte("baseURL = 'a'\n# edited\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := c.Write(root, "https://x.example/"); !errors.Is(err, ErrConfigChanged) {
		t.Fatalf("err = %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(root.Name(), "hugo.toml"))
	if !strings.Contains(string(got), "# edited") || strings.Contains(string(got), "x.example") {
		t.Fatalf("file modified: %q", got)
	}
}

func TestBaseURLWarnings(t *testing.T) {
	root := baseRoot(t, map[string]string{
		"hugo.toml":                   "baseURL = 'a'\n",
		"config/production/hugo.toml": "baseURL = 'b'\n",
	})
	c, err := ReadBaseURL(root, "hugo.toml", func(k string) string {
		if k == "HUGO_BASEURL" {
			return "https://env/"
		}
		return ""
	})
	if err != nil || len(c.Warnings) != 2 {
		t.Fatalf("%+v %v", c, err)
	}
}

func TestBaseURLRefusesUnsupportedValues(t *testing.T) {
	for file, in := range map[string]string{
		"hugo.toml": "baseURL = \"\"\"\nx\"\"\"\n",
		"hugo.yaml": "baseURL:\n  x\n",
		"hugo.json": "{\"baseURL\": 3}",
	} {
		root := baseRoot(t, map[string]string{file: in})
		if _, err := ReadBaseURL(root, file, noEnv); err == nil {
			t.Errorf("%s accepted", file)
		}
	}
}
