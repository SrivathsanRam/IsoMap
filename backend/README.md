## Getting Started

### Installing Go

Download and install Go by following the instructions [here](https://go.dev/doc/install).

### Running the app
1. [Fork](https://docs.github.com/en/get-started/quickstart/fork-a-repo#forking-a-repository) this repo.
2. [Clone](https://docs.github.com/en/get-started/quickstart/fork-a-repo#cloning-your-forked-repository) **your** forked repo.
3. Open your terminal and navigate to the directory containing your cloned project.
4. From `backend`, run `go run ./cmd/gtfs-build` to create the optimized transit graph. This step is optional in development because the server can build it in memory from the raw feed.
5. Run `go run ./cmd/server` and head over to http://localhost:8000/users to view the response.

The server loads the isochrone graph from `GTFS_GRAPH_PATH` when configured. By
default it checks `../gtfs/generated/singapore.graph` and
`gtfs/generated/singapore.graph`. If neither exists, it builds the graph in
memory from `GTFS_FEED_PATH` or the repository's raw Singapore feed.

`POST /isochrone` no longer calls Mapbox. It accepts:

```json
{
  "lat": 1.3521,
  "lon": 103.8198,
  "minutes": 30,
  "departure_time": "2026-08-13T08:00:00+08:00"
}
```

`departure_time` is optional and defaults to the current time in
`Asia/Singapore`. The response is a GeoJSON Feature containing a MultiPolygon.


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

