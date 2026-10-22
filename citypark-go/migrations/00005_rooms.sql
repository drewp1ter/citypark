-- +goose Up
CREATE TABLE rooms (
    id             text PRIMARY KEY CHECK (id IN ('vip', 'single-balcony', 'single', 'double-twin', 'double-euro')),
    sort           integer NOT NULL,
    kind           text NOT NULL CHECK (kind IN ('single', 'double', 'vip')),
    balcony        boolean NOT NULL,
    name           text NOT NULL,
    hotel_name     text NOT NULL DEFAULT '',
    numbers        text NOT NULL,
    area           text NOT NULL,
    area_detailed  text NOT NULL DEFAULT '',
    capacity       text NOT NULL DEFAULT '',
    description    text NOT NULL DEFAULT '',
    price_kopecks  bigint NOT NULL CHECK (price_kopecks > 0),
    photo_url      text NOT NULL DEFAULT '',
    photo_alt      text NOT NULL DEFAULT '',
    photo_position text NOT NULL DEFAULT '',
    hotel_note     text NOT NULL DEFAULT '',
    updated_at     timestamptz NOT NULL DEFAULT now()
);

INSERT INTO rooms (id, sort, kind, balcony, name, hotel_name, numbers, area, area_detailed, capacity, description,
                   price_kopecks, photo_url, photo_alt, photo_position, hotel_note)
VALUES
    ('vip', 1, 'vip', true, 'VIP-номер', 'VIP, 2 комнаты', '№ 301', '45 + 7 м²', '45 м² + балкон 7 м²', 'до 4 гостей',
     '<p>Гостиная и спальня, расширенная комплектация, балкон над кронами парка.</p>',
     800000, '/images/room-vip.jpg', 'VIP-номер', '', '№ 301 · до 4 гостей'),
    ('single-balcony', 2, 'single', true, 'Одноместный с балконом', '', '№ 201', '23 + 7 м²', '', '', '',
     450000, '/images/room-single.jpg', 'Одноместный номер с балконом', '', '№ 201 · двуспальная кровать'),
    ('single', 3, 'single', false, 'Одноместный', '', '№ 204, 304', '19 м²', '', '', '',
     400000, '/images/room-single.jpg', 'Одноместный номер', 'right center', '№ 204, 304 · двуспальная'),
    ('double-twin', 4, 'double', false, 'Двухместный, 2 кровати', '', '№ 203, 206, 302, 303', '23–25 м²', '', '', '',
     450000, '/images/room-double.jpg', 'Двухместный номер с двумя кроватями', '', '№ 203, 206, 302, 303 · можно сдвинуть'),
    ('double-euro', 5, 'double', false, 'Двухместный «евро»', '', '№ 202, 205, 305, 306', '23–25 м²', '', '', '',
     450000, '/images/room-vip.jpg', 'Двухместный номер с кроватью евро', 'center 70%', '№ 202, 205, 305, 306');

-- +goose Down
DROP TABLE rooms;
