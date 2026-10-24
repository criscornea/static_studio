package content

import (
	"errors"
	"io/fs"
	"path"
	"regexp"
	"strings"
	"unicode/utf8"
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

// Search finds pages under dirs whose title, tags or body contain query,
// ignoring case. Title matches come first, then path order.
func Search(fsys fs.FS, dirs []string, query string) (*Results, error) {
	return (*Index)(nil).Search(fsys, dirs, query)
}

// Search is like the package-level Search but matches against text cached in
// ix, so unchanged pages are not read again.
func (ix *Index) Search(fsys fs.FS, dirs []string, query string) (*Results, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, ErrEmptyQuery
	}
	m := newMatcher(query)

	var titleHits, bodyHits []Hit
	seen := map[string]bool{}
	for _, dir := range dirs {
		if err := ix.searchDir(fsys, dir, m, seen, &titleHits, &bodyHits); err != nil {
			return nil, err
		}
	}

	// Each dir is walked in path order, so each group is sorted per dir.
	// Never nil, so an empty result encodes as [] and not null.
	hits := append(append([]Hit{}, titleHits...), bodyHits...)

	res := &Results{Hits: hits}
	if len(hits) > SearchLimit {
		res.Hits = hits[:SearchLimit]
		res.Truncated = true
	}
	return res, nil
}

// searchDir appends the matching pages under dir to titleHits and bodyHits.
func (ix *Index) searchDir(fsys fs.FS, dir string, m matcher, seen map[string]bool, titleHits, bodyHits *[]Hit) error {
	return fs.WalkDir(fsys, dir, func(name string, entry fs.DirEntry, err error) error {
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
		if seen[name] {
			return nil
		}
		seen[name] = true

		hit, inTitle, ok := ix.matchPage(fsys, name, entry, m)
		switch {
		case !ok:
		case inTitle:
			*titleHits = append(*titleHits, hit)
		default:
			*bodyHits = append(*bodyHits, hit)
		}
		return nil
	})
}

// matchPage reports whether the page at name matches re. inTitle is true when
// the title or a tag matches. A page with broken frontmatter is searched as
// plain text, so a typo in its settings does not hide it.
func (ix *Index) matchPage(fsys fs.FS, name string, entry fs.DirEntry, m matcher) (hit Hit, inTitle, ok bool) {
	info, err := entry.Info()
	if err != nil {
		return Hit{}, false, false
	}
	e := ix.lookup(fsys, name, info)
	if e == nil {
		return Hit{}, false, false
	}

	hit = Hit{Path: name, Title: e.title, Draft: e.draft}
	inTitle = m.re.MatchString(e.title) || m.re.MatchString(e.tags)
	body := e.body

	loc := m.find(e)
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

// matcher is a compiled case-insensitive query.
type matcher struct {
	re *regexp.Regexp
	// lower is the lower-cased query when it is pure ASCII, else empty.
	lower string
}

func newMatcher(query string) matcher {
	m := matcher{re: regexp.MustCompile("(?i)" + regexp.QuoteMeta(query))}
	if isASCII(query) {
		m.lower = strings.ToLower(query)
	}
	return m
}

// find returns the first match in the page body. Pure-ASCII bodies use a
// plain substring search on the cached lower-case copy; the answer is the
// same as the regexp's.
func (m matcher) find(e *indexed) []int {
	if m.lower != "" && e.lower != "" {
		i := strings.Index(e.lower, m.lower)
		if i < 0 {
			return nil
		}
		return []int{i, i + len(m.lower)}
	}
	return m.re.FindStringIndex(e.body)
}
