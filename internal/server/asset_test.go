package server

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

var testPNG = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00")

func (a *api) upload(id, name string, data []byte) *httptest.ResponseRecorder {
	a.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", name)
	if err != nil {
		a.t.Fatal(err)
	}
	_, _ = fw.Write(data)
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:7777/api/asset", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if id != "" {
		req.Header.Set("X-Project-ID", id)
	}
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)
	return rec
}

func TestAssetUpload(t *testing.T) {
	a := newAPI(t)
	dir := newHugoDir(t)
	opened := decodeProject(t, a.open(dir))

	rec := a.upload(opened.ID, "Cat.png", testPNG)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d (body: %s)", rec.Code, rec.Body)
	}
	if _, err := os.Stat(filepath.Join(dir, "static/images/cat.png")); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"/images/cat.png"`)) {
		t.Errorf("body = %s", rec.Body)
	}

	rec = a.upload(opened.ID, "Cat.png", testPNG)
	if rec.Code != http.StatusCreated || !bytes.Contains(rec.Body.Bytes(), []byte(`"/images/cat-1.png"`)) {
		t.Fatalf("duplicate: status = %d (body: %s)", rec.Code, rec.Body)
	}

	cases := []struct {
		name string
		file string
		data []byte
		id   string
		code int
	}{
		{"not image", "x.png", []byte("hello"), opened.ID, http.StatusUnsupportedMediaType},
		{"svg", "x.svg", []byte("<svg/>"), opened.ID, http.StatusUnsupportedMediaType},
		{"too large", "big.png", append(append([]byte{}, testPNG...), make([]byte, 11<<20)...), opened.ID, http.StatusRequestEntityTooLarge},
		{"bad name", "noext", testPNG, opened.ID, http.StatusBadRequest},
		{"stale id", "y.png", testPNG, "wrong", http.StatusConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := a.upload(tc.id, tc.file, tc.data)
			if rec.Code != tc.code {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tc.code, rec.Body)
			}
		})
	}
}

func TestAssetUploadNotMultipart(t *testing.T) {
	a := newAPI(t)
	opened := decodeProject(t, a.open(newHugoDir(t)))
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:7777/api/asset", bytes.NewReader([]byte("x")))
	req.Header.Set("X-Project-ID", opened.ID)
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}
