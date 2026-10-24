package content

import "strings"

// Dirs is the set of resolved content directories of a project, slash
// separated and relative to the project root.
type Dirs []string

// Containing returns the content dir that name lies strictly inside. name
// must be a clean relative path. A sibling that only shares a prefix, such
// as content-old next to content, does not match. With nested dirs the
// longest match wins.
func (d Dirs) Containing(name string) (string, bool) {
	best, found := "", false
	for _, dir := range d {
		if dir == "" || dir == "." {
			continue
		}
		if strings.HasPrefix(name, dir+"/") && len(name) > len(dir)+1 && (!found || len(dir) > len(best)) {
			best, found = dir, true
		}
	}
	return best, found
}

// Has reports whether dir is exactly one of the content dirs.
func (d Dirs) Has(dir string) bool {
	for _, x := range d {
		if x == dir {
			return true
		}
	}
	return false
}
