# Singapore GTFS data

This directory contains the source feed, small test fixtures, and locally generated graph artifacts used by the isochrone engine.

```text
gtfs/
|-- raw/         Source GTFS archives
|-- fixtures/    Small synthetic feeds used by tests
`-- generated/   Build output consumed by the backend
```

The generated directory is intentionally excluded from version control except for its README. Graph artifacts are reproducible outputs of `backend/cmd/gtfs-build` and `backend/cmd/footpath-build`.

From the `backend` directory, validate and build the current feed with:

```powershell
go run ./cmd/gtfs-build
go run ./cmd/footpath-build
```

To validate without producing artifacts:

```powershell
go run ./cmd/gtfs-build -validate-only
```

The raw archive is the source of truth. Production packaging should include
`generated/singapore.graph` plus `generated/singapore.walk` and can omit the
raw ZIP files because request handling only needs the generated artifacts.

If a packaged graph is unavailable, the backend can build the same artifact in
memory from the raw feed during startup. Set `GTFS_GRAPH_PATH` or
`GTFS_FEED_PATH` when deployment paths differ from the repository layout.
The equivalent walking overrides are `FOOTPATH_GRAPH_PATH` and
`FOOTPATH_FEED_PATH`.
