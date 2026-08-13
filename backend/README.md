## Getting Started

### Installing Go

Download and install Go by following the instructions [here](https://go.dev/doc/install).

### Running the app
1. [Fork](https://docs.github.com/en/get-started/quickstart/fork-a-repo#forking-a-repository) this repo.
2. [Clone](https://docs.github.com/en/get-started/quickstart/fork-a-repo#cloning-your-forked-repository) **your** forked repo.
3. Open your terminal and navigate to the directory containing your cloned project.
4. From `backend`, run `go run ./cmd/gtfs-build` and `go run ./cmd/footpath-build` to create the optimized transit and pedestrian artifacts. These steps are optional in development because the server can build both in memory from the raw archives.
5. Run `go run ./cmd/server` and head over to http://localhost:8000/users to view the response.

The server loads the isochrone graph from `GTFS_GRAPH_PATH` when configured. By
default it checks `../gtfs/generated/singapore.graph` and
`gtfs/generated/singapore.graph`. If neither exists, it builds the graph in
memory from `GTFS_FEED_PATH` or the repository's raw Singapore feed.

The walking stage similarly loads `FOOTPATH_GRAPH_PATH`, checks
`../gtfs/generated/singapore.walk` and `gtfs/generated/singapore.walk`, then
builds the graph in memory from `FOOTPATH_FEED_PATH` or
`Footpath_Mar2026.zip`. A bounded multi-source walking Dijkstra starts at the
origin and at every transit-reached stop before marching squares extracts the
MultiPolygon. If no footpath source exists, the server logs the issue and uses
the radial model as a controlled fallback.

The source shapefile is an asset inventory rather than a topological route
network and has no level/access attributes. Its default 15 m raster is a
deliberate conflation: it is the smallest tested resolution with one dominant
Singapore-wide component. Use OSM or a self-hosted pedestrian router when
bridge, crossing, wheelchair, or turn-rule accuracy is required.
Runtime loading rejects artifacts whose largest component contains less than
80% of their walkable cells, preventing a finer but unusably fragmented raster
from silently reaching production.
Before redistributing the source or generated artifact in production, verify
the dataset licence; the archive's embedded XML contains an older
"internal use only" restriction.

`POST /isochrone` no longer calls Mapbox. It accepts:

```json
{
  "lat": 1.3521,
  "lon": 103.8198,
  "minutes": 30,
  "departure_time": "2026-08-13T08:00:00+08:00"
}
```

`departure_time` is optional and defaults to 08:00 on the current date in
`Asia/Singapore`, a representative in-service time for the headway model. Send
an explicit RFC3339 value for a time-specific query. The response is a GeoJSON
Feature containing a MultiPolygon.

Location autocomplete uses Singapore's OneMap Search API through
`GET /places/search?q=<query>&limit=<1-10>`. Configure the server-only
`ONEMAP_API_EMAIL` and `ONEMAP_API_PASSWORD` values so it can cache and renew
OneMap's short-lived access token automatically. `ONEMAP_API_KEY` remains an
optional fallback, but an expired JWT is rejected. The frontend never receives
these credentials. If OneMap is temporarily unavailable, the map search falls
back to Mapbox Geocoding with permanent-result mode using the existing public
Mapbox token; OneMap remains the richer provider for Singapore POIs and bus
stops.


### Navigating the code
This is the main file structure. Note that this is simply *one of* various paradigms to organise your code, and is just a bare starting point.
```
.
├── cmd
│   ├── server
├── internal
│   ├── api         # Encapsulates types and utilities related to the API
│   ├── dataacess   # Data Access layer accesses data from the database
│   ├── database    # Encapsulates the types and utilities related to the database
│   ├── handlers    # Handler functions to respond to requests
│   ├── models      # Definitions of objects used in the application
│   ├── router      # Encapsulates types and utilities related to the router
│   ├── routes      # Defines routes that are used in the application
├── README.md
├── go.mod
└── go.sum
```

Main directories/files to note:
* `cmd` contains the main entry point for the application
* `internal` holds most of the functional code for your project that is specific to the core logic of your application
* `README.md` is a form of documentation about the project. It is what you are reading right now.
* `go.mod` contains important metadata, for example, the dependencies in the project. See [here](https://go.dev/ref/mod) for more information
* `go.sum` See [here](https://go.dev/ref/mod) for more information

