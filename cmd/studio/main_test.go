package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewLoggerJSONAndLevel(t *testing.T) {
	var buf bytes.Buffer
	l, err := newLogger(&buf, "warn", "json")
	if err != nil {
		t.Fatal(err)
	}
	l.Info("hidden")
	l.Warn("shown", "k", "v")
	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("not json: %q: %v", buf.String(), err)
	}
	if rec["msg"] != "shown" || rec["k"] != "v" {
		t.Fatalf("unexpected record: %v", rec)
	}
}

func TestNewLoggerText(t *testing.T) {
	var buf bytes.Buffer
	l, err := newLogger(&buf, "debug", "text")
	if err != nil {
		t.Fatal(err)
	}
	l.Debug("hi")
	if !strings.Contains(buf.String(), "msg=hi") {
		t.Fatalf("got %q", buf.String())
	}
}

func TestNewLoggerInvalid(t *testing.T) {
	if _, err := newLogger(&bytes.Buffer{}, "loud", "text"); err == nil {
		t.Fatal("want level error")
	}
	if _, err := newLogger(&bytes.Buffer{}, "info", "xml"); err == nil {
		t.Fatal("want format error")
	}
}
