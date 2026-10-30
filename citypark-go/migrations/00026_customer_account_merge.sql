-- +goose Up
ALTER TABLE users
    ADD COLUMN merged_into_user_id bigint REFERENCES users(id),
    ADD CONSTRAINT users_merge_not_self CHECK (merged_into_user_id IS NULL OR merged_into_user_id <> id);

-- +goose Down
-- Dropping the marker cannot undo transferred identities, orders, or addresses.
ALTER TABLE users DROP CONSTRAINT users_merge_not_self;
ALTER TABLE users DROP COLUMN merged_into_user_id;
