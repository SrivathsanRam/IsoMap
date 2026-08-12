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
