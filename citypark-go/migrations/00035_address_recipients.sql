-- +goose Up
ALTER TABLE user_addresses ADD COLUMN recipient_name text NOT NULL DEFAULT '', ADD COLUMN recipient_phone text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE user_addresses DROP COLUMN recipient_phone, DROP COLUMN recipient_name;
