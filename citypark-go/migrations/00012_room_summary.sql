-- +goose Up
ALTER TABLE rooms ADD COLUMN summary text NOT NULL DEFAULT '';
UPDATE rooms SET summary = 'Гостиная и спальня, расширенная комплектация, балкон над кронами парка.', updated_at = now()
WHERE id = 'vip';

-- +goose Down
ALTER TABLE rooms DROP COLUMN summary;
