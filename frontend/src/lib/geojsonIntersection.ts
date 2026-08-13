import { featureCollection, multiPolygon } from "@turf/helpers";
import intersect from "@turf/intersect";
import type { GeoJSONPosition, MultiPolygonGeometry } from "../../types";
import { normalizeMultiPolygon } from "./geojson.ts";

export function intersectMultiPolygons(
  geometries: MultiPolygonGeometry[],
): MultiPolygonGeometry | null {
  const normalized = geometries
    .map(normalizeMultiPolygon)
    .filter((geometry): geometry is MultiPolygonGeometry => geometry !== null);
  // Every member constrains the result. An empty/invalid member area makes the
  // all-members intersection empty; it must never be silently skipped.
  if (normalized.length === 0 || normalized.length !== geometries.length) {
    return null;
  }
  if (normalized.length === 1) {
    return normalized[0];
  }

  const result = intersect(
    featureCollection(normalized.map((geometry) => multiPolygon(geometry.coordinates))),
  );
  if (!result) {
    return null;
  }
  if (result.geometry.type === "Polygon") {
    return normalizeMultiPolygon({
      type: "MultiPolygon",
      coordinates: [copyPolygonCoordinates(result.geometry.coordinates)],
    });
  }
  if (result.geometry.type === "MultiPolygon") {
    return normalizeMultiPolygon({
      type: "MultiPolygon",
      coordinates: result.geometry.coordinates.map(copyPolygonCoordinates),
    });
  }
  return null;
}

function copyPolygonCoordinates(polygon: number[][][]): GeoJSONPosition[][] {
  return polygon.map((ring) =>
    ring.map((position) => [position[0] ?? Number.NaN, position[1] ?? Number.NaN]),
  );
}
