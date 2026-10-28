-- +goose Up
CREATE TABLE working_hours (
    id         smallint PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    start_time text NOT NULL CHECK (start_time ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
    end_time   text NOT NULL CHECK (end_time ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (start_time < end_time)
);
INSERT INTO working_hours (start_time, end_time) VALUES ('09:00', '22:00');

-- +goose Down
DROP TABLE working_hours;
