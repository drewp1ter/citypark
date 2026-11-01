-- +goose Up
CREATE TABLE delivery_settings (
    id smallint PRIMARY KEY CHECK (id = 1),
    cooking text NOT NULL CHECK (char_length(btrim(cooking)) BETWEEN 1 AND 100),
    outskirts_label text NOT NULL CHECK (char_length(btrim(outskirts_label)) BETWEEN 1 AND 60),
    outskirts_min_kopecks bigint NOT NULL CHECK (
        outskirts_min_kopecks BETWEEN 0 AND 10000000 AND outskirts_min_kopecks % 100 = 0),
    outskirts_max_kopecks bigint NOT NULL CHECK (
        outskirts_max_kopecks BETWEEN 0 AND 10000000 AND outskirts_max_kopecks % 100 = 0),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (outskirts_min_kopecks <= outskirts_max_kopecks)
);
INSERT INTO delivery_settings (id, cooking, outskirts_label, outskirts_min_kopecks, outskirts_max_kopecks)
VALUES (1, '~30 мин', 'СОТ и ДНТ', 20000, 35000);

-- +goose Down
DROP TABLE delivery_settings;
