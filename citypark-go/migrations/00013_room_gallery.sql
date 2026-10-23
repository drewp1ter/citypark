-- +goose Up
ALTER TABLE rooms ADD COLUMN gallery jsonb NOT NULL DEFAULT '[]';
UPDATE rooms SET updated_at = now(), gallery = '[
  {"url": "/images/room-vip-living.jpg", "alt": "VIP-номер: гостиная"},
  {"url": "/images/room-vip-bathroom.jpg", "alt": "VIP-номер: санузел"},
  {"url": "/images/room-vip-balcony.jpg", "alt": "VIP-номер: балкон"}
]'
WHERE id = 'vip';

-- +goose Down
ALTER TABLE rooms DROP COLUMN gallery;
