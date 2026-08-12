# Spatial indexing

This package currently provides the local Singapore projection and walking
travel-time raster. Direct walking and post-transit walking are approximated
with a configurable speed and network-distance factor. The default is 1.3 m/s
and a 1.2 distance factor on a 150-metre grid.

The walking approximation is local and deterministic. A future offline OSM
pedestrian graph can replace its cost source without changing contour output.

Origin access currently scans the feed's boarding stops, chooses up to 64 stops
within one kilometre, and converts their distance to walking seconds. With only
5,450 feed stops, this scan remains inexpensive and avoids another persisted
index.
