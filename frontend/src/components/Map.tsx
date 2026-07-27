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
import {
  PresetOverlaySidebar,
  type PresetOverlay,
} from "./PresetOverlaySidebar";

const api = import.meta.env.VITE_API_URL ?? "http://localhost:8000";
const mapboxAccessToken =
  import.meta.env.MAPBOX_ACCESS_TOKEN ?? import.meta.env.VITE_MAPBOX_ACCESS_TOKEN;
const singapore: L.LatLngExpression = [1.3521, 103.8198];
const overlayColors = ["#f97316", "#16a34a", "#dc2626", "#7c3aed", "#0891b2", "#ca8a04"];

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
  const presetLayerRef = useRef<L.LayerGroup | null>(null);
  const isochroneRequestRef = useRef(0);

  const [mode, setMode] = useState<MapToolMode>("isochrone");
  const [routeSelection, setRouteSelection] = useState<Partial<{ start: Place; end: Place }>>({});
  const [routes, setRoutes] = useState<RouteOption[]>([]);
  const [activeRouteIndex, setActiveRouteIndex] = useState(0);
  const [isochronePlace, setIsochronePlace] = useState<Place | null>(null);
  const [isochroneMinutes, setIsochroneMinutes] = useState(15);
  const [isochroneAreaKm2, setIsochroneAreaKm2] = useState<number | null>(null);
  const [presets, setPresets] = useState<PresetOverlay[]>([]);
  const [activePresetIDs, setActivePresetIDs] = useState<string[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState(mapboxAccessToken ? "" : "MAPBOX_ACCESS_TOKEN is not set.");

  useEffect(() => {
    if (!elementRef.current || mapRef.current) {
      return;
    }

    const map = L.map(elementRef.current).setView(singapore, 12);
    if (mapboxAccessToken) {
      L.tileLayer(
        `https://api.mapbox.com/styles/v1/mapbox/streets-v12/tiles/{z}/{x}/{y}?access_token=${mapboxAccessToken}`,
        {
          attribution:
            '&copy; <a href="https://www.mapbox.com/about/maps/">Mapbox</a> &copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>',
          maxZoom: 19,
          tileSize: 512,
          zoomOffset: -1,
        },
      ).addTo(map);
    }
    mapRef.current = map;
    presetLayerRef.current = L.layerGroup().addTo(map);

    return () => {
      map.remove();
      mapRef.current = null;
      presetLayerRef.current = null;
    };
  }, []);

  useEffect(() => {
    drawPresetOverlays();
  }, [activePresetIDs, presets]);

  function changeMode(nextMode: MapToolMode) {
    setMode(nextMode);
    clearMapLayers();
    setRoutes([]);
    setIsochronePlace(null);
    setIsochroneAreaKm2(null);
    setError("");
  }

  async function selectIsochronePlace(place: Place) {
    setIsochronePlace(place);
    await drawIsochrone(place, isochroneMinutes);
  }

  async function changeIsochroneMinutes(minutes: number) {
    setIsochroneMinutes(minutes);
    if (isochronePlace) {
      await drawIsochrone(isochronePlace, minutes);
    }
  }

  async function drawIsochrone(place: Place, minutes: number) {
    const requestID = isochroneRequestRef.current + 1;
    isochroneRequestRef.current = requestID;

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
        body: JSON.stringify({ lat, lon, minutes }),
      });
      if (!response.ok) {
        throw new Error("Could not load isochrone.");
      }
      const points = (await response.json()) as Point[];
      if (requestID !== isochroneRequestRef.current) {
        return;
      }

      const polygon = points.map((point) => [point.lat, point.lon] as L.LatLngExpression);

      polygonRef.current = L.polygon(polygon, {
        color: "#2563eb",
        fillColor: "#60a5fa",
        fillOpacity: 0.25,
      }).addTo(map);
      setIsochroneAreaKm2(calculatePolygonAreaKm2(points));
      map.fitBounds(polygonRef.current.getBounds());
    } catch (err) {
      if (requestID === isochroneRequestRef.current) {
        setError(err instanceof Error ? err.message : "Could not load isochrone.");
      }
    } finally {
      if (requestID === isochroneRequestRef.current) {
        setIsLoading(false);
      }
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

  function createPresetOverlay(preset: PresetOverlay) {
    setPresets((current) => upsertPreset(current, preset));
    setActivePresetIDs((current) => [...current.filter((presetID) => presetID !== preset.id), preset.id]);
  }

  function togglePreset(preset: PresetOverlay) {
    setPresets((current) => upsertPreset(current, preset));
    setActivePresetIDs((current) =>
      current.includes(preset.id)
        ? current.filter((activePresetID) => activePresetID !== preset.id)
        : [...current, preset.id],
    );
  }

  function drawPresetOverlays() {
    const map = mapRef.current;
    const presetLayer = presetLayerRef.current;
    if (!map || !presetLayer) {
      return;
    }

    presetLayer.clearLayers();
    const activePresets = presets.filter((preset) => activePresetIDs.includes(preset.id));
    const bounds: L.LatLngExpression[] = [];

    activePresets.forEach((preset, presetIndex) => {
      const color = overlayColors[presetIndex % overlayColors.length];
      preset.locations.forEach((location) => {
        const position: L.LatLngExpression = [location.latitude, location.longitude];
        bounds.push(position);
        L.circleMarker(position, {
          radius: 7,
          color,
          fillColor: color,
          fillOpacity: 0.82,
          weight: 2,
        })
          .bindPopup(`<strong>${escapeHTML(location.name)}</strong><br>${escapeHTML(location.address)}`)
          .addTo(presetLayer);
      });
    });

    if (bounds.length > 0) {
      map.fitBounds(L.latLngBounds(bounds), { padding: [60, 60], maxZoom: 14 });
    }
  }

  return (
    <main className="screen">
      <SearchBar
        mode={mode}
        routeSelection={routeSelection}
        isochroneAreaKm2={isochroneAreaKm2}
        isochroneMinutes={isochroneMinutes}
        routeResults={routes}
        activeRouteIndex={activeRouteIndex}
        isLoading={isLoading}
        error={error}
        onModeChange={changeMode}
        onIsochroneSelect={selectIsochronePlace}
        onIsochroneMinutesChange={changeIsochroneMinutes}
        onRoutePlaceChange={changeRoutePlace}
        onRouteSubmit={submitRoute}
        onRouteSelect={selectRoute}
      />
      <PresetOverlaySidebar
        activePresetIDs={activePresetIDs}
        onPresetCreated={createPresetOverlay}
        onTogglePreset={togglePreset}
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

function escapeHTML(value: string) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function upsertPreset(presets: PresetOverlay[], preset: PresetOverlay) {
  const withoutPreset = presets.filter((current) => current.id !== preset.id);
  return [...withoutPreset, preset];
}
