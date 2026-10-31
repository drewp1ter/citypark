package promo

import (
	"errors"
	"net/http"
	"strings"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/validate"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

type Handler struct {
	db     *sqlx.DB
	tokens *auth.Tokens
}

func NewHandler(db *sqlx.DB, tokens *auth.Tokens) *Handler { return &Handler{db, tokens} }
func (h *Handler) Routes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(h.tokens.RequireRole(auth.RoleAdmin))
		r.Get("/promo-codes", httpx.Handle(h.list))
		r.Get("/promo-codes/{id}", httpx.Handle(h.get))
		r.Post("/promo-codes", httpx.Handle(h.write))
		r.Put("/promo-codes/{id}", httpx.Handle(h.write))
	})
}
func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	list, err := NewStore(h.db).List(r.Context())
	if err != nil {
		return err
	}
	httpx.JSON(w, 200, list)
	return nil
}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	c, err := NewStore(h.db).ByID(r.Context(), id)
	if err != nil {
		return campaignError(err)
	}
	httpx.JSON(w, 200, c)
	return nil
}
func (h *Handler) write(w http.ResponseWriter, r *http.Request) error {
	var id int64
	var err error
	if r.Method == http.MethodPut {
		id, err = httpx.PathID(r, "id")
		if err != nil {
			return err
		}
	}
	var in Input
	if err = httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	if err = validate.Struct(in); err != nil {
		return err
	}
	var c Campaign
	err = store.InTx(r.Context(), h.db, func(tx *sqlx.Tx) error {
		if r.Method == http.MethodPost {
			c, err = NewStore(tx).Create(r.Context(), in)
		} else {
			c, err = NewStore(tx).Update(r.Context(), id, in)
		}
		return err
	})
	if err != nil {
		return campaignError(err)
	}
	status := http.StatusOK
	if r.Method == http.MethodPost {
		status = http.StatusCreated
	}
	httpx.JSON(w, status, c)
	return nil
}
func campaignError(err error) error {
	if errors.Is(err, store.ErrNotFound) {
		return httpx.NewError(404, "Промокод не найден.")
	}
	if store.IsUniqueViolation(err) {
		return httpx.NewError(409, "Такой промокод уже существует.")
	}
	var v *ValidationError
	if errors.As(err, &v) {
		if v.Field == "code" && v.Message == "immutable" {
			return httpx.NewError(409, "Код нельзя изменить после создания.")
		}
		messages := map[string]string{
			"code":                    "Промокод должен содержать от 3 до 32 латинских букв, цифр, дефисов или подчёркиваний.",
			"percentage":              "Укажите целый процент от 1 до 99.",
			"minimumEligibleSubtotal": "Укажите положительную сумму.",
			"maximumDiscount":         "Укажите положительную сумму.",
			"endsAt":                  "Дата окончания должна быть позже даты начала.",
			"target":                  "Выберите все блюда или укажите блюда и категории.",
		}
		return campaignFieldError(v.Field, messages[v.Field])
	}
	if store.IsFKViolation(err) {
		field := "productIds"
		if strings.Contains(err.Error(), "category_id") {
			field = "categoryIds"
		}
		return campaignFieldError(field, "Выбранное блюдо или категория не найдены.")
	}
	return err
}
func campaignFieldError(field, message string) error {
	e := httpx.NewError(400, "Ошибка валидации, пожалуйста проверьте введенные вами данные")
	e.Fields = map[string]string{field: message}
	return e
}
