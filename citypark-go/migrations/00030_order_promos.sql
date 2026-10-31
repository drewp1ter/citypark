-- +goose Up
ALTER TABLE orders
 ADD COLUMN promo_code_id bigint REFERENCES promo_codes(id),
 ADD COLUMN promo_code text,
 ADD COLUMN promo_percentage integer CHECK (promo_percentage BETWEEN 1 AND 99),
 ADD COLUMN subtotal_kopecks bigint NOT NULL DEFAULT 0 CHECK (subtotal_kopecks >= 0),
 ADD COLUMN discount_kopecks bigint NOT NULL DEFAULT 0 CHECK (discount_kopecks >= 0);
ALTER TABLE order_items
 ADD COLUMN discount_kopecks bigint NOT NULL DEFAULT 0 CHECK (discount_kopecks >= 0),
 ADD COLUMN promo_eligible boolean NOT NULL DEFAULT false;
UPDATE orders o SET subtotal_kopecks = COALESCE((SELECT sum(oi.offered_price_kopecks * oi.amount) FROM order_items oi WHERE oi.order_id=o.id),0), updated_at=now();
ALTER TABLE orders ADD CONSTRAINT order_discount_limit CHECK (discount_kopecks <= subtotal_kopecks);
-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
 IF EXISTS (SELECT 1 FROM orders WHERE promo_code IS NOT NULL OR discount_kopecks > 0)
 OR EXISTS (SELECT 1 FROM order_items WHERE discount_kopecks > 0 OR promo_eligible) THEN
 RAISE EXCEPTION 'cannot discard order promo snapshots; archive and resolve them before rollback';
 END IF;
END $$;
-- +goose StatementEnd
ALTER TABLE order_items DROP COLUMN promo_eligible, DROP COLUMN discount_kopecks;
ALTER TABLE orders DROP CONSTRAINT order_discount_limit, DROP COLUMN discount_kopecks, DROP COLUMN subtotal_kopecks,
 DROP COLUMN promo_percentage, DROP COLUMN promo_code, DROP COLUMN promo_code_id;
