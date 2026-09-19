// Package frontmatter reads and writes the metadata block at the top of a
// content file, preserving its original format and any fields the editor
// does not understand.
package frontmatter

import (
	"bytes"
	"errors"
	"fmt"
)

// Format is the markup used for a file's frontmatter block.
type Format string

const (
	// FormatNone means the file has no frontmatter block.
	FormatNone Format = ""
	FormatYAML Format = "yaml"
	FormatTOML Format = "toml"
)

// ErrUnterminated means the file opens a frontmatter block but never closes it.
var ErrUnterminated = errors.New("frontmatter block is not closed")

var fences = map[Format][]byte{
	FormatYAML: []byte("---"),
	FormatTOML: []byte("+++"),
}

// Document is a content file split into its frontmatter and its body.
type Document struct {
	Format Format
	// Meta is the frontmatter block without its fences, exactly as written.
	Meta []byte
	// Body is everything after the frontmatter block.
	Body []byte
}

var bom = []byte{0xEF, 0xBB, 0xBF}

// Split divides a content file into its frontmatter block and its body
func Split(content []byte) (*Document, error) {
	content = bytes.TrimPrefix(content, bom)

	format, rest, ok := openingFence(content)
	if !ok {
		return &Document{Format: FormatNone, Body: content}, nil
	}

	meta, body, ok := untilClosingFence(rest, fences[format])
	if !ok {
		return nil, fmt.Errorf("%w: expected a closing %s", ErrUnterminated, fences[format])
	}

	return &Document{Format: format, Meta: meta, Body: body}, nil
}

// openingFence reports which fence the content opens with, and returns
// everything after that first line.
func openingFence(content []byte) (Format, []byte, bool) {
	for format, fence := range fences {
		rest, ok := cutLine(content, fence)
		if ok {
			return format, rest, true
		}
	}
	return FormatNone, nil, false
}

// cutLine matches a line consisting of exactly want, and returns the
// content after it.
func cutLine(content, want []byte) ([]byte, bool) {
	line, rest := splitLine(content)
	if !bytes.Equal(bytes.TrimRight(line, "\r"), want) {
		return nil, false
	}
	return rest, true
}

// splitLine returns the first line of content without its newline,
// and everything after it.
func splitLine(content []byte) (line, rest []byte) {
	i := bytes.IndexByte(content, '\n')
	if i < 0 {
		return content, nil
	}
	return content[:i], content[i+1:]
}

// untilClosingFence scans line by line for a closing fence, returning the
// lines before it and the content after it.
func untilClosingFence(content, fence []byte) (meta, body []byte, ok bool) {
	var consumed int
	rest := content

	for len(rest) > 0 {
		line, next := splitLine(rest)
		if bytes.Equal(bytes.TrimRight(line, "\r"), fence) {
			return content[:consumed], next, true
		}
		consumed += len(rest) - len(next)
		rest = next
	}
	return nil, nil, false
}
