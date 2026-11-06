package catalog

import (
	"errors"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/richtext"
	"github.com/drewp1ter/citypark-go/internal/store"
	"net/http"
)

func (h *Handler) putProductFullDescription(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	var in struct {
		FullDescription *string `json:"fullDescription"`
	}
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	if in.FullDescription == nil {
		e := httpx.BadRequest("Ошибка валидации, пожалуйста проверьте введенные вами данные")
		e.Fields = map[string]string{"fullDescription": "Укажите полное описание"}
		return e
	}
	description := richtext.Sanitize(richtext.Short, *in.FullDescription)
	if richtext.TextLength(description) > 10000 {
		e := httpx.BadRequest("Ошибка валидации, пожалуйста проверьте введенные вами данные")
		e.Fields = map[string]string{"fullDescription": "Не более 10000 символов"}
		return e
	}
	if err := h.store.SetProductFullDescription(r.Context(), id, description); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return httpx.NotFound()
		}
		return err
	}
	list, err := h.store.ProductsByIDs(r.Context(), []int64{id})
	if err != nil {
		return err
	}
	if len(list) == 0 {
		return httpx.NotFound()
	}
	if err := h.store.attachProductImages(r.Context(), list); err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, list[0].View(h.clock.Now()))
	return nil
}
