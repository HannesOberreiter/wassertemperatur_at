# Agent Notes

## Project

Go web app for `wassertemperatur.at`.

- Entry: `main.go`
- App code: `internal/`
- Static assets: `assets/`
- DB path default: `db/wasser.db`
- Server port: `1323`

## Commands

```bash
go run . fetch   # fetch data once

go run .         # start server

air              # dev reload

go test ./...
```

Docker fetch:

```bash
docker compose exec web /wassertemperatur fetch
```

## Important

- Do not delete `db/` during debugging; it contains local fetched data.
- Startup must not fetch data. Fetch via `go run . fetch` or cron only.
- Cron default is hourly (`CRON_INTERVAL=1h`).
- Do not scrape Bergfex; user said not allowed.
- AGES data belongs on `/wasserqualitaet`, not main temperature table.
- Main temperature table excludes AGES.
- State dropdowns are data-driven via `ListStates`, not hardcoded.
- HTMX is local: `assets/js/htmx.min.js`; avoid CDN scripts.

## Current data sources

- AGES Badegewässerdatenbank: quality page + AGES detail parameters.
- Oberösterreich ZRXP source.
- Kärnten hydrographie JSON; accepts timestamps with and without seconds.
- Ausseerland scraper exists but source page currently has no server-rendered table, so may return no rows.

## UI notes

- German website text.
- Flat design with dark mode in `assets/css/main.css`.
- Aquavital sponsor is shown above bottom area.
- Sources, creator, and GitHub link are on `/impressum`.
- Detail pages:
  - normal sources: daily avg/median/high/low/count with paging.
  - AGES: all reported parameters (temperature, depth, quality, enterococci, E. coli).
