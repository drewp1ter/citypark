-- +goose Up
ALTER TABLE orders
    ADD COLUMN iiko_status text NOT NULL DEFAULT '' CHECK (iiko_status IN ('', 'awaiting_approval', 'queued', 'sending', 'created', 'failed', 'review')),
    ADD COLUMN iiko_payload bytea,
    ADD COLUMN iiko_send_deadline timestamptz;
ALTER TABLE order_items
    ADD COLUMN iiko_comment text NOT NULL DEFAULT '',
    ADD COLUMN excluded_from_iiko boolean NOT NULL DEFAULT false;
CREATE INDEX orders_iiko_queue_idx ON orders (id) WHERE iiko_status IN ('queued', 'sending');

-- +goose Down
DROP INDEX orders_iiko_queue_idx;
ALTER TABLE order_items DROP COLUMN excluded_from_iiko, DROP COLUMN iiko_comment;
ALTER TABLE orders DROP COLUMN iiko_send_deadline, DROP COLUMN iiko_payload, DROP COLUMN iiko_status;
