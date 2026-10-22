# Task 54: Content block rows for the home and hotel pages

## Goal
A migration creates every content block the site uses (home page, hotel page, rooms page facts), with the label,
section, type and current value. After it, `GET /content/blocks` returns exactly the blocks the site's static data file
has, with the same values.

## Depends on
50, and citypark-modern tasks 15 and 16 (they create the list of blocks).

## Read first
`docs/tasks/CONVENTIONS.md`, `migrations/00007_content_blocks.sql`, `internal/content/store.go`,
`../citypark-modern/src/entities/content/model/blocks.data.ts` (the source of truth for this migration).

## Background
citypark-modern tasks 15 and 16 moved the editable copy of the site into `blocks.data.ts`: one entry per block key with its
type, Russian label and section, and the current text. Block keys are created only by migrations (ADR 0001), so this
task copies that file into SQL. Later new keys need a new migration and a new entry in that file.

## Files to create
- `migrations/00010_content_blocks_seed.sql`
- `internal/content/seed_test.go`

## Spec
One `INSERT INTO content_blocks (key, page, section, sort, label, type, max_length, text_value, image_url,
image_alt, image_position) VALUES ...;` with one row per entry of `blocks.data.ts`, in file order:
- `page` = the first segment of the key; `sort` = the position of the entry within its page (1, 2, …);
- `section`, `label`, `type`, `max_length` (`maxLength` of the entry, or 0) copied;
- `text`/`richtext`: `text_value` = the value exactly (keep `{{…}}` placeholders; use normal spaces, the site's
  `typo()` is not part of the value); `lines`: the items joined with `E'\n'`;
- `image`: `image_url`, `image_alt`, `image_position` from the entry; `text_value` `''`.
Escape single quotes in SQL (`''`). Down: `DELETE FROM content_blocks WHERE key = ANY(ARRAY[...])` with the same keys.

## Tests to write
- `TestSeedBlocks`: after migrations, `GET /content/blocks` (through the handler) → the number of blocks equals the
  number of entries in `blocks.data.ts` (write that number as a constant in the test, with a comment naming the file; citypark-modern task 16 wrote it under its line in `../citypark-modern/docs/tasks/PROGRESS.md`);
  for three entries of different types (pick one `text`, one `lines`, one `image`), the value equals the file's value.
- `TestSeedBlocksValid`: for every seeded row, `validateValue` of task 50 accepts its own value (catches a placeholder
  name or a length that the rules would reject).
- `TestSeedKeysUnique`: the migration applies on an empty database (covered by `testutil.NewDB`) and
  `SELECT count(DISTINCT page)` equals the pages used in the file.

## Done when
The commands in CONVENTIONS "Done means" pass and with `make run`,
`curl -s 'localhost:3000/api/v1/content/blocks?page=home'` prints the home page blocks with the values of
`blocks.data.ts`.

## Out of scope
Changing the list of blocks (that is a site task plus a new migration), endpoints (task 50).
