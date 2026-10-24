package content

import "testing"

func TestDirsContaining(t *testing.T) {
	d := Dirs{"content", "content.de", "content/nested"}
	tests := []struct {
		name string
		want string
		ok   bool
	}{
		{"content/a.md", "content", true},
		{"content.de/a.md", "content.de", true},
		{"content/nested/a.md", "content/nested", true},
		{"content-old/a.md", "", false},
		{"contentx/a.md", "", false},
		{"content.md", "", false},
		{"content/", "", false},
		{"content", "", false},
		{"other/content/a.md", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		got, ok := d.Containing(tt.name)
		if got != tt.want || ok != tt.ok {
			t.Errorf("Containing(%q) = %q, %v; want %q, %v", tt.name, got, ok, tt.want, tt.ok)
		}
	}
	if (Dirs{}).Has("content") || !d.Has("content.de") || d.Has("content-old") {
		t.Error("Has wrong")
	}
}
