package content

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/criscornea/static_studio/internal/frontmatter"
)

// archetypeDir is where Hugo keeps page templates, relative to the project
// root. Hugo lets a config move it; that is not followed yet.
const archetypeDir = "archetypes"

// defaultArchetype is the name Hugo falls back to when no archetype matches.
const defaultArchetype = "default"

// builtinArchetype is used when the project has no archetypes/default.md.
// It matches what `hugo new` writes.
const builtinArchetype = `+++
title = '{{ replace .Name "-" " " | title }}'
date = '{{ .Date }}'
draft = true
+++
`

// maxSlugLength caps the file name derived from a page name.
const maxSlugLength = 80

var (
	ErrInvalidName          = errors.New("page name has no usable letters or digits")
	ErrInvalidFolder        = errors.New("folder is not valid")
	ErrUnknownArchetype     = errors.New("archetype not found")
	ErrUnsupportedArchetype = errors.New("archetype uses template features that are not supported")
	ErrInvalidArchetype     = errors.New("archetype produces invalid frontmatter")
	ErrPageExists           = errors.New("a page with this name already exists")
)

// Archetype is a page template a new page can start from.
type Archetype struct {
	// Name is the template's file name without the extension.
	Name string `json:"name"`
	// Builtin is true for the fallback used when the project has none.
	Builtin bool `json:"builtin,omitempty"`
}

// Archetypes lists the page templates of a project: the markdown files in
// its archetypes folder, with "default" first. Hugo also allows folders
// (page bundles) there; they are skipped. The built-in default is listed
// when the project does not define one.
func Archetypes(fsys fs.FS) []Archetype {
	var out []Archetype
	hasDefault := false

	entries, _ := fs.ReadDir(fsys, archetypeDir) // a missing folder just means none
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || skip(name) || strings.ToLower(path.Ext(name)) != ".md" {
			continue
		}
		base := strings.TrimSuffix(name, path.Ext(name))
		hasDefault = hasDefault || base == defaultArchetype
		out = append(out, Archetype{Name: base})
	}
	if !hasDefault {
		out = append(out, Archetype{Name: defaultArchetype, Builtin: true})
	}

	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i].Name, out[j].Name
		if (a == defaultArchetype) != (b == defaultArchetype) {
			return a == defaultArchetype
		}
		return strings.ToLower(a) < strings.ToLower(b)
	})
	return out
}

// NewPage describes a page to create.
type NewPage struct {
	// Archetype is the template name. Empty picks the one named after the
	// folder's first segment if it exists, otherwise the default.
	Archetype string
	// Folder is relative to the content directory. Empty means its root.
	Folder string
	// Name is what the user typed. The file name is derived from it.
	Name string
}

// CreatePage renders the archetype and writes the result as a new page in
// the content directory of root. It never overwrites an existing file.
// now is the creation time given to the template.
func CreatePage(root *os.Root, contentDir string, req NewPage, now time.Time) (*Page, error) {
	slug := Slugify(req.Name)
	if slug == "" {
		return nil, ErrInvalidName
	}
	folder, err := cleanFolder(req.Folder)
	if err != nil {
		return nil, err
	}

	fsys := root.FS()
	name := pickArchetype(fsys, req.Archetype, folder)
	source, err := archetypeSource(fsys, name)
	if err != nil {
		return nil, err
	}

	rendered, err := renderArchetype(source, archetypeData{
		Name: slug,
		Type: sectionOf(folder, name),
		Date: now.Format(time.RFC3339),
		File: archetypeFile{ContentBaseName: slug, BaseFileName: slug},
	}, now)
	if err != nil {
		return nil, err
	}

	// Never write a page the editor could not open again.
	doc, err := frontmatter.Split(rendered)
	if err == nil {
		_, err = doc.Fields()
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidArchetype, err)
	}

	dir := path.Join(contentDir, folder)
	file := path.Join(dir, slug+".md")
	if err := root.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("creating %s: %w", dir, err)
	}
	if err := writeNew(root, file, rendered); err != nil {
		return nil, err
	}
	return ReadPage(fsys, contentDir, file)
}

