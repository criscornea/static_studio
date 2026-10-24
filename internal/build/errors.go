package build

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Problem is one error Hugo reported for a file.
type Problem struct {
	// File is slash-separated and relative to the project root.
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column,omitempty"`
	Message string `json:"message"`
}

// location matches Hugo's quoted "path:line:col" marker.
var location = regexp.MustCompile(`"((?:[A-Za-z]:)?[^"]+?):(\d+)(?::(\d+))?"`)

// logPrefix matches Hugo's leading level and timestamp, such as
// "ERROR 2024/01/02 03:04:05 " or "Error: ".
var logPrefix = regexp.MustCompile(`^(?:ERROR|Error|WARN)(?: \d{4}/\d\d/\d\d \d\d:\d\d:\d\d)?:? `)

// ParseLine extracts a file error from one line of Hugo output. Paths inside
// root are made relative to it; paths outside root are ignored.
func ParseLine(root, line string) (Problem, bool) {
	m := location.FindStringSubmatchIndex(line)
	if m == nil {
		return Problem{}, false
	}

	file, ok := relative(root, line[m[2]:m[3]])
	if !ok {
		return Problem{}, false
	}
	num, err := strconv.Atoi(line[m[4]:m[5]])
	if err != nil {
		return Problem{}, false
	}
	col := 0
	if m[6] >= 0 {
		col, _ = strconv.Atoi(line[m[6]:m[7]])
	}

	tail := line[m[1]:]
	tail = strings.TrimPrefix(tail, ":")
	rest := strings.TrimRight(line[:m[0]], " ") + " " + strings.TrimSpace(tail)
	rest = logPrefix.ReplaceAllString(strings.TrimSpace(rest), "")
	rest = strings.TrimLeft(rest, ": ")
	if rest == "" {
		rest = strings.TrimSpace(line)
	}

	return Problem{File: file, Line: num, Column: col, Message: rest}, true
}

func relative(root, p string) (string, bool) {
	if filepath.IsAbs(p) || filepath.VolumeName(p) != "" {
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return "", false
		}
		p = rel
	}
	p = filepath.ToSlash(filepath.Clean(p))
	if p == ".." || strings.HasPrefix(p, "../") || p == "." {
		return "", false
	}

	return p, true
}
