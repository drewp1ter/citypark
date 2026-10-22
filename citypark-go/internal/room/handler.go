package room

import (
	"errors"
	"log/slog"
	"maps"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/richtext"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/upload"
	"github.com/drewp1ter/citypark-go/internal/validate"
)

const (
	maxDescriptionLength = 1000
	maxPercent           = 100
	// validationMessage is the top-level message of validate.Struct, used here for the checks it cannot express.
	validationMessage = "Ошибка валидации, пожалуйста проверьте введенные вами данные"
)

var positionRe = regexp.MustCompile(`^(\d{1,3})% (\d{1,3})%$`)

// Handler serves the room types.
type Handler struct {
	store   *Store
	uploads *upload.Storage
	tokens  *auth.Tokens
}

func NewHandler(s *Store, uploads *upload.Storage, tokens *auth.Tokens) *Handler {
	return &Handler{store: s, uploads: uploads, tokens: tokens}
}

type photoInput struct {
	URL      string `json:"url" validate:"required"`
	Alt      string `json:"alt" validate:"max=200"`
	Position string `json:"position" validate:"max=100"`
}

// updateRequest is the body of PUT /rooms/{id}. The price is a float so that a third decimal can be detected
// (money.Kopecks would round it away while decoding).
type updateRequest struct {
	Name          string     `json:"name" validate:"required,max=100"`
	HotelName     string     `json:"hotelName" validate:"max=100"`
	Numbers       string     `json:"numbers" validate:"required,max=100"`
	Area          string     `json:"area" validate:"required,max=100"`
	AreaDetailed  string     `json:"areaDetailed" validate:"max=100"`
	Capacity      string     `json:"capacity" validate:"max=100"`
	Description   string     `json:"description"`
	PricePerNight float64    `json:"pricePerNight" validate:"required,gt=0,max=1000000"`
	Photo         photoInput `json:"photo"`
	HotelNote     string     `json:"hotelNote" validate:"max=100"`
}

type photoResponse struct {
	URL      string `json:"url"`
	Alt      string `json:"alt"`
	Position string `json:"position"`
}

type response struct {
	ID            string        `json:"id"`
	Kind          string        `json:"kind"`
	Balcony       bool          `json:"balcony"`
	Name          string        `json:"name"`
	HotelName     string        `json:"hotelName"`
	Numbers       string        `json:"numbers"`
	Area          string        `json:"area"`
	AreaDetailed  string        `json:"areaDetailed"`
	Capacity      string        `json:"capacity"`
	Description   string        `json:"description"`
	PricePerNight money.Kopecks `json:"pricePerNight"`
	Photo         photoResponse `json:"photo"`
	HotelNote     string        `json:"hotelNote"`
	UpdatedAt     time.Time     `json:"updatedAt"`
}

// Routes registers GET /rooms (public) and PUT /rooms/{id} (admin).
func (h *Handler) Routes(r chi.Router) {
	r.Get("/rooms", httpx.Handle(h.list))
	r.Group(func(r chi.Router) {
		r.Use(h.tokens.RequireRole(auth.RoleAdmin))
		r.Put("/rooms/{id}", httpx.Handle(h.update))
	})
}

func (h *Handler) response(r Room) response {
	return response{
		ID: r.ID, Kind: r.Kind, Balcony: r.Balcony, Name: r.Name, HotelName: r.HotelName,
		Numbers: r.Numbers, Area: r.Area, AreaDetailed: r.AreaDetailed, Capacity: r.Capacity,
		Description: r.Description, PricePerNight: r.Price,
		Photo:     photoResponse{URL: h.uploads.Public(r.PhotoURL), Alt: r.PhotoAlt, Position: r.PhotoPosition},
		HotelNote: r.HotelNote, UpdatedAt: r.UpdatedAt,
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	rooms, err := h.store.List(r.Context())
	if err != nil {
		return err
	}
	out := make([]response, 0, len(rooms))
	for _, room := range rooms {
		out = append(out, h.response(room))
	}
	httpx.JSON(w, http.StatusOK, out)
	return nil
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) error {
	var req updateRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	trim(&req)
	room, err := h.check(req)
	if err != nil {
		return err
	}
	room.ID = chi.URLParam(r, "id")

	saved, oldPhoto, err := h.store.Update(r.Context(), room)
	if errors.Is(err, store.ErrNotFound) {
		return httpx.NotFound()
	}
	if err != nil {
		return err
	}
	if oldPhoto != saved.PhotoURL {
		if err := h.uploads.Remove(oldPhoto); err != nil {
			slog.Error("remove replaced room photo", "room", saved.ID, "path", oldPhoto, "error", err)
		}
	}
	httpx.JSON(w, http.StatusOK, h.response(saved))
	return nil
}

// trim runs before validation so that a whitespace-only name fails "required".
func trim(req *updateRequest) {
	for _, s := range []*string{
		&req.Name, &req.HotelName, &req.Numbers, &req.Area, &req.AreaDetailed, &req.Capacity, &req.HotelNote,
		&req.Photo.URL, &req.Photo.Alt, &req.Photo.Position,
	} {
		*s = strings.TrimSpace(*s)
	}
}

// check validates req and returns the Room to store (without ID). All invalid fields are reported together.
func (h *Handler) check(req updateRequest) (Room, error) {
	fields := map[string]string{}
	var top *httpx.Error
	if err := validate.Struct(req); err != nil {
		if !errors.As(err, &top) {
			return Room{}, err
		}
		maps.Copy(fields, top.Fields)
	}

	if _, bad := fields["pricePerNight"]; !bad && !hasKopecksPrecision(req.PricePerNight) {
		fields["pricePerNight"] = "не более 2 знаков после запятой"
	}

	photoURL := ""
	if _, bad := fields["photo.url"]; !bad {
		stored, err := h.uploads.Stored(req.Photo.URL)
		if err != nil {
			fields["photo.url"] = "недопустимый адрес фото"
		}
		photoURL = stored
	}
	if _, bad := fields["photo.position"]; !bad && !validPosition(req.Photo.Position) {
		fields["photo.position"] = "формат «50% 30%»"
	}

	description := richtext.Sanitize(richtext.Short, req.Description)
	if richtext.TextLength(description) > maxDescriptionLength {
		fields["description"] = "не более 1000 символов"
	}

	if len(fields) > 0 {
		if top == nil {
			top = httpx.NewError(http.StatusBadRequest, validationMessage)
		}
		top.Fields = fields
		return Room{}, top
	}
	return Room{
		Name: req.Name, HotelName: req.HotelName, Numbers: req.Numbers, Area: req.Area,
		AreaDetailed: req.AreaDetailed, Capacity: req.Capacity, Description: description,
		Price:    money.FromRubles(req.PricePerNight),
		PhotoURL: photoURL, PhotoAlt: req.Photo.Alt, PhotoPosition: req.Photo.Position,
		HotelNote: req.HotelNote,
	}, nil
}

// hasKopecksPrecision reports whether rubles has at most two decimals.
func hasKopecksPrecision(rubles float64) bool {
	return math.Abs(rubles*100-float64(money.FromRubles(rubles))) < 1e-6
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
