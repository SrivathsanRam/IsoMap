# Isochrone engine

This package will coordinate graph loading, bounded transit routing, travel-time surface construction, and contour generation.

```text
isochrone/
|-- gtfs/       GTFS archive parsing and validation
|-- graph/      Runtime graph model, building, and serialization
|-- routing/    Bounded time-dependent Dijkstra
|-- spatial/    Stop lookup and travel-time grid indexes
|-- walking/    Footpath ZIP parsing and versioned raster artifacts
|-- footpath/   Bounded pedestrian access/egress Dijkstra
`-- contour/    Marching squares and polygon assembly
```

The HTTP layer belongs in `internal/handlers/isochrone`; this package remains independent of HTTP.

`Engine.Generate` automatically finds nearby origin stops, then:

1. runs bounded, time-dependent Dijkstra;
2. seeds the walking graph at the origin and every reached stop's arrival time;
3. runs bounded multi-source walking Dijkstra and creates a travel-time raster;
4. extracts and assembles a GeoJSON `MultiPolygon` with marching squares.

Callers may still supply explicit origin-stop seeds for deterministic tests or
specialized access models. When no transit stop is within the configured access
radius, the engine returns a direct-walking isochrone.

`NewWithWalkingProviders` installs the same walking implementation for origin
access and post-transit egress. The server uses the locally generated LTA
footpath raster and retains the radial implementation only as a missing-data
fallback. An OSM graph can replace the walking provider without changing the
transit or contour layers.
