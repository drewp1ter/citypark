package news

import (
	"errors"
	"log/slog"
	"maps"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/richtext"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/upload"
	"github.com/drewp1ter/citypark-go/internal/validate"
)

const (
	maxBodyLength = 50000
	maxPercent    = 100
	dateLayout    = "2006-01-02"
	// validationMessage is the top-level message of validate.Struct, used here for the checks it cannot express.
	validationMessage = "Ошибка валидации, пожалуйста проверьте введенные вами данные"
)

var (
	positionRe = regexp.MustCompile(`^(\d{1,3})% (\d{1,3})%$`)
	slugRe     = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

	errSlugTaken = httpx.NewError(http.StatusConflict, "Новость с таким адресом уже есть")
)

// Handler serves the news. It takes *sqlx.DB because writes need a transaction.
type Handler struct {
	db              *sqlx.DB
	uploads         *upload.Storage
	tokens          *auth.Tokens
	clock           clock.Clock
	defaultPageSize int
}

func NewHandler(db *sqlx.DB, uploads *upload.Storage, tokens *auth.Tokens, c clock.Clock, defaultPageSize int) *Handler {
	return &Handler{db: db, uploads: uploads, tokens: tokens, clock: c, defaultPageSize: defaultPageSize}
}

type photoInput struct {
	URL      string `json:"url"`
	Alt      string `json:"alt" validate:"max=200"`
	Position string `json:"position" validate:"max=100"`
}

// saveRequest is the body of POST /news and PUT /news/{id}.
type saveRequest struct {
	Slug       string     `json:"slug" validate:"max=100"`
	Date       string     `json:"date"`
	Title      string     `json:"title" validate:"required,max=200"`
	ShortTitle string     `json:"shortTitle" validate:"max=100"`
	Excerpt    string     `json:"excerpt" validate:"max=300"`
	Body       string     `json:"body"`
	Photo      photoInput `json:"photo"`
}

type photoResponse struct {
	URL      string `json:"url"`
	Alt      string `json:"alt"`
	Position string `json:"position"`
}

// summary is an item without the body, as the list shows it.
type summary struct {
	ID         int64         `json:"id"`
	Slug       string        `json:"slug"`
	Date       string        `json:"date"`
	Title      string        `json:"title"`
	ShortTitle string        `json:"shortTitle"`
	Excerpt    string        `json:"excerpt"`
	Photo      photoResponse `json:"photo"`
	UpdatedAt  time.Time     `json:"updatedAt"`
}

type fullResponse struct {
	ID         int64         `json:"id"`
	Slug       string        `json:"slug"`
	Date       string        `json:"date"`
	Title      string        `json:"title"`
	ShortTitle string        `json:"shortTitle"`
	Excerpt    string        `json:"excerpt"`
	Body       string        `json:"body"`
	Photo      photoResponse `json:"photo"`
	UpdatedAt  time.Time     `json:"updatedAt"`
}

type listResponse struct {
	News  []summary `json:"news"`
	Count int       `json:"count"`
}

// Routes registers GET /news and GET /news/{slug} (public), POST /news, PUT /news/{id} and DELETE /news/{id}
// (admin).
func (h *Handler) Routes(r chi.Router) {
	r.Get("/news", httpx.Handle(h.list))
	r.Get("/news/{slug}", httpx.Handle(h.get))
	r.Group(func(r chi.Router) {
		r.Use(h.tokens.RequireRole(auth.RoleAdmin))
		r.Post("/news", httpx.Handle(h.create))
		r.Put("/news/{id}", httpx.Handle(h.update))
		r.Delete("/news/{id}", httpx.Handle(h.remove))
	})
}

func (h *Handler) summary(it Item) summary {
	return summary{
		ID: it.ID, Slug: it.Slug, Date: it.Date.Format(dateLayout), Title: it.Title, ShortTitle: it.ShortTitle,
		Excerpt:   it.Excerpt,
		Photo:     photoResponse{URL: h.uploads.Public(it.PhotoURL), Alt: it.PhotoAlt, Position: it.PhotoPosition},
		UpdatedAt: it.UpdatedAt,
	}
}

