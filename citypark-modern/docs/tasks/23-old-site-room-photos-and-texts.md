# Task 23: Full-size room photos and hotel texts from the old site

## Goal
The two room photos that the old site has in full size replace the 204×142 px copies in `public/images/`, so room
cards are sharp on retina and desktop. The static fallback of the content blocks gets the guest information texts that
citypark-go task 55 put in the database, so the hotel page shows the same text when the backend is down.

## Depends on
20, and citypark-go task 55.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/entities/content/model/blocks.data.ts` (entries `hotel.guest-info.*`),
`../citypark-go/migrations/00011_old_site_hotel_texts.sql` (written by citypark-go task 55; read only).

## Background
`public/images/room-double.jpg` and `room-vip.jpg` are 204×142 px, but cards show them up to about 600 px wide. The
old CRA site (`/workspace/citypark_frontend`, read only) has the same photos at 1280×853. It has no own photo of the
single and «евро» rooms: `room-single.jpg` stays as it is, and the owner will upload real photos in the admin.
`blocks.data.ts` is the fallback that `getPageContent()` uses when the backend does not answer; it must match the
database values (citypark-go task 54 says so).

## Files to change
- `public/images/room-double.jpg`: replace with
  `/workspace/citypark_frontend/src/features/Hotel/components/DoubleRoom/images/double_room.jpg` (two single beds,
  the same scene as now).
- `public/images/room-vip.jpg`: replace with
  `/workspace/citypark_frontend/src/features/Hotel/components/VipRoom/images/vip_room.jpg` (the bedroom with a double
  bed, the same scene as now).
- `src/entities/content/model/blocks.data.ts`: entries `hotel.guest-info.2.text` … `hotel.guest-info.6.text`.

## Spec

### Photos
Copy the files byte for byte (`cp`), keep the names, so every place that uses them (the room rows in the backend,
`blocks.data.ts` lines with `/images/room-*.jpg`) keeps working. Keep `object-position` values as they are; then look
at `/rooms`, `/hotel` and `/` at 390 and 1440 px and fix a `photo_position` only if the crop shows the wrong part of
the room (it is a backend value: if one needs to change, write it under the task line in PROGRESS for the user
instead of changing it).

### Fallback texts
For each of the five keys, set `value` to the exact `text_value` of migration 00011 (the same HTML, the same
`{{breakfastDelivery}}` placeholder), and `maxLength: 6000` as in the migration. Keep every other entry unchanged.

## Tests to write
No unit-test runner. On a production build:
- `/hotel`: open each «Информация для гостей» item → none shows «Скоро добавим.»; «Порча имущества» lists
  «Халат — 2 000 ₽, частичная порча — 900 ₽»; the breakfast section shows «100 ₽» in place of the placeholder.
- Same check with the backend unreachable (ask the user to stop it, or set `MENU_API_URL` to a closed port for the
  run): the texts come from `blocks.data.ts` and look the same.
- `/rooms` at 1440 px: the double and VIP photos are sharp; `ls -la public/images/room-*.jpg` shows the new sizes.

## Done when
CONVENTIONS "Done means" passes; `node scripts/qa/flows/hotel.mjs`, `node scripts/qa/flows/rooms.mjs` and
`npm run qa:responsive` print no unexpected values.

## Out of scope
Room descriptions (they come only from the backend), new photos for the single and «евро» rooms, the VIP gallery,
3D tours (task 22).
