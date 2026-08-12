# GTFS parsing

This package reads the required tables directly from a GTFS ZIP and validates
the fields needed by the transit graph. Parsing checks CSV structure and scalar
syntax; `Feed.Validate` reports semantic and cross-table errors together.

Supported tables:

- `feed_info.txt` (optional)
- `agency.txt`
- `stops.txt`
- `routes.txt`
- `calendar.txt`
- `trips.txt`
- `stop_times.txt`
- `frequencies.txt` (optional)
- `transfers.txt` (optional)

`calendar_dates.txt`, pathway routing, and route-specific transfer rules are
not yet part of the graph format.
