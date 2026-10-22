-- +goose Up
CREATE TABLE hotel_rooms (
    id                  smallint PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    vip_free            integer NOT NULL CHECK (vip_free BETWEEN 0 AND 1),
    single_balcony_free integer NOT NULL CHECK (single_balcony_free BETWEEN 0 AND 1),
    single_free         integer NOT NULL CHECK (single_free BETWEEN 0 AND 2),
    double_twin_free    integer NOT NULL CHECK (double_twin_free BETWEEN 0 AND 4),
    double_euro_free    integer NOT NULL CHECK (double_euro_free BETWEEN 0 AND 4),
    updated_at          timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE hotel_rooms;
