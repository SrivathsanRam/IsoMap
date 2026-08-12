# Isochrone engine

This package will coordinate graph loading, bounded transit routing, travel-time surface construction, and contour generation.

```text
isochrone/
|-- gtfs/       GTFS archive parsing and validation
|-- graph/      Runtime graph model, building, and serialization
|-- routing/    Bounded time-dependent Dijkstra
|-- spatial/    Stop lookup and travel-time grid indexes
`-- contour/    Marching squares and polygon assembly
```

The HTTP layer belongs in `internal/handlers/isochrone`; this package remains independent of HTTP.

`Engine.Generate` currently accepts precomputed origin-stop seeds, then:

1. runs bounded, time-dependent Dijkstra;
2. gives each reached stop a walking radius from its remaining time;
3. creates a raster of minimum travel times, including direct walking;
4. extracts and assembles a GeoJSON `MultiPolygon` with marching squares.

Nearest-stop access is the remaining input-side stage before this engine can
serve arbitrary coordinates without caller-provided stop seeds.

`NewWithSurface` is the integration boundary for a future OSM walking graph or
batched walking API. Such a provider supplies a compatible travel-time grid;
the routing and contour layers remain unchanged.
