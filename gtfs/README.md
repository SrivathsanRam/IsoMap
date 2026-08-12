# Singapore GTFS data

This directory contains the source feed, small test fixtures, and locally generated graph artifacts used by the isochrone engine.

```text
gtfs/
|-- raw/         Source GTFS archives
|-- fixtures/    Small synthetic feeds used by tests
`-- generated/   Build output consumed by the backend
```

The generated directory is intentionally excluded from version control except for its README. Graph artifacts will be reproducible outputs of `backend/cmd/gtfs-build`.
