package catalog

import (
	"errors"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/store"
	"net/http"
)

func (h *Handler) putProductBlock(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	var in struct {
		IsOrderBlocked *bool `json:"isOrderBlocked"`
	}
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	if in.IsOrderBlocked == nil {
		e := httpx.BadRequest("Ошибка валидации, пожалуйста проверьте введенные вами данные")
		e.Fields = map[string]string{"isOrderBlocked": "Укажите, заблокирован ли заказ блюда"}
		return e
	}
	if err := h.store.SetProductOrderBlocked(r.Context(), id, *in.IsOrderBlocked); err != nil {
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