func (h *Handler) full(it Item) fullResponse {
	s := h.summary(it)
	return fullResponse{
		ID: s.ID, Slug: s.Slug, Date: s.Date, Title: s.Title, ShortTitle: s.ShortTitle, Excerpt: s.Excerpt,
		Body: it.Body, Photo: s.Photo, UpdatedAt: s.UpdatedAt,
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	limit, offset, err := httpx.Pagination(r, h.defaultPageSize)
	if err != nil {
		return err
	}
	items, count, err := NewStore(h.db).List(r.Context(), limit, offset)
	if err != nil {
		return err
	}
	out := make([]summary, 0, len(items))
	for _, it := range items {
		out = append(out, h.summary(it))
	}
	httpx.JSON(w, http.StatusOK, listResponse{News: out, Count: count})
	return nil
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) error {
	it, err := NewStore(h.db).GetBySlug(r.Context(), chi.URLParam(r, "slug"))
	if errors.Is(err, store.ErrNotFound) {
		return httpx.NotFound()
	}
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, h.full(it))
	return nil
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) error {
	var req saveRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	trim(&req)
	in, err := h.check(req)
	if err != nil {
		return err
	}

	var created Item
	err = store.InTx(r.Context(), h.db, func(tx *sqlx.Tx) error {
		var err error
		created, err = NewStore(tx).Create(r.Context(), in)
		return err
	})
	if err != nil {
		return writeErr(err)
	}
	httpx.JSON(w, http.StatusCreated, h.full(created))
	return nil
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	var req saveRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	trim(&req)
	in, err := h.check(req)
	if err != nil {
		return err
	}

	var (
		saved    Item
		oldPhoto string
	)
	err = store.InTx(r.Context(), h.db, func(tx *sqlx.Tx) error {
		var err error
		saved, oldPhoto, err = NewStore(tx).Update(r.Context(), id, in)
		return err
	})
	if err != nil {
		return writeErr(err)
	}
	if oldPhoto != saved.PhotoURL {
		h.removePhoto(saved.ID, oldPhoto)
	}
	httpx.JSON(w, http.StatusOK, h.full(saved))
	return nil
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	photo, err := NewStore(h.db).Delete(r.Context(), id)
	if err != nil {
		return writeErr(err)
	}
	h.removePhoto(id, photo)
	httpx.NoContent(w)
	return nil
}

// removePhoto deletes a photo file that no news item uses any more. A failure is logged, not returned.
func (h *Handler) removePhoto(id int64, path string) {
	if err := h.uploads.Remove(path); err != nil {
		slog.Error("remove news photo", "news", id, "path", path, "error", err)
	}
}

// trim runs before validation so that a whitespace-only title fails "required".
func trim(req *saveRequest) {
	for _, s := range []*string{
		&req.Slug, &req.Date, &req.Title, &req.ShortTitle, &req.Excerpt,
		&req.Photo.URL, &req.Photo.Alt, &req.Photo.Position,
	} {
		*s = strings.TrimSpace(*s)
	}
}

// check validates req and returns the Input to store. All invalid fields are reported together.
func (h *Handler) check(req saveRequest) (Input, error) {
	fields := map[string]string{}
	var top *httpx.Error
	if err := validate.Struct(req); err != nil {
		if !errors.As(err, &top) {
			return Input{}, err
		}
		maps.Copy(fields, top.Fields)
	}

	if _, bad := fields["slug"]; !bad && req.Slug != "" && !slugRe.MatchString(req.Slug) {
		fields["slug"] = "только латиница, цифры и дефисы"
	}

	date, ok := h.parseDate(req.Date)
	if !ok {
		fields["date"] = "дата в формате ГГГГ-ММ-ДД"
	}

	photoURL, err := h.uploads.Stored(req.Photo.URL)
	if err != nil {
		fields["photo.url"] = "недопустимый адрес фото"
	}
	if _, bad := fields["photo.position"]; !bad && !validPosition(req.Photo.Position) {
		fields["photo.position"] = "формат «50% 30%»"
	}

	body := richtext.Sanitize(richtext.Article, req.Body)
	if richtext.TextLength(body) > maxBodyLength {
		fields["body"] = "не более 50000 символов"
	}

	if len(fields) > 0 {
		if top == nil {
			top = httpx.NewError(http.StatusBadRequest, validationMessage)
		}
		top.Fields = fields
		return Input{}, top
	}
	return Input{
		Slug: req.Slug, Date: date, Title: req.Title, ShortTitle: req.ShortTitle, Excerpt: req.Excerpt, Body: body,
		PhotoURL: photoURL, PhotoAlt: req.Photo.Alt, PhotoPosition: req.Photo.Position,
	}, nil
}

// parseDate turns "" into today (in the clock's location) and "YYYY-MM-DD" into that day at 00:00 UTC.
func (h *Handler) parseDate(s string) (time.Time, bool) {
	if s == "" {
		now := h.clock.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC), true
	}
	d, err := time.Parse(dateLayout, s)
	return d, err == nil
}

// writeErr maps store errors of a write to API errors.
func writeErr(err error) error {
	switch {
	case store.IsUniqueViolation(err):
		return errSlugTaken
	case errors.Is(err, store.ErrNotFound):
		return httpx.NotFound()
	}
	return err
}

// validPosition accepts "" or "<0-100>% <0-100>%".
func validPosition(p string) bool {
	if p == "" {
		return true
	}
	m := positionRe.FindStringSubmatch(p)
	if m == nil {
		return false
	}
	for _, part := range m[1:] {
		if n, _ := strconv.Atoi(part); n > maxPercent {
			return false
		}
	}
	return true
}
