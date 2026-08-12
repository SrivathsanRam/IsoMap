# Transit graph

This package expands validated GTFS records into a time-dependent transit
graph. Every stop-time occurrence has separate arrival and departure nodes so
dwell, riding, boarding, and alighting remain distinct operations.

The graph is stored as a gzip-compressed, versioned Go binary with a compact
adjacency index. A JSON manifest records its schema, source hash, feed version,
counts, and geographic bounds. The binary artifact is a backend/runtime asset;
it is not a frontend asset.
