# Spatial indexing

This package currently provides the local Singapore projection and walking
travel-time raster. Direct walking and post-transit walking are approximated
with a configurable speed and network-distance factor. The default is 1.3 m/s
and a 1.2 distance factor on a 150-metre grid.

The walking approximation is local and deterministic. A future offline OSM
pedestrian graph can replace its cost source without changing contour output.
Nearest-stop lookup remains to be implemented here.
