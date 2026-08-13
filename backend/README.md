# Backend

Go API for accounts, presets, outings, routing, search, and offline isochrones.

## Run

```powershell
go run ./cmd/server
```

Build local graphs when needed:

```powershell
go run ./cmd/gtfs-build
go run ./cmd/footpath-build
```

The server can build missing graphs from the raw ZIP files, but startup is slower.

## Main environment variables

- `DATABASE_URL`
- `GTFS_GRAPH_PATH` / `GTFS_FEED_PATH`
- `FOOTPATH_GRAPH_PATH` / `FOOTPATH_FEED_PATH`
- `ONEMAP_API_EMAIL` / `ONEMAP_API_PASSWORD`
- `ONEMAP_API_KEY` as a fallback

`POST /isochrone` returns a GeoJSON `MultiPolygon`. If `departure_time` is missing, it uses 08:00 Singapore time.

## Check

```powershell
go test ./...
go vet ./...
```
