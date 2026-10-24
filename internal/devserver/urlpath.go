package devserver

import (
	"net/url"
	"path"
	"strings"
)

// PageURLPath maps a content file such as "content/posts/hello.md" to the
// path Hugo serves it at, "/posts/hello/". It follows Hugo's default
// permalinks only: `url`, `slug` and language suffixes are not considered.
// Segments follow Hugo's MakePath: spaces become hyphens, and the path is
// lowercased unless keepCase is set (Hugo's disablePathToLower).
// ok is false when name is not inside contentDir.
func PageURLPath(contentDir, name string, keepCase bool) (string, bool) {
	rel, found := strings.CutPrefix(name, contentDir+"/")
	if !found || rel == "" {
		return "", false
	}

	rel = strings.TrimSuffix(rel, path.Ext(rel))
	dir, leaf := path.Split(rel)
	if leaf == "_index" || leaf == "index" {
		rel = strings.TrimSuffix(dir, "/")
	}
	if rel == "" {
		return "/", true
	}

	segs := strings.Split(rel, "/")
	for i, s := range segs {
		s = strings.ReplaceAll(strings.TrimSpace(s), " ", "-")
		if !keepCase {
			s = strings.ToLower(s)
		}
		segs[i] = url.PathEscape(s)
	}

	return "/" + strings.Join(segs, "/") + "/", true
}
