package devserver

import "testing"

func TestPageURLPath(t *testing.T) {
	tests := []struct {
		name string
		want string
		ok   bool
	}{
		{"content/posts/hello.md", "/posts/hello/", true},
		{"content/_index.md", "/", true},
		{"content/posts/_index.md", "/posts/", true},
		{"content/posts/bundle/index.md", "/posts/bundle/", true},
		{"content/about.markdown", "/about/", true},
		{"content/a b/c.md", "/a-b/c/", true},
		{"content/Posts/Hello World.md", "/posts/hello-world/", true},
		{"static/x.md", "", false},
		{"content", "", false},
		{"contentx/a.md", "", false},
	}
	for _, tt := range tests {
		got, ok := PageURLPath("content", tt.name, false)
		if got != tt.want || ok != tt.ok {
			t.Errorf("%s = %q, %v; want %q, %v", tt.name, got, ok, tt.want, tt.ok)
		}
	}
}

func TestPageURLPathKeepCase(t *testing.T) {
	got, ok := PageURLPath("content", "content/Posts/Hello World.md", true)
	if got != "/Posts/Hello-World/" || !ok {
		t.Errorf("got %q, %v", got, ok)
	}
}
