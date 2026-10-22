// Package upload stores admin-uploaded images on the local disk and serves them.
package upload

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	uploadsPrefix = "/uploads/"
	imagesPrefix  = "/images/"
)

// ErrBadURL is returned by Stored for a photo URL the server does not accept.
var ErrBadURL = errors.New("bad photo url")

// Storage keeps uploaded images in a directory and builds their public URLs.
type Storage struct{ dir, base string }

func NewStorage(dir, publicBaseURL string) *Storage {
	return &Storage{dir: dir, base: strings.TrimRight(publicBaseURL, "/")}
}

// Save writes data under dir/YYYY/MM/<32 hex>.<ext> (time from now, random name from crypto/rand)
// and returns the stored path "/uploads/YYYY/MM/<name>.<ext>".
func (s *Storage) Save(now time.Time, data []byte, ext string) (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("random file name: %w", err)
	}
	rel := fmt.Sprintf("%04d/%02d/%s.%s", now.Year(), int(now.Month()), hex.EncodeToString(raw[:]), ext)
	full := filepath.Join(s.dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", fmt.Errorf("create upload directory: %w", err)
	}
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return "", fmt.Errorf("write upload: %w", err)
	}
	return uploadsPrefix + rel, nil
}

// Public turns a stored path into the URL sent to clients: "/uploads/..." → base + path; anything else unchanged.
func (s *Storage) Public(path string) string {
	if strings.HasPrefix(path, uploadsPrefix) {
		return s.base + path
	}
	return path
}

// Stored checks a photo URL received from a client and returns what goes into the database.
// Allowed: "" (no photo), "/images/<file>" (a photo shipped with the site), "/uploads/<path>",
// base + "/uploads/<path>". Anything else → ErrBadURL.
func (s *Storage) Stored(url string) (string, error) {
	if url == "" {
		return "", nil
	}
	path := url
	if rest, ok := strings.CutPrefix(url, s.base+uploadsPrefix); ok && s.base != "" {
		path = uploadsPrefix + rest
	}
	for _, prefix := range []string{uploadsPrefix, imagesPrefix} {
		if rest, ok := strings.CutPrefix(path, prefix); ok && safeRel(rest) {
			return path, nil
		}
	}
	return "", ErrBadURL
}

// Remove deletes the file of a stored "/uploads/..." path; other paths are ignored. A missing file is not an error.
// Call it after the database transaction that dropped the reference has committed; log errors, do not return them
// to the API user.
func (s *Storage) Remove(path string) error {
	rel, ok := strings.CutPrefix(path, uploadsPrefix)
	if !ok {
		return nil
	}
	if !safeRel(rel) {
		return ErrBadURL
	}
	err := os.Remove(filepath.Join(s.dir, filepath.FromSlash(rel)))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove upload: %w", err)
	}
	return nil
}

// Serve serves dir at /uploads/ (no directory listings, Cache-Control: public, max-age=31536000, immutable).
func (s *Storage) Serve() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel, ok := strings.CutPrefix(r.URL.Path, uploadsPrefix)
		if !ok || !safeRel(rel) {
			http.NotFound(w, r)
			return
		}
		f, err := os.Open(filepath.Join(s.dir, filepath.FromSlash(rel)))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeContent(w, r, info.Name(), info.ModTime(), f)
	})
}

// safeRel reports whether rel is a plain relative file path: not empty, not a directory,
// and without "..", backslashes, query strings, fragments or empty segments.
func safeRel(rel string) bool {
	if rel == "" || strings.HasSuffix(rel, "/") {
		return false
	}
	return !strings.ContainsAny(rel, `\?#`) && !strings.Contains(rel, "..") && !strings.Contains(rel, "//")
}
