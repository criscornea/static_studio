package content

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

// MaxAssetSize is the largest image SaveAsset accepts.
const MaxAssetSize = 10 << 20

// assetSubdir is the folder inside the asset dir that receives uploads.
const assetSubdir = "images"

// maxAssetSuffix is how many "-N" suffixes SaveAsset tries on a name clash.
const maxAssetSuffix = 999

var (
	ErrAssetTooLarge = errors.New("image is too large")
	ErrAssetType     = errors.New("not a supported image")
	ErrAssetName     = errors.New("image name is not usable")
	ErrAssetExists   = errors.New("an image with this name already exists")
)

// imageTypes maps a sniffed content type to the extensions allowed for it.
// SVG is left out on purpose: it can carry scripts.
var imageTypes = map[string][]string{
	"image/png":  {".png"},
	"image/jpeg": {".jpg", ".jpeg"},
	"image/gif":  {".gif"},
	"image/webp": {".webp"},
}

// SaveAsset stores data as assetDir/images/<name> and returns the site URL
// path of the new file (for example "/images/photo.png"). Only png, jpeg, gif
// and webp are accepted; the type is checked from the bytes and must match the
// extension. It never overwrites: on a name clash it adds a numeric suffix
// ("photo-1.png"). A name with no usable letters gets a generated one. It
// refuses symlinked folders.
func SaveAsset(root *os.Root, assetDir, filename string, data []byte) (string, error) {
	if len(data) > MaxAssetSize {
		return "", ErrAssetTooLarge
	}
	name, err := cleanAssetName(filename)
	if err != nil {
		return "", err
	}
	exts, ok := imageTypes[http.DetectContentType(data)]
	if !ok || !contains(exts, path.Ext(name)) {
		return "", ErrAssetType
	}

	defer lockRoot(root)()
	dir := assetDir + "/" + assetSubdir
	// Reuse the symlink check: it walks every folder from assetDir down.
	if info, err := root.Lstat(assetDir); err == nil && (info.Mode()&fs.ModeSymlink != 0 || !info.IsDir()) {
		return "", ErrNotEditable
	}
	if err := checkParents(root, assetDir, dir+"/"+name); err != nil {
		return "", err
	}
	if err := root.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("creating %s: %w", dir, err)
	}
	stem := strings.TrimSuffix(name, path.Ext(name))
	ext := path.Ext(name)
	for i := 0; i <= maxAssetSuffix; i++ {
		cand := stem + ext
		if i > 0 {
			cand = fmt.Sprintf("%s-%d%s", stem, i, ext)
		}
		full := dir + "/" + cand
		err := writeFileAtomicNew(root, full, data, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("saving %s: %w", full, err)
		}
		return "/" + assetSubdir + "/" + cand, nil
	}
	return "", ErrAssetExists
}

// cleanAssetName reduces a client-supplied file name to a safe lowercase base
// name made of letters, digits, '-', '_' and a single extension.
func cleanAssetName(filename string) (string, error) {
	filename = strings.ReplaceAll(filename, "\\", "/")
	base := path.Base(filename)
	ext := strings.ToLower(path.Ext(base))
	stem := strings.TrimSuffix(base, path.Ext(base))
	var b strings.Builder
	for _, r := range strings.ToLower(stem) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ' || r == '.':
			b.WriteByte('-')
		}
	}
	s := strings.Trim(b.String(), "-_")
	if ext == "" || len(s) > 100 {
		return "", ErrAssetName
	}
	if s == "" {
		var rnd [4]byte
		if _, err := rand.Read(rnd[:]); err != nil {
			return "", ErrAssetName
		}
		s = "image-" + hex.EncodeToString(rnd[:])
	}
	return s + ext, nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
