package content

import (
	"io/fs"
	"path"
	"sort"
	"strings"
)

// Node is one entry in the content tree. A directory has Children;
// a file has a Size and an Ext.
type Node struct {
	Name     string  `json:"name"`
	Path     string  `jons:"path"`
	IsDir    bool    `json:"isDir"`
	Size     int64   `json:"size,omitempty"`
	Ext      string  `json:"ext,omitempty"`
	Children []*Node `json:"children,omitempty"`
}

// editableExts are the file types the editor can open. Everything else is
// hidden from the tree: the user is picking a page to edit, not browsing a disk.
var editableExts = map[string]bool{
	".md":       true,
	".markdown": true,
	".mdx":      true,
	".html":     true,
}

// Tree walks dir within fsys and returns its root node. Directories that
// contain no editable files are omitted. A missing dir is not an error:
// the returned root simply has no children.
func Tree(fsys fs.FS, dir string) (*Node, error) {
	root := &Node{Name: path.Base(dir), Path: dir, IsDir: true}

	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		// A project without a content directory yet is empty, not broken.
		return root, nil //noling:nilerr // see doc comment
	}

	for _, entry := range entries {
		name := entry.Name()
		if skip(name) {
			continue
		}
		child := path.Join(dir, name)

		if entry.IsDir() {
			sub, err := Tree(fsys, child)
			if err != nil {
				return nil, err
			}
			if len(sub.Children) > 0 {
				root.Children = append(root.Children, sub)
			}
			continue
		}

		if !editableExts[strings.ToLower(path.Ext(name))] {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			// The file vanished between ReadDir and Info. Skip it.
			continue
		}
		root.Children = append(root.Children, &Node{
			Name:  name,
			Path:  child,
			IsDir: false,
			Size:  info.Size(),
			Ext:   strings.ToLower(path.Ext(name)),
		})
	}

	sortChildren(root)
	return root, nil
}

// skip reports wether an entry should never appear in the tree.
func skip(name string) bool {
	return strings.HasPrefix(name, ".") || name == "node_modules"
}

// sortChildren orders directories first, then files, each alphabetically.
func sortChildren(n *Node) {
	sort.SliceStable(n.Children, func(i, j int) bool {
		a, b := n.Children[i], n.Children[j]
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
}
