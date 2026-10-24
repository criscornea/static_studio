package build

import "testing"

func TestParseLine(t *testing.T) {
	const root = "/home/u/site"
	tests := []struct {
		name string
		line string
		want Problem
		ok   bool
	}{
		{
			"yaml frontmatter",
			`ERROR 2024/05/01 10:11:12 error building site: process: readAndProcessContent: "/home/u/site/content/posts/a.md:3:8": failed to unmarshal YAML: yaml: line 2: mapping values are not allowed in this context`,
			Problem{"content/posts/a.md", 3, 8, "error building site: process: readAndProcessContent: failed to unmarshal YAML: yaml: line 2: mapping values are not allowed in this context"},
			true,
		},
		{
			"template",
			`ERROR render of "page" failed: "/home/u/site/layouts/_default/single.html:5:12": execute of template failed: template: _default/single.html:5:12: executing "main" at <.Foo>: can't evaluate field Foo`,
			Problem{"layouts/_default/single.html", 5, 12, `render of "page" failed: execute of template failed: template: _default/single.html:5:12: executing "main" at <.Foo>: can't evaluate field Foo`},
			true,
		},
		{
			"no column, Error prefix",
			`Error: error building site: "/home/u/site/content/b.md:7": TOML parse error`,
			Problem{"content/b.md", 7, 0, "error building site: TOML parse error"},
			true,
		},
		{"outside root", `ERROR "/other/place/x.md:1:1": bad`, Problem{}, false},
		{"no location", `Total in 12 ms`, Problem{}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ParseLine(root, tc.line)
			if ok != tc.ok || got != tc.want {
				t.Fatalf("got %+v %v, want %+v %v", got, ok, tc.want, tc.ok)
			}
		})
	}
}
