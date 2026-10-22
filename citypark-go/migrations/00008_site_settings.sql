-- +goose Up
CREATE TABLE site_settings (
    id         smallint PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    data       jsonb NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO site_settings (data) VALUES ($json${
  "phones": {
    "restaurant": {"label": "Ресторан и доставка", "shortLabel": "Ресторан", "display": "8 918 311-97-91", "number": "+79183119791", "altDisplay": "8 (86155) 3-30-03", "altNumber": "+78615533003"},
    "hotel": {"label": "Отель", "shortLabel": "Отель", "display": "8 918 311-97-10", "number": "+79183119710", "altDisplay": "8 (86155) 3-30-02", "altNumber": "+78615533002"},
    "free": {"label": "Бесплатно по России", "shortLabel": "По России", "display": "8 800 100-24-41", "number": "88001002441", "altDisplay": "", "altNumber": ""},
    "landline": {"label": "Городской", "shortLabel": "Городской", "display": "(86155) 2-25-38", "number": "+78615522538", "altDisplay": "", "altNumber": ""}
  },
  "address": {"postcode": "352630", "city": "Белореченск", "street": "ул. Гоголя, 61"},
  "email": "belcitypark@mail.ru",
  "company": {"name": "ООО «Олимп»", "inn": "2309117400", "ogrn": "1092309002378"},
  "social": [
    {"name": "ВКонтакте", "label": "VK", "url": "https://vk.com/cityparkvip", "display": "vk.com/cityparkvip"},
    {"name": "Одноклассники", "label": "OK", "url": "https://ok.ru/cityparkvip", "display": "ok.ru/cityparkvip"}
  ]
}$json$::jsonb);

-- +goose Down
DROP TABLE site_settings;
