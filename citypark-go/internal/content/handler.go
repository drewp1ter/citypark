package content

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/richtext"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/upload"
)

const (
	defaultTextLength     = 300
	defaultRichtextLength = 2000
	maxLines              = 50
	maxLineLength         = 200
	maxAltLength          = 200
	maxPercent            = 100
	// validationMessage is the top-level message of validate.Struct, used here for the checks it cannot express.
	validationMessage = "Ошибка валидации, пожалуйста проверьте введенные вами данные"
	wrongType         = "неверный тип значения"
)

// Placeholders are the names allowed inside {{...}} in a text. The site replaces each with a formatted price.
var Placeholders = []string{"minRoomPrice", "lateCheckoutPerHour", "smokingFine", "breakfastDelivery"}

var (
	placeholderRe = regexp.MustCompile(`\{\{\s*([a-zA-Z]+)\s*\}\}`)
	positionRe    = regexp.MustCompile(`^(\d{1,3})% (\d{1,3})%$`)
)

// Handler serves the content blocks. It takes *sqlx.DB because an image write needs a transaction.
type Handler struct {
	db      *sqlx.DB
	uploads *upload.Storage
	tokens  *auth.Tokens
}

func NewHandler(db *sqlx.DB, uploads *upload.Storage, tokens *auth.Tokens) *Handler {
	return &Handler{db: db, uploads: uploads, tokens: tokens}
}

type imageInput struct {
	URL      string `json:"url"`
	Alt      string `json:"alt"`
	Position string `json:"position"`
}

// valueInput is the body of PUT /content/blocks/{key}: "value" for text, lines and richtext, "image" for image.
type valueInput struct {
	Value json.RawMessage `json:"value"`
	Image *imageInput     `json:"image"`

	// uploads checks image URLs; the handler sets it after decoding.
	uploads *upload.Storage
}

type imageResponse struct {
	URL      string `json:"url"`
	Alt      string `json:"alt"`
	Position string `json:"position"`
}

