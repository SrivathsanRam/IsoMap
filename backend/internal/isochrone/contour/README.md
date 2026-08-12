# Contour generation

This package extracts a travel-time threshold with marching squares, resolves
ambiguous saddle cells, stitches segments into closed rings, removes rings
below a configurable area, classifies nested rings as holes, normalizes ring
orientation, and returns RFC 7946 GeoJSON `MultiPolygon` coordinates.
