package content

import (
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/criscornea/static_studio/internal/frontmatter"
)

// Node is one entry in the content tree. A directory has Children;
// a file has a Size and an Ext, and Fields when it has readable frontmatter.
type Node struct {
	Name     string              `json:"name"`
	Path     string              `json:"path"`
	IsDir    bool                `json:"isDir"`
	Size     int64               `json:"size,omitempty"`
	Ext      string              `json:"ext,omitempty"`
	Fields   *frontmatter.Fields `json:"fields,omitempty"`
	Children []*Node             `json:"children,omitempty"`
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
			Name:   name,
			Path:   child,
			IsDir:  false,
			Size:   info.Size(),
			Ext:    strings.ToLower(path.Ext(name)),
			Fields: readFields(fsys, child, info.Size()),
		})
	}

	sortChildren(root)
	return root, nil
}

// readFields returns the frontmatter fields of the file at name, or nil when
// the file has no frontmatter, is too large, or cannot be read or parsed.
// A broken page must not break the tree: the page view reports the error
// when the user opens it.
func readFields(fsys fs.FS, name string, size int64) *frontmatter.Fields {
	if size > maxPageSize {
		return nil
	}
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil
	}
	doc, err := frontmatter.Split(data)
	if err != nil || doc.Format == frontmatter.FormatNone {
		return nil
	}
	fields, err := doc.Fields()
	if err != nil {
		return nil
	}
	return fields
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
