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
- [ ] 14 Hotel prices as numbers
- [ ] 15 Content blocks, part 1: the entity and the home page [13, 14]
- [ ] 16 Content blocks, part 2: the hotel and rooms pages [15]
- [ ] 17 Rooms from the backend [13, 15] (needs citypark-go tasks 45, 48)
- [ ] 18 News from the backend [12, 17] (needs citypark-go task 49)
- [ ] 19 Site settings from the backend [17] (needs citypark-go task 51)
- [ ] 20 Content blocks and hotel prices from the backend [16, 17] (needs citypark-go tasks 50, 52, 54)
