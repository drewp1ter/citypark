package catalog

import (
	"errors"
	"net/http"
	"strings"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/salewindow"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/go-chi/chi/v5"
)

// AdminRoutes registers category settings protected by the staff admin role.
func (h *Handler) AdminRoutes(r chi.Router, tokens *auth.Tokens) {
	r.Group(func(r chi.Router) {
		r.Use(tokens.RequireRole(auth.RoleAdmin))
		r.Put("/products/{id}/ordering-block", httpx.Handle(h.putProductBlock))
		r.Put("/categories/{id}/ordering-hours", httpx.Handle(h.putCategoryHours))
	})
}

func (h *Handler) putCategoryHours(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	var in struct {
		Days      *string `json:"days"`
		StartTime *string `json:"startTime"`
		EndTime   *string `json:"endTime"`
	}
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	fields := map[string]string{}
	var start, end *int
	if in.Days == nil || (*in.Days != "" && (len(*in.Days) != 7 || strings.Trim(*in.Days, "01") != "" || *in.Days == "0000000")) {
		fields["days"] = "Выберите дни недели"
	}
	if in.StartTime == nil {
		fields["startTime"] = "Укажите время или пустую строку"
	}
	if in.EndTime == nil {
		fields["endTime"] = "Укажите время или пустую строку"
	}
	if in.StartTime != nil && in.EndTime != nil {
		a, b := strings.TrimSpace(*in.StartTime), strings.TrimSpace(*in.EndTime)
		if a != "" || b != "" {
			x, ok := salewindow.ParseClock(a)
			if !ok {
				fields["startTime"] = "Укажите время в формате ЧЧ:ММ"
			}
			y, okEnd := salewindow.ParseClock(b)
			if !okEnd {
				fields["endTime"] = "Укажите время в формате ЧЧ:ММ"
			}
			if ok && okEnd && x >= y {
				fields["endTime"] = "Время окончания должно быть позже начала"
			}
			start, end = &x, &y
		}
	}
	if len(fields) > 0 {
		e := httpx.NewError(http.StatusBadRequest, "Ошибка валидации, пожалуйста проверьте введенные вами данные")
		e.Fields = fields
		return e
	}
	err = h.store.SetCategoryHours(r.Context(), id, start, end, *in.Days)
	if errors.Is(store.NotFound(err), store.ErrNotFound) {
		return httpx.NotFound()
	}
	if err != nil {
		return err
	}
	return h.getCategory(w, r)
}
