-- +goose Up
ALTER TABLE product_images DROP CONSTRAINT product_images_url_key;
ALTER TABLE product_images ADD CONSTRAINT product_images_product_id_url_key UNIQUE (product_id, url);

-- +goose Down
-- Refuse rollback while shared URLs exist rather than discard image associations.
ALTER TABLE product_images ADD CONSTRAINT product_images_url_key UNIQUE (url);
ALTER TABLE product_images DROP CONSTRAINT product_images_product_id_url_key;