type blockResponse struct {
	Key       string         `json:"key"`
	Page      string         `json:"page"`
	Section   string         `json:"section"`
	Label     string         `json:"label"`
	Type      string         `json:"type"`
	MaxLength int            `json:"maxLength"`
	Value     any            `json:"value"`
	Image     *imageResponse `json:"image"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

// Routes registers GET /content/blocks (public) and PUT /content/blocks/{key} (admin).
func (h *Handler) Routes(r chi.Router) {
	r.Get("/content/blocks", httpx.Handle(h.list))
	r.Group(func(r chi.Router) {
		r.Use(h.tokens.RequireRole(auth.RoleAdmin))
		r.Put("/content/blocks/{key}", httpx.Handle(h.put))
	})
}

func (h *Handler) response(b Block) blockResponse {
	out := blockResponse{
		Key: b.Key, Page: b.Page, Section: b.Section, Label: b.Label, Type: b.Type, UpdatedAt: b.UpdatedAt,
	}
	switch b.Type {
	case TypeText:
		out.MaxLength = effectiveLength(b.MaxLength, defaultTextLength)
		out.Value = b.TextValue
	case TypeRichtext:
		out.MaxLength = effectiveLength(b.MaxLength, defaultRichtextLength)
		out.Value = b.TextValue
	case TypeLines:
		lines := []string{}
		if b.TextValue != "" {
			lines = strings.Split(b.TextValue, "\n")
		}
		out.Value = lines
	case TypeImage:
		out.Image = &imageResponse{URL: h.uploads.Public(b.ImageURL), Alt: b.ImageAlt, Position: b.ImagePosition}
	}
	return out
}

func effectiveLength(column, def int) int {
	if column > 0 {
		return column
	}
	return def
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	blocks, err := NewStore(h.db).List(r.Context(), r.URL.Query().Get("page"))
	if err != nil {
		return err
	}
	out := make([]blockResponse, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, h.response(b))
	}
	httpx.JSON(w, http.StatusOK, out)
	return nil
}

func (h *Handler) put(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	block, err := NewStore(h.db).Get(ctx, chi.URLParam(r, "key"))
	if errors.Is(err, store.ErrNotFound) {
		return httpx.NotFound()
	}
	if err != nil {
		return err
	}
	var in valueInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	in.uploads = h.uploads
	stored, err := validateValue(block, in)
	if err != nil {
		return err
	}

	var (
		saved  Block
		oldURL string
	)
	err = store.InTx(ctx, h.db, func(tx *sqlx.Tx) error {
		var err error
		if block.Type == TypeImage {
			saved, oldURL, err = NewStore(tx).SetImage(ctx, block.Key, stored, in.Image.Alt, in.Image.Position)
		} else {
			saved, err = NewStore(tx).SetText(ctx, block.Key, stored)
		}
		return err
	})
	if errors.Is(err, store.ErrNotFound) {
		return httpx.NotFound()
	}
	if err != nil {
		return err
	}
	if block.Type == TypeImage && oldURL != saved.ImageURL {
		if err := h.uploads.Remove(oldURL); err != nil {
			slog.Error("remove content image", "block", saved.Key, "path", oldURL, "error", err)
		}
	}
	httpx.JSON(w, http.StatusOK, h.response(saved))
	return nil
}

// validateValue checks the request against the block's type and returns what goes into the database: text_value for
// text, lines and richtext, image_url for image (in.Image is trimmed in place and then holds alt and position).
// Every error is an *httpx.Error with the invalid fields.
func validateValue(b Block, in valueInput) (string, error) {
	switch b.Type {
	case TypeText:
		return validateText(b, in)
	case TypeLines:
		return validateLines(in)
	case TypeRichtext:
		return validateRichtext(b, in)
	case TypeImage:
		return validateImage(in)
	}
	return "", fmt.Errorf("unknown block type %q", b.Type)
}

func fieldError(field, message string) *httpx.Error {
	e := httpx.NewError(http.StatusBadRequest, validationMessage)
	e.Fields = map[string]string{field: message}
	return e
}

// decodeString reads a JSON string; absent, null and non-string values are rejected.
func decodeString(raw json.RawMessage) (string, bool) {
	var s *string
	if err := json.Unmarshal(raw, &s); err != nil || s == nil {
		return "", false
	}
	return *s, true
}

func validateText(b Block, in valueInput) (string, error) {
	s, ok := decodeString(in.Value)
	if !ok {
		return "", fieldError("value", wrongType)
	}
	s = strings.TrimSpace(s)
	if strings.ContainsAny(s, "\r\n") {
		return "", fieldError("value", "не должно содержать переносов строк")
	}
	if limit := effectiveLength(b.MaxLength, defaultTextLength); utf8.RuneCountInString(s) > limit {
		return "", fieldError("value", fmt.Sprintf("не более %d символов", limit))
	}
	if err := checkPlaceholders(s); err != nil {
		return "", err
	}
	return s, nil
}

func validateLines(in valueInput) (string, error) {
	var raw *[]string
	if err := json.Unmarshal(in.Value, &raw); err != nil || raw == nil {
		return "", fieldError("value", wrongType)
	}
	lines := make([]string, 0, len(*raw))
	for _, l := range *raw {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if strings.ContainsAny(l, "\r\n") {
			return "", fieldError("value", "строки не должны содержать переносов")
		}
		if utf8.RuneCountInString(l) > maxLineLength {
			return "", fieldError("value", fmt.Sprintf("каждая строка не более %d символов", maxLineLength))
		}
		if err := checkPlaceholders(l); err != nil {
			return "", err
		}
		lines = append(lines, l)
	}
	if len(lines) < 1 || len(lines) > maxLines {
		return "", fieldError("value", fmt.Sprintf("от 1 до %d строк", maxLines))
	}
	return strings.Join(lines, "\n"), nil
}

func validateRichtext(b Block, in valueInput) (string, error) {
	s, ok := decodeString(in.Value)
	if !ok {
		return "", fieldError("value", wrongType)
	}
	clean := richtext.Sanitize(richtext.Short, s)
	if limit := effectiveLength(b.MaxLength, defaultRichtextLength); richtext.TextLength(clean) > limit {
		return "", fieldError("value", fmt.Sprintf("не более %d символов", limit))
	}
	if err := checkPlaceholders(clean); err != nil {
		return "", err
	}
	return clean, nil
}

func validateImage(in valueInput) (string, error) {
	if len(in.Value) > 0 && string(in.Value) != "null" {
		return "", fieldError("value", wrongType)
	}
	if in.Image == nil {
		return "", fieldError("image", wrongType)
	}
	img := in.Image
	img.URL = strings.TrimSpace(img.URL)
	img.Alt = strings.TrimSpace(img.Alt)
	img.Position = strings.TrimSpace(img.Position)

	fields := map[string]string{}
	url, err := in.uploads.Stored(img.URL)
	switch {
	case img.URL == "":
		fields["image.url"] = "поле не должно быть пустым"
	case err != nil:
		fields["image.url"] = "недопустимый адрес фото"
	}
	if utf8.RuneCountInString(img.Alt) > maxAltLength {
		fields["image.alt"] = fmt.Sprintf("не более %d символов", maxAltLength)
	}
	if !validPosition(img.Position) {
		fields["image.position"] = "формат «50% 30%»"
	}
	if len(fields) > 0 {
		e := httpx.NewError(http.StatusBadRequest, validationMessage)
		e.Fields = fields
		return "", e
	}
	return url, nil
}

// checkPlaceholders rejects a {{name}} that is not in Placeholders.
func checkPlaceholders(s string) error {
	for _, m := range placeholderRe.FindAllStringSubmatch(s, -1) {
		if !isPlaceholder(m[1]) {
			return fieldError("value", "неизвестная подстановка {{"+m[1]+"}}")
		}
	}
	return nil
}

func isPlaceholder(name string) bool {
	for _, p := range Placeholders {
		if p == name {
			return true
		}
	}
	return false
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
