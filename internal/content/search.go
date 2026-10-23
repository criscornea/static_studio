package content

import (
	"errors"
	"io/fs"
	"path"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/criscornea/static_studio/internal/frontmatter"
)

var ErrEmptyQuery = errors.New("empty search query")

const (
	// SearchLimit caps how many hits Search returns.
	SearchLimit = 50

	// snippetBefore and snippetAfter are how many bytes of context a snippet
	// keeps on each side of the match.
	snippetBefore = 40
	snippetAfter  = 80
)

// Hit is one page that matches a search.
type Hit struct {
	Path  string `json:"path"`
	Title string `json:"title"`
	Draft bool   `json:"draft"`
	// Snippet is the text around the first match in the body, with
	// whitespace collapsed. It is empty when only the title or tags match.
	Snippet string `json:"snippet,omitempty"`
}

// Results is the outcome of a search.
type Results struct {
	Hits []Hit `json:"hits"`
	// Truncated is true when more than SearchLimit pages matched.
	Truncated bool `json:"truncated"`
}

// Search finds pages under dir whose title, tags or body contain query,
// ignoring case. Title matches come first, then path order. It reads every
// page on each call, which is fine for a personal site.
func Search(fsys fs.FS, dir, query string) (*Results, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, ErrEmptyQuery
	}
	re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(query))

	var titleHits, bodyHits []Hit
	err := fs.WalkDir(fsys, dir, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			// An unreadable entry must not fail the whole search.
			if entry != nil && entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			if name != dir && skip(entry.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if skip(entry.Name()) || !editableExts[strings.ToLower(path.Ext(name))] {
			return nil
		}

		hit, inTitle, ok := matchPage(fsys, name, entry, re)
		switch {
		case !ok:
		case inTitle:
			titleHits = append(titleHits, hit)
		default:
			bodyHits = append(bodyHits, hit)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// WalkDir visits in path order, so each group is already sorted.
	// Never nil, so an empty result encodes as [] and not null.
	hits := append(append([]Hit{}, titleHits...), bodyHits...)

	res := &Results{Hits: hits}
	if len(hits) > SearchLimit {
		res.Hits = hits[:SearchLimit]
		res.Truncated = true
	}
	return res, nil
}

// matchPage reports whether the page at name matches re. inTitle is true when
// the title or a tag matches. A page with broken frontmatter is searched as
// plain text, so a typo in its settings does not hide it.
func matchPage(fsys fs.FS, name string, entry fs.DirEntry, re *regexp.Regexp) (hit Hit, inTitle, ok bool) {
	info, err := entry.Info()
	if err != nil || info.Size() > maxPageSize {
		return Hit{}, false, false
	}
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return Hit{}, false, false
	}

	hit = Hit{Path: name}
	body := string(data)

	if doc, err := frontmatter.Split(data); err == nil {
		body = string(doc.Body)
		if fields, err := doc.Fields(); err == nil {
			hit.Title = fields.Title
			hit.Draft = fields.Draft
			inTitle = re.MatchString(fields.Title) || re.MatchString(strings.Join(fields.Tags, "\n"))
		}
	}

	loc := re.FindStringIndex(body)
	if loc != nil {
		hit.Snippet = snippet(body, loc[0], loc[1])
	}
	return hit, inTitle, inTitle || loc != nil
}

// snippet returns the text around body[start:end], on rune boundaries and
// with runs of whitespace collapsed to one space.
func snippet(body string, start, end int) string {
	from := max(start-snippetBefore, 0)
	to := min(end+snippetAfter, len(body))
	for from > 0 && !utf8.RuneStart(body[from]) {
		from--
	}
	for to < len(body) && !utf8.RuneStart(body[to]) {
		to++
	}

	out := strings.Join(strings.Fields(body[from:to]), " ")
	if from > 0 {
		out = "…" + out
	}
	if to < len(body) {
		out += "…"
	}
	return out
}
