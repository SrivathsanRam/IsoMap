# ISOMAP

Singapore travel planning with transit isochrones, routing, presets, and group overlap.

## Features

- Offline GTFS transit isochrones
- Footpath-based walking access and egress
- OneMap search and public-transit routing
- Preset place overlays and isochrone filtering
- Group travel-area intersections
- Saved places and outing links

Mapbox is used for map tiles only.

## Run

Backend:

```powershell
cd backend
go run ./cmd/gtfs-build
go run ./cmd/footpath-build
go run ./cmd/server
```

Frontend:

```powershell
cd frontend
npm install
npm run dev
```

Copy the relevant `.env.example` files before starting. The backend uses PostgreSQL and server-side OneMap credentials.

## Check

```powershell
cd backend
go test ./...

cd ../frontend
npm run test:geometry
npm run lint
npm run build
```

GTFS files and generated graph notes are in [`gtfs/`](gtfs/README.md).
