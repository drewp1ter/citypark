-- +goose Up
CREATE TABLE homepage_menu (
    id smallint PRIMARY KEY CHECK (id = 1),
    data jsonb NOT NULL CHECK (jsonb_typeof(data->'categories') = 'array'),
    updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO homepage_menu (id, data) VALUES (1, '{"categories":[]}');

-- +goose Down
DROP TABLE homepage_menu;
