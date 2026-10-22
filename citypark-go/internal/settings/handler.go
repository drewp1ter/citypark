package settings

import (
	"errors"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/validate"
)

const (
	validationMessage = "Ошибка валидации, пожалуйста проверьте введенные вами данные"
	badNumber         = "номер: только цифры и «+»"
	bothFields        = "заполните оба поля"
	badURL            = "ссылка должна начинаться с https://"
)

var numberRe = regexp.MustCompile(`^\+?\d{10,15}$`)

type Handler struct {
	store  *Store
	tokens *auth.Tokens
}

func NewHandler(s *Store, tokens *auth.Tokens) *Handler {
	return &Handler{store: s, tokens: tokens}
}

// response is the Settings JSON plus updatedAt.
type response struct {
	Settings
	UpdatedAt time.Time `json:"updatedAt"`
}

// Routes registers GET /settings (public) and PUT /settings (admin).
func (h *Handler) Routes(r chi.Router) {
	r.Get("/settings", httpx.Handle(h.get))
	r.Group(func(r chi.Router) {
		r.Use(h.tokens.RequireRole(auth.RoleAdmin))
		r.Put("/settings", httpx.Handle(h.put))
	})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) error {
	st, at, err := h.store.Get(r.Context())
	if errors.Is(err, store.ErrNotFound) {
		return httpx.NotFound()
	}
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, response{Settings: st, UpdatedAt: at})
	return nil
}

func (h *Handler) put(w http.ResponseWriter, r *http.Request) error {
	var in Settings
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	trimStrings(reflect.ValueOf(&in).Elem())
	if in.Social == nil {
		in.Social = []Social{}
	}
	if err := check(in); err != nil {
		return err
	}
	at, err := h.store.Put(r.Context(), in)
	if errors.Is(err, store.ErrNotFound) {
		return httpx.NotFound()
	}
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, response{Settings: in, UpdatedAt: at})
	return nil
}

// trimStrings trims every string reachable from v, including structs inside slices.
func trimStrings(v reflect.Value) {
	switch v.Kind() {
	case reflect.String:
		v.SetString(strings.TrimSpace(v.String()))
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			trimStrings(v.Field(i))
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			trimStrings(v.Index(i))
		}
	}
}

// check runs the struct tags and then the rules the tags cannot express. Both kinds of error are reported together.
func check(s Settings) error {
	extra := map[string]string{}
	phones := map[string]Phone{
		"restaurant": s.Phones.Restaurant, "hotel": s.Phones.Hotel, "free": s.Phones.Free, "landline": s.Phones.Landline,
	}
	for name, p := range phones {
		prefix := "phones." + name + "."
		if p.Number != "" && !numberRe.MatchString(p.Number) {
			extra[prefix+"number"] = badNumber
		}
		if p.AltNumber != "" && !numberRe.MatchString(p.AltNumber) {
			extra[prefix+"altNumber"] = badNumber
		}
		if (p.AltDisplay == "") != (p.AltNumber == "") {
			if p.AltDisplay == "" {
				extra[prefix+"altDisplay"] = bothFields
			} else {
				extra[prefix+"altNumber"] = bothFields
			}
		}
	}
	for i, soc := range s.Social {
		if soc.URL != "" && !strings.HasPrefix(soc.URL, "https://") {
			extra["social["+strconv.Itoa(i)+"].url"] = badURL
		}
	}

	err := validate.Struct(s)
	var he *httpx.Error
	if errors.As(err, &he) {
		for k, msg := range extra {
			if _, ok := he.Fields[k]; !ok {
				he.Fields[k] = msg
			}
		}
		return he
	}
	if err != nil {
		return err
	}
	if len(extra) > 0 {
		e := httpx.NewError(http.StatusBadRequest, validationMessage)
		e.Fields = extra
		return e
	}
	return nil
}