// writeNew creates name with data, failing if it already exists. A write
// that fails half way removes the file, so no partial page is left behind.
func writeNew(root *os.Root, name string, data []byte) error {
	f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, fs.ErrExist) {
		return ErrPageExists
	}
	if err != nil {
		return fmt.Errorf("creating %s: %w", name, err)
	}

	_, err = f.Write(data)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = root.Remove(name)
		return fmt.Errorf("writing %s: %w", name, err)
	}
	return nil
}

// Slugify turns a page name into a file name: lower case letters and digits
// joined by single hyphens, at most maxSlugLength characters.
func Slugify(name string) string {
	var b strings.Builder
	pendingHyphen := false
	n := 0 // runes written so far, hyphens included

	for _, r := range strings.ToLower(name) {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			pendingHyphen = n > 0
			continue
		}
		needed := n + 1
		if pendingHyphen {
			needed++
		}
		if needed > maxSlugLength {
			break
		}
		if pendingHyphen {
			b.WriteByte('-')
			pendingHyphen = false
		}
		b.WriteRune(r)
		n = needed
	}
	return b.String()
}

// cleanFolder validates a folder relative to the content directory and
// returns it cleaned. Hidden folders and node_modules are refused, because
// the tree would never show a page created in them.
func cleanFolder(folder string) (string, error) {
	folder = strings.Trim(strings.ReplaceAll(strings.TrimSpace(folder), `\`, "/"), "/")
	if folder == "" {
		return ".", nil
	}
	clean := path.Clean(folder)
	if !fs.ValidPath(clean) || clean == "." {
		return "", ErrInvalidFolder
	}
	for _, segment := range strings.Split(clean, "/") {
		if skip(segment) {
			return "", ErrInvalidFolder
		}
	}
	return clean, nil
}

// pickArchetype returns the requested archetype, or when none was asked for
// the one named after the folder's first segment if it exists.
func pickArchetype(fsys fs.FS, requested, folder string) string {
	if requested != "" {
		return requested
	}
	section, _, _ := strings.Cut(folder, "/")
	if section != "." {
		for _, a := range Archetypes(fsys) {
			if a.Name == section && !a.Builtin {
				return section
			}
		}
	}
	return defaultArchetype
}

// sectionOf is the value of .Type in the template: the archetype's name,
// or the folder's first segment for the default archetype.
func sectionOf(folder, archetype string) string {
	if archetype != defaultArchetype {
		return archetype
	}
	if section, _, _ := strings.Cut(folder, "/"); section != "." {
		return section
	}
	return defaultArchetype
}

// archetypeSource returns the template text. Only names from Archetypes are
// accepted, so a request can never read an arbitrary file.
func archetypeSource(fsys fs.FS, name string) (string, error) {
	for _, a := range Archetypes(fsys) {
		if a.Name != name {
			continue
		}
		if a.Builtin {
			return builtinArchetype, nil
		}
		data, err := fs.ReadFile(fsys, path.Join(archetypeDir, name+".md"))
		if err != nil {
			return "", fmt.Errorf("reading archetype %s: %w", name, err)
		}
		return string(data), nil
	}
	return "", ErrUnknownArchetype
}

type archetypeFile struct {
	ContentBaseName string
	BaseFileName    string
}

// archetypeData is what a template can refer to, a subset of what Hugo
// offers: .Name, .Type, .Date and .File.ContentBaseName.
type archetypeData struct {
	Name string
	Type string
	Date string
	File archetypeFile
}

func renderArchetype(source string, data archetypeData, now time.Time) ([]byte, error) {
	funcs := template.FuncMap{
		"replace": func(input, old, replacement string) string { return strings.ReplaceAll(input, old, replacement) },
		"title":   titleCase,
		"lower":   strings.ToLower,
		"upper":   strings.ToUpper,
		"now":     func() time.Time { return now },
	}

	tmpl, err := template.New("archetype").Funcs(funcs).Option("missingkey=error").Parse(source)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnsupportedArchetype, err)
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnsupportedArchetype, err)
	}
	return out.Bytes(), nil
}

// titleCase upper-cases the first letter of every word. Hugo's own title
// function also keeps small words like "of" in lower case; this does not.
func titleCase(s string) string {
	prev := ' '
	return strings.Map(func(r rune) rune {
		defer func() { prev = r }()
		if unicode.IsSpace(prev) {
			return unicode.ToUpper(r)
		}
		return r
	}, s)
}
