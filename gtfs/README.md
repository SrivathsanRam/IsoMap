# Singapore GTFS data

This directory contains the source feed, small test fixtures, and locally generated graph artifacts used by the isochrone engine.

```text
gtfs/
|-- raw/         Source GTFS archives
|-- fixtures/    Small synthetic feeds used by tests
`-- generated/   Build output consumed by the backend
```

The generated directory is intentionally excluded from version control except for its README. Graph artifacts are reproducible outputs of `backend/cmd/gtfs-build`.

From the `backend` directory, validate and build the current feed with:

```powershell
go run ./cmd/gtfs-build
```

To validate without producing artifacts:

```powershell
go run ./cmd/gtfs-build -validate-only
```

The raw archive is the source of truth. Production packaging should include
`generated/singapore.graph` and can omit the raw ZIP because request handling
only needs the generated graph.
