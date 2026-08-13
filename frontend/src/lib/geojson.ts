import type { GeoJSONPosition, MultiPolygonGeometry } from "../../types";

const coordinateEpsilon = 1e-10;

export function pointInMultiPolygon(
  longitude: number,
  latitude: number,
  geometry: MultiPolygonGeometry | null,
): boolean {
  if (!geometry || !Number.isFinite(longitude) || !Number.isFinite(latitude)) {
    return false;
  }

  return geometry.coordinates.some((polygon) => {
    if (polygon.length === 0) {
      return false;
    }
    const exterior = pointInRing(longitude, latitude, polygon[0]);
    if (exterior === "outside") {
      return false;
    }
    if (exterior === "boundary") {
      return true;
    }

    for (const hole of polygon.slice(1)) {
      const holePosition = pointInRing(longitude, latitude, hole);
      if (holePosition === "inside") {
        return false;
      }
      if (holePosition === "boundary") {
        return true;
      }
    }
    return true;
  });
}

export function normalizeMultiPolygon(
  geometry: MultiPolygonGeometry,
): MultiPolygonGeometry | null {
  const coordinates = geometry.coordinates.flatMap((polygon) => {
    if (polygon.length === 0) {
      return [];
    }
    const exterior = normalizeRing(polygon[0]);
    if (!exterior) {
      return [];
    }
    const holes = polygon
      .slice(1)
      .map(normalizeRing)
      .filter((ring): ring is GeoJSONPosition[] => ring !== null);
    return [[exterior, ...holes]];
  });

  return coordinates.length > 0 ? { type: "MultiPolygon", coordinates } : null;
}

export function multiPolygonAreaKm2(geometry: MultiPolygonGeometry): number {
  return geometry.coordinates.reduce((total, polygon) => {
    if (polygon.length === 0) {
      return total;
    }
    const exteriorArea = sphericalRingAreaKm2(polygon[0]);
    const holesArea = polygon
      .slice(1)
      .reduce((holeTotal, ring) => holeTotal + sphericalRingAreaKm2(ring), 0);
    return total + Math.max(0, exteriorArea - holesArea);
  }, 0);
}

function normalizeRing(ring: GeoJSONPosition[]): GeoJSONPosition[] | null {
  const cleaned: GeoJSONPosition[] = [];
  for (const position of ring) {
    if (
      position.length < 2 ||
      !Number.isFinite(position[0]) ||
      !Number.isFinite(position[1])
    ) {
      continue;
    }
    const next: GeoJSONPosition = [position[0], position[1]];
    const previous = cleaned[cleaned.length - 1];
    if (!previous || !samePosition(previous, next)) {
      cleaned.push(next);
    }
  }
  if (cleaned.length < 3) {
    return null;
  }
  if (!samePosition(cleaned[0], cleaned[cleaned.length - 1])) {
    cleaned.push([...cleaned[0]]);
  }
  return cleaned.length >= 4 ? cleaned : null;
}

function pointInRing(
  longitude: number,
  latitude: number,
  ring: GeoJSONPosition[],
): "inside" | "outside" | "boundary" {
  let inside = false;
  for (let index = 0, previousIndex = ring.length - 1; index < ring.length; previousIndex = index++) {
    const current = ring[index];
    const previous = ring[previousIndex];
    if (pointOnSegment(longitude, latitude, previous, current)) {
      return "boundary";
    }
    const crossesLatitude = current[1] > latitude !== previous[1] > latitude;
    if (
      crossesLatitude &&
      longitude <
        ((previous[0] - current[0]) * (latitude - current[1])) /
          (previous[1] - current[1]) +
          current[0]
    ) {
      inside = !inside;
    }
  }
  return inside ? "inside" : "outside";
}

function pointOnSegment(
  longitude: number,
  latitude: number,
  start: GeoJSONPosition,
  end: GeoJSONPosition,
): boolean {
  const cross =
    (longitude - start[0]) * (end[1] - start[1]) -
    (latitude - start[1]) * (end[0] - start[0]);
  if (Math.abs(cross) > coordinateEpsilon) {
    return false;
  }
  return (
    longitude >= Math.min(start[0], end[0]) - coordinateEpsilon &&
    longitude <= Math.max(start[0], end[0]) + coordinateEpsilon &&
    latitude >= Math.min(start[1], end[1]) - coordinateEpsilon &&
    latitude <= Math.max(start[1], end[1]) + coordinateEpsilon
  );
}

function samePosition(left: GeoJSONPosition, right: GeoJSONPosition): boolean {
  return (
    Math.abs(left[0] - right[0]) <= coordinateEpsilon &&
    Math.abs(left[1] - right[1]) <= coordinateEpsilon
  );
}

function sphericalRingAreaKm2(points: GeoJSONPosition[]) {
  if (points.length < 4) {
    return 0;
  }

  const earthRadiusKm = 6371.0088;
  let sum = 0;
  for (let index = 0; index < points.length - 1; index += 1) {
    const [lon1, lat1] = points[index];
    const [lon2, lat2] = points[index + 1];
    sum +=
      toRadians(lon2 - lon1) *
      (2 + Math.sin(toRadians(lat1)) + Math.sin(toRadians(lat2)));
  }
  return Math.abs((sum * earthRadiusKm * earthRadiusKm) / 2);
}

function toRadians(degrees: number) {
  return (degrees * Math.PI) / 180;
}
