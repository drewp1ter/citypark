-- +goose Up
ALTER TABLE delivery_prices ADD COLUMN region_code text NOT NULL DEFAULT 'RU-KDA'
    CHECK (region_code IN ('RU-KDA', 'RU-AD'));

-- +goose Down
ALTER TABLE delivery_prices DROP COLUMN region_code;
