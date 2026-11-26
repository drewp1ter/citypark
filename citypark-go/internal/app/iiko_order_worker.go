package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/drewp1ter/citypark-go/internal/iiko"
	"github.com/drewp1ter/citypark-go/internal/order"
)

// Run processes approved orders and flags interrupted sends for staff review.
func (n *IikoOrders) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for ctx.Err() == nil {
		worked, err := n.processOrder(ctx)
		if err != nil && ctx.Err() == nil {
			n.log.Error("process iiko order", "error", err)
		}
		if worked && err == nil {
			continue
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

func (n *IikoOrders) processOrder(ctx context.Context) (bool, error) {
	var expired int64
	err := n.db.GetContext(ctx, &expired, `UPDATE orders SET iiko_status = 'review', iiko_send_deadline = NULL,
		iiko_payload = NULL, updated_at = now() WHERE id = (
			SELECT id FROM orders WHERE iiko_status = 'sending' AND iiko_send_deadline <= now()
			ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED
		) RETURNING id`)
	if err == nil {
		n.review(ctx, expired)
		return true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	if !n.configured() {
		return false, nil
	}

	var id int64
	var payload []byte
	err = n.db.QueryRowxContext(ctx, `UPDATE orders SET iiko_status = 'sending',
		iiko_send_deadline = now() + interval '15 minutes', updated_at = now() WHERE id = (
			SELECT id FROM orders WHERE iiko_status = 'queued' ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED
		) RETURNING id, iiko_payload`).Scan(&id, &payload)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	sendCtx, cancel := context.WithTimeout(ctx, 13*time.Minute)
	defer cancel()
	var delivery iiko.DeliveryOrder
	var iikoID string
	state := "failed"
	submitErr := json.Unmarshal(payload, &delivery)
	if submitErr == nil {
		state, iikoID, submitErr = n.submit(sendCtx, id, delivery)
	}
	saveCtx, finish := context.WithTimeout(context.WithoutCancel(ctx), sendTimeout)
	defer finish()
	res, err := n.db.ExecContext(saveCtx, `UPDATE orders SET iiko_status = $2,
		iiko_order_id = $3, iiko_send_deadline = NULL, iiko_payload = NULL, updated_at = now()
		WHERE id = $1 AND iiko_status = 'sending'`, id, state, iikoID)
	if err != nil {
		return true, err
	}
	changed, err := res.RowsAffected()
	if err != nil || changed == 0 {
		return true, err
	}
	switch state {
	case "review":
		n.review(saveCtx, id)
	case "failed":
		n.fail(saveCtx, order.Order{ID: id}, submitErr)
	case "created":
		n.log.Info("order created in iiko", "order_id", id, "iiko_id", iikoID)
	}
	return true, nil
}

func (n *IikoOrders) review(ctx context.Context, id int64) {
	n.alert(ctx, order.Order{ID: id}, fmt.Sprintf("Заказ №%d отправлен в iiko, но создание не подтверждено. Проверьте его в iikoFront.", id))
}
