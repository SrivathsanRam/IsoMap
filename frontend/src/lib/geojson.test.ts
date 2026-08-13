import assert from "node:assert/strict";
import test from "node:test";
import type { GeoJSONPosition, MultiPolygonGeometry } from "../../types";
import { pointInMultiPolygon } from "./geojson.ts";
import { intersectMultiPolygons } from "./geojsonIntersection.ts";

test("pointInMultiPolygon handles disconnected polygons, holes, and boundaries", () => {
  const geometry: MultiPolygonGeometry = {
    type: "MultiPolygon",
    coordinates: [
      [square(0, 0, 8, 8), square(2, 2, 4, 4)],
      [square(10, 10, 12, 12)],
    ],
  };

  assert.equal(pointInMultiPolygon(1, 1, geometry), true);
  assert.equal(pointInMultiPolygon(3, 3, geometry), false, "points in holes are excluded");
  assert.equal(pointInMultiPolygon(2, 3, geometry), true, "hole boundaries belong to the polygon");
  assert.equal(pointInMultiPolygon(11, 11, geometry), true, "second polygon is included");
  assert.equal(pointInMultiPolygon(9, 9, geometry), false);
});

test("intersectMultiPolygons preserves multiple disconnected common areas", () => {
  const first: MultiPolygonGeometry = {
    type: "MultiPolygon",
    coordinates: [[square(0, 0, 4, 4)], [square(10, 0, 14, 4)]],
  };
  const second: MultiPolygonGeometry = {
    type: "MultiPolygon",
    coordinates: [[square(2, 1, 12, 3)]],
  };
  const third: MultiPolygonGeometry = {
    type: "MultiPolygon",
    coordinates: [[square(1, 1.5, 13, 2.5)]],
  };

  const overlap = intersectMultiPolygons([first, second, third]);
  assert.ok(overlap);
  assert.equal(overlap.type, "MultiPolygon");
  assert.equal(overlap.coordinates.length, 2);
  assert.equal(pointInMultiPolygon(3, 2, overlap), true);
  assert.equal(pointInMultiPolygon(11, 2, overlap), true);
  assert.equal(pointInMultiPolygon(7, 2, overlap), false);
});

test("intersectMultiPolygons respects holes and returns null for an empty common area", () => {
  const withHole: MultiPolygonGeometry = {
    type: "MultiPolygon",
    coordinates: [[square(0, 0, 10, 10), square(3, 3, 7, 7)]],
  };
  const insideHole: MultiPolygonGeometry = {
    type: "MultiPolygon",
    coordinates: [[square(4, 4, 6, 6)]],
  };

  assert.equal(intersectMultiPolygons([withHole, insideHole]), null);
  assert.equal(
    intersectMultiPolygons([
      withHole,
      { type: "MultiPolygon", coordinates: [] },
      withHole,
    ]),
    null,
    "an empty member is not ignored",
  );
});

function square(minX: number, minY: number, maxX: number, maxY: number): GeoJSONPosition[] {
  return [
    [minX, minY],
    [maxX, minY],
    [maxX, maxY],
    [minX, maxY],
    [minX, minY],
  ];
}
