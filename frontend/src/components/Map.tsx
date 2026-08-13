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
import { getIsochrone, type MultiPolygonGeometry } from "../lib/api";
import {
  DEFAULT_ISOCHRONE_DEPARTURE_TIME,
  toSingaporeDepartureRFC3339,
} from "../lib/departureTime";
import { multiPolygonAreaKm2, pointInMultiPolygon } from "../lib/geojson";

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
  segments?: Array<{
    mode: string;
    geometry: Point[];
  }>;
};

type DirectionsResponse = {
  routes: RouteOption[];
};

export function Map() {
  const elementRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<L.Map | null>(null);
  const markerRef = useRef<L.Marker | null>(null);
  const routeMarkersRef = useRef<L.Marker[]>([]);
  const polygonRef = useRef<L.GeoJSON | null>(null);
  const routeRef = useRef<L.FeatureGroup | null>(null);
  const presetLayerRef = useRef<L.LayerGroup | null>(null);
  const isochroneRequestRef = useRef(0);
  const routeRequestRef = useRef(0);
  const departureRequestTimerRef = useRef<number | null>(null);

  const [mode, setMode] = useState<MapToolMode>("isochrone");
  const [routeSelection, setRouteSelection] = useState<Partial<{ start: Place; end: Place }>>({});
  const [routes, setRoutes] = useState<RouteOption[]>([]);
  const [activeRouteIndex, setActiveRouteIndex] = useState(0);
  const [isochronePlace, setIsochronePlace] = useState<Place | null>(null);
  const [isochroneMinutes, setIsochroneMinutes] = useState(15);
  const [isochroneDepartureTime, setIsochroneDepartureTime] = useState(
    DEFAULT_ISOCHRONE_DEPARTURE_TIME,
  );
  const [searchProximity, setSearchProximity] = useState({ lat: 1.3521, lon: 103.8198 });
  const [isochroneAreaKm2, setIsochroneAreaKm2] = useState<number | null>(null);
  const [isochroneTransitUsed, setIsochroneTransitUsed] = useState<boolean | null>(null);
  const [isochroneGeometry, setIsochroneGeometry] = useState<MultiPolygonGeometry | null>(null);
  const [showOnlyPresetsInsideIsochrone, setShowOnlyPresetsInsideIsochrone] = useState(false);
  const [presets, setPresets] = useState<PresetOverlay[]>([]);
  const [activePresetIDs, setActivePresetIDs] = useState<string[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState(mapboxAccessToken ? "" : "MAPBOX_ACCESS_TOKEN is not set.");
  const presetFilterGeometry = showOnlyPresetsInsideIsochrone ? isochroneGeometry : null;

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

    const updateSearchProximity = () => {
      const center = map.getCenter();
      setSearchProximity({ lat: center.lat, lon: center.lng });
    };
    map.on("moveend", updateSearchProximity);

    return () => {
      map.off("moveend", updateSearchProximity);
      map.remove();
      mapRef.current = null;
      presetLayerRef.current = null;
      if (departureRequestTimerRef.current !== null) {
        window.clearTimeout(departureRequestTimerRef.current);
      }
    };
  }, []);

  useEffect(() => {
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
      const visibleLocations = presetFilterGeometry
        ? preset.locations.filter((location) =>
            pointInMultiPolygon(location.longitude, location.latitude, presetFilterGeometry),
          )
        : preset.locations;
      visibleLocations.forEach((location) => {
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
  }, [activePresetIDs, presetFilterGeometry, presets]);

  function changeMode(nextMode: MapToolMode) {
    cancelScheduledDepartureRequest();
    isochroneRequestRef.current += 1;
    routeRequestRef.current += 1;
    setMode(nextMode);
    clearMapLayers();
    setRoutes([]);
    setIsochronePlace(null);
    setIsochroneAreaKm2(null);
    setIsochroneTransitUsed(null);
    setIsochroneGeometry(null);
    setShowOnlyPresetsInsideIsochrone(false);
    setIsLoading(false);
    setError("");
  }

  async function selectIsochronePlace(place: Place) {
    cancelScheduledDepartureRequest();
    setIsochronePlace(place);
    await drawIsochrone(place, isochroneMinutes, isochroneDepartureTime);
  }

  function clearIsochroneSelection() {
    cancelScheduledDepartureRequest();
    isochroneRequestRef.current += 1;
    setIsochronePlace(null);
    setIsochroneAreaKm2(null);
    setIsochroneTransitUsed(null);
    setIsochroneGeometry(null);
    setShowOnlyPresetsInsideIsochrone(false);
    setIsLoading(false);
    setError("");
    clearMapLayers();
  }

  async function changeIsochroneMinutes(minutes: number) {
    cancelScheduledDepartureRequest();
    setIsochroneMinutes(minutes);
    if (isochronePlace) {
      await drawIsochrone(isochronePlace, minutes, isochroneDepartureTime);
    }
  }

  function changeIsochroneDepartureTime(departureTime: string) {
    setIsochroneDepartureTime(departureTime);
    cancelScheduledDepartureRequest();
    if (isochronePlace && departureTime) {
      isochroneRequestRef.current += 1;
      departureRequestTimerRef.current = window.setTimeout(() => {
        departureRequestTimerRef.current = null;
        void drawIsochrone(isochronePlace, isochroneMinutes, departureTime);
      }, 350);
    }
  }

  function cancelScheduledDepartureRequest() {
    if (departureRequestTimerRef.current !== null) {
      window.clearTimeout(departureRequestTimerRef.current);
      departureRequestTimerRef.current = null;
    }
  }

  async function drawIsochrone(place: Place, minutes: number, departureTime: string) {
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
    setIsochroneTransitUsed(null);
    map.setView([lat, lon], 15);
    markerRef.current = L.marker([lat, lon]).addTo(map);

    try {
      const feature = await getIsochrone(
        { lat, lon },
        minutes,
        toSingaporeDepartureRFC3339(departureTime),
      );
      if (requestID !== isochroneRequestRef.current) {
        return;
      }

      polygonRef.current = L.geoJSON(feature, {
        style: {
          color: "#2563eb",
          fillColor: "#60a5fa",
          fillOpacity: 0.25,
          weight: 3,
        },
      }).addTo(map);
      setIsochroneGeometry(feature.geometry);
      setIsochroneAreaKm2(multiPolygonAreaKm2(feature.geometry));
      setIsochroneTransitUsed(feature.properties.transit_used);
      const bounds = polygonRef.current.getBounds();
      if (bounds.isValid()) {
        map.fitBounds(bounds, { padding: [30, 30] });
      }
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
    routeRequestRef.current += 1;
    const nextSelection = { ...routeSelection, [field]: place };
    setRouteSelection(nextSelection);
    setRoutes([]);
    setIsLoading(false);
    setError("");
    showRouteSelection(nextSelection);
  }

  function clearRoutePlace(field: "start" | "end") {
    routeRequestRef.current += 1;
    const nextSelection = { ...routeSelection };
    delete nextSelection[field];
    setRouteSelection(nextSelection);
    setRoutes([]);
    setIsLoading(false);
    setError("");
    showRouteSelection(nextSelection);
  }

  async function submitRoute() {
    if (!routeSelection.start || !routeSelection.end) {
      return;
    }

    const requestID = routeRequestRef.current + 1;
    routeRequestRef.current = requestID;
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
          departure_time: isochroneDepartureTime,
        }),
      });
      if (!response.ok) {
        throw new Error("Could not load routes from OneMap.");
      }

      const data = (await response.json()) as DirectionsResponse;
      if (requestID !== routeRequestRef.current) {
        return;
      }
      setRoutes(data.routes);
      setActiveRouteIndex(0);
      drawRoute(data.routes[0], routeSelection.start, routeSelection.end);
    } catch (err) {
      if (requestID !== routeRequestRef.current) {
        return;
      }
      setRoutes([]);
      setError(err instanceof Error ? err.message : "Could not load routes from OneMap.");
    } finally {
      if (requestID === routeRequestRef.current) {
        setIsLoading(false);
      }
    }
  }

  function showRouteSelection(selection: Partial<{ start: Place; end: Place }>) {
    const map = mapRef.current;
    clearMapLayers();
    if (!map) {
      return;
    }

    const points = [selection.start, selection.end].filter(
      (selected): selected is Place => selected !== undefined,
    );
    routeMarkersRef.current = points.map((selected) =>
      L.marker(placeToLatLng(selected)).addTo(map),
    );
    if (points.length === 1) {
      map.setView(placeToLatLng(points[0]), 15);
    } else if (points.length > 1) {
      map.fitBounds(L.latLngBounds(points.map(placeToLatLng)), { padding: [40, 40] });
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

    const routeLayer = L.featureGroup().addTo(map);
    const drawableSegments = (route.segments ?? []).filter(
      (segment) => segment.geometry.length >= 2,
    );

    if (drawableSegments.length > 0) {
      drawableSegments.forEach((segment) => {
        const mode = segment.mode.toUpperCase();
        const isWalking = mode === "WALK";
        L.polyline(
          segment.geometry.map((point) => [point.lat, point.lon] as L.LatLngExpression),
          {
            color: isWalking ? "#64748b" : transitRouteColor(mode),
            weight: isWalking ? 4 : 6,
            opacity: isWalking ? 0.8 : 0.92,
            dashArray: isWalking ? "7 7" : undefined,
          },
        ).addTo(routeLayer);
      });
    } else {
      L.polyline(
        route.geometry.map((point) => [point.lat, point.lon] as L.LatLngExpression),
        { color: "#2563eb", weight: 5, opacity: 0.85 },
      ).addTo(routeLayer);
    }
    routeRef.current = routeLayer;

    routeMarkersRef.current = [
      L.marker(placeToLatLng(start)).addTo(map),
      L.marker(placeToLatLng(end)).addTo(map),
    ];

    const bounds = routeLayer.getBounds();
    if (bounds.isValid()) {
      map.fitBounds(bounds, { padding: [40, 40] });
    }
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

  return (
    <main className="screen">
      <SearchBar
        mode={mode}
        routeSelection={routeSelection}
        isochronePlace={isochronePlace}
        searchProximity={searchProximity}
        isochroneAreaKm2={isochroneAreaKm2}
        isochroneMinutes={isochroneMinutes}
        isochroneDepartureTime={isochroneDepartureTime}
        isochroneTransitUsed={isochroneTransitUsed}
        routeResults={routes}
        activeRouteIndex={activeRouteIndex}
        isLoading={isLoading}
        error={error}
        onModeChange={changeMode}
        onIsochroneSelect={selectIsochronePlace}
        onIsochroneClear={clearIsochroneSelection}
        onIsochroneMinutesChange={changeIsochroneMinutes}
        onIsochroneDepartureTimeChange={changeIsochroneDepartureTime}
        onRoutePlaceChange={changeRoutePlace}
        onRoutePlaceClear={clearRoutePlace}
        onRouteSubmit={submitRoute}
        onRouteSelect={selectRoute}
      />
      <PresetOverlaySidebar
        activePresetIDs={activePresetIDs}
        isochroneGeometry={isochroneGeometry}
        showOnlyInsideIsochrone={showOnlyPresetsInsideIsochrone}
        onShowOnlyInsideIsochroneChange={setShowOnlyPresetsInsideIsochrone}
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

function transitRouteColor(mode: string): string {
  switch (mode) {
    case "BUS":
      return "#2563eb";
    case "SUBWAY":
    case "RAIL":
      return "#7c3aed";
    case "TRAM":
      return "#0891b2";
    case "FERRY":
      return "#0284c7";
    default:
      return "#2563eb";
  }
}

function escapeHTML(value: string) {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}

function upsertPreset(presets: PresetOverlay[], preset: PresetOverlay) {
  const withoutPreset = presets.filter((current) => current.id !== preset.id);
  return [...withoutPreset, preset];
}
