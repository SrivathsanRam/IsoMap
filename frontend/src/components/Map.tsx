import { useEffect, useRef, useState } from "react";
import L from "leaflet";
import "leaflet/dist/leaflet.css";
import markerIcon from "leaflet/dist/images/marker-icon.png";
import markerIcon2x from "leaflet/dist/images/marker-icon-2x.png";
import markerShadow from "leaflet/dist/images/marker-shadow.png";
import {
  SearchBar,
  type MapToolMode,
  type Place,
  type RouteSummary,
} from "./SearchBar";

const api = import.meta.env.VITE_API_URL ?? "http://localhost:8000";
const singapore: L.LatLngExpression = [1.3521, 103.8198];

L.Icon.Default.mergeOptions({
  iconRetinaUrl: markerIcon2x,
  iconUrl: markerIcon,
  shadowUrl: markerShadow,
});

type Point = {
  lat: number;
  lon: number;
};

type RouteOption = RouteSummary & {
  geometry: Point[];
};

type DirectionsResponse = {
  routes: RouteOption[];
};

export function Map() {
  const elementRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<L.Map | null>(null);
  const markerRef = useRef<L.Marker | null>(null);
  const routeMarkersRef = useRef<L.Marker[]>([]);
  const polygonRef = useRef<L.Polygon | null>(null);
  const routeRef = useRef<L.Polyline | null>(null);

  const [mode, setMode] = useState<MapToolMode>("isochrone");
  const [routeSelection, setRouteSelection] = useState<Partial<{ start: Place; end: Place }>>({});
  const [routes, setRoutes] = useState<RouteOption[]>([]);
  const [activeRouteIndex, setActiveRouteIndex] = useState(0);
  const [isochroneAreaKm2, setIsochroneAreaKm2] = useState<number | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!elementRef.current || mapRef.current) {
      return;
    }

    const map = L.map(elementRef.current).setView(singapore, 12);
    L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
      attribution: "&copy; OpenStreetMap contributors",
    }).addTo(map);
    mapRef.current = map;

    return () => {
      map.remove();
      mapRef.current = null;
    };
  }, []);

  function changeMode(nextMode: MapToolMode) {
    setMode(nextMode);
    clearMapLayers();
    setRoutes([]);
    setIsochroneAreaKm2(null);
    setError("");
  }

  async function selectIsochronePlace(place: Place) {
    const lat = Number(place.lat);
    const lon = Number(place.lon);
    const map = mapRef.current;
    if (!map) {
      return;
    }

    clearMapLayers();
    setIsLoading(true);
    setError("");
    map.setView([lat, lon], 15);
    markerRef.current = L.marker([lat, lon]).addTo(map);

    try {
      const response = await fetch(`${api}/isochrone`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ lat, lon }),
      });
      if (!response.ok) {
        throw new Error("Could not load isochrone.");
      }
      const points = (await response.json()) as Point[];
      const polygon = points.map((point) => [point.lat, point.lon] as L.LatLngExpression);

      polygonRef.current = L.polygon(polygon, {
        color: "#2563eb",
        fillColor: "#60a5fa",
        fillOpacity: 0.25,
      }).addTo(map);
      setIsochroneAreaKm2(calculatePolygonAreaKm2(points));
      map.fitBounds(polygonRef.current.getBounds());
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not load isochrone.");
    } finally {
      setIsLoading(false);
    }
  }

  function changeRoutePlace(field: "start" | "end", place: Place) {
    setRouteSelection((current) => ({ ...current, [field]: place }));
    setRoutes([]);
    setError("");
    clearMapLayers();
  }

  async function submitRoute() {
    if (!routeSelection.start || !routeSelection.end) {
      return;
    }

    setIsLoading(true);
    setError("");
    try {
      const response = await fetch(`${api}/routing/directions`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          start: placeToPoint(routeSelection.start),
          end: placeToPoint(routeSelection.end),
          mode: "transit",
        }),
      });
      if (!response.ok) {
        throw new Error("Could not load routes from OneMap.");
      }

      const data = (await response.json()) as DirectionsResponse;
      setRoutes(data.routes);
      setActiveRouteIndex(0);
      drawRoute(data.routes[0], routeSelection.start, routeSelection.end);
    } catch (err) {
      setRoutes([]);
      setError(err instanceof Error ? err.message : "Could not load routes from OneMap.");
    } finally {
      setIsLoading(false);
    }
  }

  function selectRoute(index: number) {
    setActiveRouteIndex(index);
    if (routeSelection.start && routeSelection.end) {
      drawRoute(routes[index], routeSelection.start, routeSelection.end);
    }
  }

  function drawRoute(route: RouteOption | undefined, start: Place, end: Place) {
    const map = mapRef.current;
    if (!map || !route) {
      return;
    }

    clearMapLayers();

    const line = route.geometry.map((point) => [point.lat, point.lon] as L.LatLngExpression);
    routeRef.current = L.polyline(line, {
      color: "#2563eb",
      weight: 5,
      opacity: 0.85,
    }).addTo(map);

    routeMarkersRef.current = [
      L.marker(placeToLatLng(start)).addTo(map),
      L.marker(placeToLatLng(end)).addTo(map),
    ];

    map.fitBounds(routeRef.current.getBounds(), { padding: [40, 40] });
  }

  function clearMapLayers() {
    markerRef.current?.remove();
    markerRef.current = null;
    polygonRef.current?.remove();
    polygonRef.current = null;
    routeRef.current?.remove();
    routeRef.current = null;
    routeMarkersRef.current.forEach((marker) => marker.remove());
    routeMarkersRef.current = [];
  }

  return (
    <main className="screen">
      <SearchBar
        mode={mode}
        routeSelection={routeSelection}
        isochroneAreaKm2={isochroneAreaKm2}
        routeResults={routes}
        activeRouteIndex={activeRouteIndex}
        isLoading={isLoading}
        error={error}
        onModeChange={changeMode}
        onIsochroneSelect={selectIsochronePlace}
        onRoutePlaceChange={changeRoutePlace}
        onRouteSubmit={submitRoute}
        onRouteSelect={selectRoute}
      />
      <div ref={elementRef} className="map" />
    </main>
  );
}

function placeToPoint(place: Place): Point {
  return {
    lat: Number(place.lat),
    lon: Number(place.lon),
  };
}

function placeToLatLng(place: Place): L.LatLngExpression {
  return [Number(place.lat), Number(place.lon)];
}

function calculatePolygonAreaKm2(points: Point[]) {
  if (points.length < 3) {
    return 0;
  }

  const radiusMeters = 6_371_008.8;
  let area = 0;

  for (let i = 0; i < points.length; i += 1) {
    const current = points[i];
    const next = points[(i + 1) % points.length];
    area += toRadians(next.lon - current.lon) *
      (2 + Math.sin(toRadians(current.lat)) + Math.sin(toRadians(next.lat)));
  }

  return Math.abs((area * radiusMeters * radiusMeters) / 2) / 1_000_000;
}

function toRadians(value: number) {
  return (value * Math.PI) / 180;
}
