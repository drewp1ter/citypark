# 0001: Admin panel is a separate HeroUI SPA; editable content lives in citypark-go

Date: 2026-10-22. Status: accepted.

## Context

Staff need to edit delivery prices, see order history, edit the site's content (texts, photos, room and hotel prices)
and see the hotel reports from 1C. The old admin (`citypark_ai`, CRA + Redux, served at `/manage`) did menu, news,
blog, room counts and the 1C reports. The new site (`citypark-modern`) keeps its content in TypeScript data files and
JSX, and the backend (`citypark-go`) has no content tables.

## Decision

- **A new project `citypark-admin/`**: Vite + React 19 SPA, HeroUI v3, TanStack Router + Query, types generated from
  `citypark-go/internal/apidocs/openapi.yaml`. It calls `citypark-go` directly with the Bearer JWT (kept in
  `sessionStorage`) and is served as static files on its own subdomain (`admin.<domain>`). Russian UI only.
- **Only the existing `admin` role.** No "rooms only" or "manager" role.
- **Content lives in `citypark-go`**: typed tables for rooms, news, site settings and hotel prices, and key-value
  **content blocks** for one-off page copy. Block keys are created only by migrations; the admin edits values.
  The site's `entities/*/api` getters read them (time-based revalidation, 60 s). No drafts: a save is live.
- **Images** are uploaded to `citypark-go` and stored on a local disk volume (`UPLOAD_DIR`), served at `/uploads/...`.
  `next/image` on the site does the resizing. Every photo has `alt` and an optional focal point (`position`).
  A replaced or deleted photo's file is removed after the database commit.
- **Rich text** (Tiptap in the admin → HTML → sanitized by `bluemonday` in Go) for the news body, room descriptions,
  hotel texts and home page text blocks. Two profiles: *article* (news) and *short* (everything else). Titles,
  labels and captions stay plain text.
- **Prices are numbers stored once.** Room prices only on the room; hotel fees and the breakfast list are number
  fields; texts that show a price use placeholders (`{{smokingFine}}`) that the site fills in. Dish prices stay in iiko
  and are never edited here.
- **Menu**: read-only in the admin; names, prices, descriptions and dish photos all come from iiko. A sync button calls
  an admin endpoint; the backend also syncs on a timer. One lock prevents two syncs at once.
- **Orders**: read-only history with server-side search. No confirm, delete or item editing from the admin.
- **Reports**: only the hotel reports (1C `CityParkHotel`) in v1, through a `citypark-go` proxy that holds the 1C
  credentials. The restaurant reports of the old admin are not ported: restaurant sales are in iiko now, and its
  report API (iikoServer OLAP) is a separate integration for later.
- **The blog is removed** from the site, although `design/reference/` shows it: nobody will maintain it.

## Rejected

- **Payload CMS inside `citypark-modern`**: content editing would be free, but it means a second admin UI and login
  next to the HeroUI app (orders, delivery prices and reports are backend data and would be hand-built screens anyway),
  and the public site would own a database and media storage.
- **Directus / Strapi**: one more service to run, a second set of admin users.
- **Calling 1C from the browser** as the old admin did: it shipped the 1C credentials in the JS bundle.

## Consequences

- Work order: backend endpoints → admin screens → site getters switch to the API. The site first moves hardcoded copy
  into content blocks with keys (still static), so the switch to the API changes only the `api/` segments.
- The old `/manage` admin is switched off by hand by the owner after the new admin goes live.
