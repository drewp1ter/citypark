# Progress

Take the first unchecked task. Tick it only when every "Done when" command passes. The numbers in brackets are the
tasks it depends on. A task that needs a citypark-go task says so; check `../citypark-go/docs/tasks/PROGRESS.md` and
stop if that task is not ticked.

## Foundation
- [x] 01 Project scaffold
- [x] 02 Typed API client and token storage [01]
- [x] 03 Sign-in, access guard and app shell [02]

## Operations
- [x] 04 Delivery prices [03]
- [x] 05 Order history [03] (needs citypark-go task 42)
- [x] 06 Free rooms (read only) [03]
- [x] 07 Menu (read only) and the sync button [03] (needs citypark-go task 43)

## Shared editors
- [ ] 08 Image field (upload, alt text, focal point) [02] (needs citypark-go task 45)
- [-] 09 Dish photos REMOVED: dish photos come from iiko
- [ ] 10 Rich text editor [01]

## Content
- [ ] 11 Rooms [08, 10] (needs citypark-go task 48)
- [ ] 12 News [11] (needs citypark-go task 49)
- [ ] 13 Page content (home and hotel) [08, 10] (needs citypark-go tasks 50, 54)
- [ ] 14 Site settings (contacts) [11] (needs citypark-go task 51)
- [ ] 15 Hotel prices [14] (needs citypark-go task 52)

## Reports and shipping
- [ ] 16 Hotel reports [03] (needs citypark-go task 53)
- [ ] 17 Docker image and deployment notes [16]
