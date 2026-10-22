-- +goose Up
CREATE TABLE content_blocks (
    key            text PRIMARY KEY CHECK (key ~ '^[a-z0-9-]+(\.[a-z0-9-]+)+$'),
    page           text NOT NULL,
    section        text NOT NULL DEFAULT '',
    sort           integer NOT NULL DEFAULT 0,
    label          text NOT NULL,
    type           text NOT NULL CHECK (type IN ('text', 'lines', 'richtext', 'image')),
    max_length     integer NOT NULL DEFAULT 0,
    text_value     text NOT NULL DEFAULT '',
    image_url      text NOT NULL DEFAULT '',
    image_alt      text NOT NULL DEFAULT '',
    image_position text NOT NULL DEFAULT '',
    updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX content_blocks_page_idx ON content_blocks (page, sort);

-- +goose Down
DROP TABLE content_blocks;
