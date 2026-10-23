# Progress

Take the first unchecked task. Tick it only when every "Done when" command passes. The numbers in brackets are the tasks it depends on.

## Backend integration
- [x] 01 Menu from the backend API

## Menu
- [x] 02 Dish description on the menu card [01]
- [x] 03 Cart sheet on the mobile menu [01]
- [x] 04 Quantity steppers in the desktop cart panel [03]
- [x] 05 Solid menu toolbar and category arrows

## Forms
- [x] 06 Dropdown Select on React Aria
- [x] 07 Clear button for text inputs

## Home page
- [x] 08 Free rooms count on the home page
- [x] 09 Free rooms from the backend [08] (needs citypark-go task 41)

## Brand
- [x] 10 Brand logo in the header and footer

## Rooms page
- [x] 11 Room availability from the backend [09] (needs citypark-go task 41)

## Admin panel support (see ../docs/adr/0001-admin-panel.md)
- [x] 12 Remove the chef's blog
- [x] 13 Room prices from one place
- [x] 14 Hotel prices as numbers
- [x] 15 Content blocks, part 1: the entity and the home page [13, 14]
- [x] 16 Content blocks, part 2: the hotel and rooms pages [15]
  - `blocks.data.ts` has 101 entries (46 home + 55 hotel). The content code now lives in `entities/content`.
- [x] 17 Rooms from the backend [13, 15] (needs citypark-go tasks 45, 48)
- [x] 18 News from the backend [12, 17] (needs citypark-go task 49)
- [x] 19 Site settings from the backend [17] (needs citypark-go task 51)
- [x] 20 Content blocks and hotel prices from the backend [16, 17] (needs citypark-go tasks 50, 52, 54)

## Content from the old site (/workspace/citypark_frontend)
- [x] 21 Legal pages from the old site
  - Owner: check the offer is current (domain cityparkvip.ru, payment methods) and send the classification certificate if it changed.
- [x] 22 3D tours from the old site
  - `public/tour` is 46M (`du -sh`): the owner can decide later whether to move it to a CDN.
- [x] 23 Full-size room photos and hotel texts from the old site [20] (needs citypark-go task 55)
- [x] 24 Short room text on the rooms page [17, 23] (needs citypark-go task 56)
  - No other place rendered `descriptionHtml` on a card. Checked on a build at 1440 px (light, dark) and 390 px; first row is about 455 px tall. In `qa:responsive` the 400s on `/` and `/news` come from Next's private-IP image guard for backend uploads (`192.168.4.15`), not from this task.
