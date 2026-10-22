package upload

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/httpx"
)

const (
	maxFileBytes = 10 << 20
	maxBodyBytes = maxFileBytes + 64<<10
)

// Handler serves POST /uploads.
type Handler struct {
	storage *Storage
	tokens  *auth.Tokens
}

func NewHandler(storage *Storage, tokens *auth.Tokens) *Handler {
	return &Handler{storage: storage, tokens: tokens}
}

type uploadResponse struct {
	URL string `json:"url"`
}

var (
	errTooLarge = httpx.NewError(http.StatusRequestEntityTooLarge, "Файл больше 10 МБ")
	errNoFile   = httpx.BadRequest("Выберите файл")
	errBadType  = httpx.BadRequest("Допустимы только JPEG, PNG и WebP")
)

// Routes registers POST /uploads (admin).
func (h *Handler) Routes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(h.tokens.RequireRole(auth.RoleAdmin))
		r.Post("/uploads", httpx.Handle(h.create))
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := r.ParseMultipartForm(maxFileBytes); err != nil {
		return parseError(err)
	}
	defer r.MultipartForm.RemoveAll()

	file, header, err := r.FormFile("file")
	if err != nil {
		return errNoFile
	}
	defer file.Close()
	if header.Size > maxFileBytes {
		return errTooLarge
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return parseError(err)
	}
	ext, ok := extension(http.DetectContentType(data))
	if !ok {
		return errBadType
	}

	path, err := h.storage.Save(time.Now(), data, ext)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, uploadResponse{URL: h.storage.Public(path)})
	return nil
}

// parseError maps a body-reading failure to the error shown to the user.
func parseError(err error) error {
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		return errTooLarge
	}
	return errNoFile
}

// extension returns the file extension for a detected image type.
func extension(contentType string) (string, bool) {
	switch contentType {
	case "image/jpeg":
		return "jpg", true
	case "image/png":
		return "png", true
	case "image/webp":
		return "webp", true
	}
	return "", false
}
