package ssg

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strings"
)

// ErrConfigChanged is returned when the config file changed after it was read.
var ErrConfigChanged = errors.New("the config file changed since it was read; reload and try again")

// BaseURLConfig is a Hugo config file read for editing its top-level baseURL.
type BaseURLConfig struct {
	// File is the slash-separated path relative to the project root.
	File string
	// Current is the existing baseURL, or "" when the file does not define one.
	Current string
	// Warnings say where another setting would override the file's baseURL.
	Warnings []string

	data []byte
	mode fs.FileMode
}

var baseURLNames = []string{
	"hugo.toml", "hugo.yaml", "hugo.yml", "hugo.json",
	"config.toml", "config.yaml", "config.yml", "config.json",
	"config/_default/hugo.toml", "config/_default/hugo.yaml", "config/_default/hugo.yml", "config/_default/hugo.json",
	"config/_default/config.toml", "config/_default/config.yaml", "config/_default/config.yml", "config/_default/config.json",
}

func readCapped(root *os.Root, name string) ([]byte, fs.FileMode, error) {
	f, err := root.Open(name)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}
	if info.Size() > maxConfigSize {
		return nil, 0, fmt.Errorf("%s is larger than %d MiB", name, maxConfigSize>>20)
	}
	data, err := io.ReadAll(f)
	return data, info.Mode().Perm(), err
}

// ReadBaseURL finds the config file that defines the top-level baseURL (the
// root hugo.*/config.* files, then config/_default/*) and reads it. When none
// defines it, the primary configFile is used and Current is empty. Warnings
// list a HUGO_BASEURL variable and config/production files that would win.
func ReadBaseURL(root *os.Root, configFile string, getenv func(string) string) (*BaseURLConfig, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	var primary *BaseURLConfig
	for _, name := range append(append([]string{}, baseURLNames...), configFile) {
		data, mode, err := readCapped(root, name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", name, err)
		}
		span, err := locateBaseURL(path.Ext(name), data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		c := &BaseURLConfig{File: name, data: data, mode: mode}
		if span.found {
			c.Current = unquote(string(data[span.start:span.end]))
			c.Warnings = baseURLOverrides(root, name, getenv)
			return c, nil
		}
		if name == configFile {
			primary = c
		}
	}
	if primary == nil {
		return nil, fmt.Errorf("the config file %s cannot be read", configFile)
	}
	primary.Warnings = baseURLOverrides(root, primary.File, getenv)
	return primary, nil
}

func baseURLOverrides(root *os.Root, chosen string, getenv func(string) string) []string {
	var w []string
	if v := strings.TrimSpace(getenv("HUGO_BASEURL")); v != "" {
		w = append(w, "The HUGO_BASEURL environment variable is set and overrides the config file.")
	}
	for _, name := range baseURLNames {
		prod := strings.Replace(name, "config/_default/", "config/production/", 1)
		if prod == name {
			continue
		}
		data, _, err := readCapped(root, prod)
		if err != nil {
			continue
		}
		if span, err := locateBaseURL(path.Ext(prod), data); err == nil && span.found {
			w = append(w, prod+" sets baseURL for production builds and overrides "+chosen+".")
		}
	}
	return w
}

// Write replaces (or inserts) the top-level baseURL, keeping every other byte
// of the file, and replaces the file atomically. It refuses when the file on
// disk no longer matches what was read.
func (c *BaseURLConfig) Write(root *os.Root, baseURL string) error {
	if strings.ContainsAny(baseURL, "\"'\\\r\n") {
		return errors.New("the address contains a character that is not allowed")
	}
	ext := path.Ext(c.File)
	span, err := locateBaseURL(ext, c.data)
	if err != nil {
		return err
	}
	val := `"` + baseURL + `"`
	var out []byte
	if span.found {
		out = append(append(append([]byte{}, c.data[:span.start]...), val...), c.data[span.end:]...)
	} else {
		out, err = insertBaseURL(ext, c.data, val)
		if err != nil {
			return err
		}
	}

	var sfx [6]byte
	if _, err := rand.Read(sfx[:]); err != nil {
		return err
	}
	tmp := c.File + ".studio-" + hex.EncodeToString(sfx[:]) + ".tmp"
	f, err := root.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, c.mode)
	if err != nil {
		return err
	}
	if _, err := f.Write(out); err != nil {
		_ = f.Close()
		_ = root.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = root.Remove(tmp)
		return err
	}
	now, _, err := readCapped(root, c.File)
	if err != nil || !bytes.Equal(now, c.data) {
		_ = root.Remove(tmp)
		return ErrConfigChanged
	}
	if err := root.Rename(tmp, c.File); err != nil {
		_ = root.Remove(tmp)
		return err
	}
	return nil
}

type valueSpan struct {
	found      bool
	start, end int // the value token, quotes included
}

func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}

func locateBaseURL(ext string, data []byte) (valueSpan, error) {
	switch strings.ToLower(ext) {
	case ".toml":
		return locateLine(data, tomlKeyRe, true)
	case ".yaml", ".yml":
		return locateLine(data, yamlKeyRe, false)
	case ".json":
		return locateJSON(data)
	}
	return valueSpan{}, errors.New("unsupported config format")
}

