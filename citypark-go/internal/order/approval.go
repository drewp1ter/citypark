package order

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/store"
)

type approvalInput struct {
	ExcludedProductIDs []int64       `json:"excludedProductIds"`
	ItemComments       []itemComment `json:"itemComments"`
}

type itemComment struct {
	ProductID int64  `json:"productId"`
	Comment   string `json:"comment"`
}

func applyApproval(o *Order, in approvalInput) error {
	items := make(map[int64]*Item, len(o.Items))
	for i := range o.Items {
		item := &o.Items[i]
		items[item.ProductID] = item
	}
	for _, id := range in.ExcludedProductIDs {
		item := items[id]
		if item == nil || item.ExcludedFromIiko {
			return httpx.BadRequest("Некорректный список исключённых блюд.")
		}
		item.ExcludedFromIiko = true
	}
	if len(in.ExcludedProductIDs) == len(items) {
		return httpx.BadRequest("Для отправки выберите хотя бы одно блюдо.")
	}
	seen := make(map[int64]bool, len(in.ItemComments))
	for _, entry := range in.ItemComments {
		item := items[entry.ProductID]
		if item == nil || item.ExcludedFromIiko || seen[entry.ProductID] {
			return httpx.BadRequest("Некорректный список комментариев к блюдам.")
		}
		seen[entry.ProductID] = true
		comment := strings.TrimSpace(entry.Comment)
		if utf8.RuneCountInString(comment) > 255 || strings.ContainsRune(comment, 0) {
			err := httpx.BadRequest(validationMessage)
			err.Fields = map[string]string{fmt.Sprintf("itemComments.%d", entry.ProductID): "Введите до 255 символов без управляющего символа NUL."}
			return err
		}
		item.IikoComment = comment
	}
	return nil
}

func (h *Handler) approve(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	var in approvalInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	ctx := r.Context()
	var o Order
	err = store.InTx(ctx, h.svc.db, func(tx *sqlx.Tx) error {
		if _, err := lockOrder(ctx, tx, id); errors.Is(err, sql.ErrNoRows) {
			return httpx.NotFound()
		} else if err != nil {
			return err
		}
		o, err = NewStore(tx).ByID(ctx, id)
		if err != nil {
			return err
		}
		if o.Status != StatusPaid || o.IikoStatus == "" {
			return httpx.NewError(http.StatusConflict, "Заказ недоступен для отправки в iiko.")
		}
		if o.IikoStatus != "awaiting_approval" {
			return nil
		}
		if err := applyApproval(&o, in); err != nil {
			return err
		}
		if h.opts.PrepareIiko == nil {
			return httpx.NewError(http.StatusServiceUnavailable, "Отправка в iiko недоступна.")
		}
		payload, err := h.opts.PrepareIiko(o)
		if err != nil {
			return err
		}
		for _, item := range o.Items {
			if !item.ExcludedFromIiko && item.IikoComment == "" {
				continue
			}
			if _, err := tx.ExecContext(ctx, `UPDATE order_items SET excluded_from_iiko = $3, iiko_comment = $4
				WHERE order_id = $1 AND product_id = $2`, id, item.ProductID, item.ExcludedFromIiko, item.IikoComment); err != nil {
				return err
			}
		}
		return tx.QueryRowxContext(ctx, `UPDATE orders SET iiko_status = 'queued', iiko_payload = $2, updated_at = now()
			WHERE id = $1 RETURNING iiko_status, updated_at`, id, payload).Scan(&o.IikoStatus, &o.UpdatedAt)
	})
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, o)
	return nil
}

func lockOrder(ctx context.Context, tx *sqlx.Tx, id int64) (string, error) {
	var state string
	err := tx.GetContext(ctx, &state, "SELECT iiko_status FROM orders WHERE id = $1 FOR UPDATE", id)
	return state, err
}

// editLegacyOrder serializes edits with payment and protects the paid review snapshot.
func (h *Handler) editLegacyOrder(ctx context.Context, id int64, edit func(*Store) error) error {
	return store.InTx(ctx, h.svc.db, func(tx *sqlx.Tx) error {
		state, err := lockOrder(ctx, tx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if state != "" {
			return httpx.NewError(http.StatusConflict, "Нельзя изменять оплаченный заказ для iiko.")
		}
		return edit(NewStore(tx))
	})
}
