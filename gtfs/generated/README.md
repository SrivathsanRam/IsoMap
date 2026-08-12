# Generated graph artifacts

`go run ./cmd/gtfs-build` from the `backend` directory writes:

- `singapore.graph`: compressed, versioned runtime graph
- `manifest.json`: human-readable feed hash, schema, counts, and bounds

Generated files are ignored by Git. Generate them during a release build and
package `singapore.graph` with the backend. They do not belong in the React
asset bundle.