var (
	tomlKeyRe = regexp.MustCompile(`(?i)^(?:baseurl|"baseurl"|'baseurl')[ \t]*=[ \t]*`)
	yamlKeyRe = regexp.MustCompile(`(?i)^(?:baseurl|"baseurl"|'baseurl'):(?:[ \t]+|$)`)
)

// locateLine finds a top-level "baseURL" line in TOML (before the first
// table header) or YAML (a key with no indentation).
func locateLine(data []byte, keyRe *regexp.Regexp, toml bool) (valueSpan, error) {
	pos := 0
	for pos < len(data) {
		end := bytes.IndexByte(data[pos:], '\n')
		if end < 0 {
			end = len(data) - pos
		}
		line := bytes.TrimRight(data[pos:pos+end], "\r")
		if toml && bytes.HasPrefix(bytes.TrimLeft(line, " \t"), []byte("[")) {
			return valueSpan{}, nil
		}
		if m := keyRe.FindIndex(line); m != nil && (toml || !isIndented(line)) {
			start := pos + m[1]
			vend, err := scanValue(data[start : pos+len(line)])
			if err != nil {
				return valueSpan{}, err
			}
			return valueSpan{true, start, start + vend}, nil
		}
		pos += end + 1
	}
	return valueSpan{}, nil
}

func isIndented(line []byte) bool { return len(line) > 0 && (line[0] == ' ' || line[0] == '\t') }

// scanValue returns the length of the scalar at the start of rest, which is
// the remainder of the line.
func scanValue(rest []byte) (int, error) {
	if len(rest) == 0 {
		return 0, errors.New("baseURL has no value on its line; edit it by hand")
	}
	switch rest[0] {
	case '"', '\'':
		if bytes.HasPrefix(rest, bytes.Repeat(rest[:1], 3)) {
			return 0, errors.New("baseURL is a multi-line string; edit it by hand")
		}
		for i := 1; i < len(rest); i++ {
			if rest[i] == '\\' && rest[0] == '"' {
				i++
			} else if rest[i] == rest[0] {
				return i + 1, nil
			}
		}
		return 0, errors.New("baseURL has an unterminated string")
	case '|', '>', '&', '*', '!', '[', '{':
		return 0, errors.New("baseURL has a value this editor cannot replace; edit it by hand")
	}
	n := len(rest)
	if i := bytes.Index(rest, []byte(" #")); i >= 0 {
		n = i
	}
	if i := bytes.Index(rest[:n], []byte("\t#")); i >= 0 {
		n = i
	}
	return len(bytes.TrimRight(rest[:n], " \t")), nil
}

func eol(data []byte) string {
	if bytes.Contains(data, []byte("\r\n")) {
		return "\r\n"
	}
	return "\n"
}

func insertBaseURL(ext string, data []byte, val string) ([]byte, error) {
	nl := eol(data)
	switch strings.ToLower(ext) {
	case ".toml":
		return append([]byte("baseURL = "+val+nl), data...), nil
	case ".yaml", ".yml":
		at := 0
		if bytes.HasPrefix(data, []byte("---")) {
			if i := bytes.IndexByte(data, '\n'); i >= 0 {
				at = i + 1
			} else {
				return append(append(data, nl...), "baseURL: "+val+nl...), nil
			}
		}
		line := "baseURL: " + val + nl
		return append(append(append([]byte{}, data[:at]...), line...), data[at:]...), nil
	}
	// JSON: insert as the first member of the top-level object.
	i := bytes.IndexByte(data, '{')
	if i < 0 {
		return nil, errors.New("the config file is not a JSON object")
	}
	rest := bytes.TrimLeft(data[i+1:], " \t\r\n")
	sep := ","
	if len(rest) == 0 || rest[0] == '}' {
		sep = ""
	}
	ins := nl + `  "baseURL": ` + val + sep
	return append(append(append([]byte{}, data[:i+1]...), ins...), data[i+1:]...), nil
}

// locateJSON finds the top-level "baseURL" member with a string value.
func locateJSON(data []byte) (valueSpan, error) {
	depth := 0
	for i := 0; i < len(data); i++ {
		switch data[i] {
		case '{', '[':
			depth++
		case '}', ']':
			depth--
		case '"':
			j := i + 1
			for j < len(data) && data[j] != '"' {
				if data[j] == '\\' {
					j++
				}
				j++
			}
			if j >= len(data) {
				return valueSpan{}, errors.New("the JSON is not valid")
			}
			tok := data[i+1 : j]
			k := j + 1
			for k < len(data) && bytes.IndexByte([]byte(" \t\r\n"), data[k]) >= 0 {
				k++
			}
			if depth == 1 && k < len(data) && data[k] == ':' && strings.EqualFold(string(tok), "baseurl") {
				k++
				for k < len(data) && bytes.IndexByte([]byte(" \t\r\n"), data[k]) >= 0 {
					k++
				}
				if k >= len(data) || data[k] != '"' {
					return valueSpan{}, errors.New("baseURL is not a string; edit it by hand")
				}
				e := k + 1
				for e < len(data) && data[e] != '"' {
					if data[e] == '\\' {
						e++
					}
					e++
				}
				if e >= len(data) {
					return valueSpan{}, errors.New("the JSON is not valid")
				}
				return valueSpan{true, k, e + 1}, nil
			}
			i = j
		}
	}
	return valueSpan{}, nil
}
