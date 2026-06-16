# Wassertemperatur Österreich

Go-Webapp für aktuelle Wassertemperaturen in Österreich.

## Entwicklung

```bash
go run . fetch

go run .
# optional mit Air
air
```

## Docker

```bash
docker compose up --build
```

Datenabruf im Container manuell starten:

```bash
docker compose exec web /wassertemperatur fetch
```

## Konfiguration

- `SQL_PATH`: SQLite-Datei, Standard `db/wasser.db`
- `CRON_INTERVAL`: Abrufintervall, Standard `1h`
