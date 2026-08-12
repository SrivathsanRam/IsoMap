# Transit routing

This package runs bounded, time-dependent Dijkstra over a generated transit
graph. Queries begin from stop seeds with walking-access times; spatial origin
lookup is deliberately kept outside the routing core.

Frequency trips currently use the expected-wait approximation:

```text
wait_seconds = headway_secs / 2
```

The search checks service calendars in the graph's agency timezone, supports
GTFS times after `24:00:00`, handles fixed-schedule trips, and stops expanding
nodes beyond the requested travel-time budget. Its dense result array aligns
with `graph.Graph.Stops` and uses `-1` for unreachable stops.
