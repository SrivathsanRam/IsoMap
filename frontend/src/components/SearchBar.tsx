import { useEffect, useState } from "react";
import { CircleDot, Route } from "lucide-react";

export type Place = {
  display_name: string;
  lat: string;
  lon: string;
};

export type MapToolMode = "isochrone" | "routing";

type RouteSelection = {
  start: Place;
  end: Place;
};

type SearchBarProps = {
  mode: MapToolMode;
  routeSelection: Partial<RouteSelection>;
  isochroneAreaKm2: number | null;
  routeResults: RouteSummary[];
  activeRouteIndex: number;
  onModeChange: (mode: MapToolMode) => void;
  onIsochroneSelect: (place: Place) => void;
  onRoutePlaceChange: (field: keyof RouteSelection, place: Place) => void;
  onRouteSubmit: () => void;
  onRouteSelect: (index: number) => void;
};

export type RouteSummary = {
  summary: string;
  duration_seconds: number;
  distance_meters: number;
  steps: {
    mode: string;
    instruction: string;
    duration_seconds: number;
    distance_meters: number;
  }[];
};

export function SearchBar({
  mode,
  routeSelection,
  isochroneAreaKm2,
  routeResults,
  activeRouteIndex,
  onModeChange,
  onIsochroneSelect,
  onRoutePlaceChange,
  onRouteSubmit,
  onRouteSelect,
}: SearchBarProps) {
  return (
    <aside className="map-panel">
      <div className="map-mode-toggle" aria-label="Map tool mode">
        <button
          type="button"
          className={mode === "isochrone" ? "active" : ""}
          onClick={() => onModeChange("isochrone")}
          title="Isochrone"
        >
          <CircleDot size={18} />
          Isochrone
        </button>
        <button
          type="button"
          className={mode === "routing" ? "active" : ""}
          onClick={() => onModeChange("routing")}
          title="Directions"
        >
          <Route size={18} />
          Routing
        </button>
      </div>

      {mode === "isochrone" ? (
        <>
          <PlaceInput placeholder="Search Singapore" onSelect={onIsochroneSelect} />
          {isochroneAreaKm2 !== null && (
            <div className="map-panel-section">
              <h3>Isochrone statistics</h3>
              <dl className="stat-list">
                <div>
                  <dt>Area</dt>
                  <dd>{formatArea(isochroneAreaKm2)}</dd>
                </div>
              </dl>
            </div>
          )}
        </>
      ) : (
        <>
          <div className="route-inputs">
            <PlaceInput
              placeholder="Start location"
              selectedPlace={routeSelection.start}
              onSelect={(place) => onRoutePlaceChange("start", place)}
            />
            <PlaceInput
              placeholder="End location"
              selectedPlace={routeSelection.end}
              onSelect={(place) => onRoutePlaceChange("end", place)}
            />
            <button
              type="button"
              className="route-submit"
              disabled={!routeSelection.start || !routeSelection.end}
              onClick={onRouteSubmit}
            >
              Get routes
            </button>
          </div>

          {routeResults.length > 0 && (
            <div className="map-panel-section route-results">
              <h3>Routes</h3>
              {routeResults.map((route, index) => (
                <button
                  type="button"
                  key={`${route.summary}-${index}`}
                  className={index === activeRouteIndex ? "route-card active" : "route-card"}
                  onClick={() => onRouteSelect(index)}
                >
                  <div className="route-card-header">
                    <strong>{formatDuration(route.duration_seconds)}</strong>
                    <span>{formatDistance(route.distance_meters)}</span>
                  </div>
                  <p>{route.summary}</p>
                  <ol>
                    {route.steps.slice(0, 4).map((step, stepIndex) => (
                      <li key={`${step.instruction}-${stepIndex}`}>{step.instruction}</li>
                    ))}
                  </ol>
                </button>
              ))}
            </div>
          )}
        </>
      )}
    </aside>
  );
}

function PlaceInput({
  placeholder,
  selectedPlace,
  onSelect,
}: {
  placeholder: string;
  selectedPlace?: Place;
  onSelect: (place: Place) => void;
}) {
  const [query, setQuery] = useState(selectedPlace?.display_name ?? "");
  const [places, setPlaces] = useState<Place[]>([]);

  useEffect(() => {
    setQuery(selectedPlace?.display_name ?? "");
  }, [selectedPlace]);

  useEffect(() => {
    if (query.trim().length < 2 || query === selectedPlace?.display_name) {
      setPlaces([]);
      return;
    }

    const controller = new AbortController();
    const timeout = window.setTimeout(async () => {
      try {
        const params = new URLSearchParams({
          q: query,
          format: "jsonv2",
          "accept-language": "en",
          bounded: "1",
          countrycodes: "sg",
          limit: "8",
          viewbox: "103.5935,1.4756,104.1076,1.1304",
        });
        const response = await fetch(`https://nominatim.openstreetmap.org/search?${params}`, {
          signal: controller.signal,
        });
        setPlaces(await response.json());
      } catch {
        if (!controller.signal.aborted) {
          setPlaces([]);
        }
      }
    }, 300);

    return () => {
      controller.abort();
      window.clearTimeout(timeout);
    };
  }, [query, selectedPlace]);

  function select(place: Place) {
    setQuery(place.display_name);
    setPlaces([]);
    onSelect(place);
  }

  return (
    <div className="place-input">
      <input
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        placeholder={placeholder}
      />
      {places.length > 0 && (
        <ul>
          {places.map((place) => (
            <li key={`${place.lat}-${place.lon}-${place.display_name}`}>
              <button type="button" onClick={() => select(place)}>
                {place.display_name}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function formatDuration(seconds: number) {
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) {
    return `${minutes} min`;
  }
  const hours = Math.floor(minutes / 60);
  const remaining = minutes % 60;
  return remaining === 0 ? `${hours} hr` : `${hours} hr ${remaining} min`;
}

function formatDistance(meters: number) {
  if (meters >= 1000) {
    return `${(meters / 1000).toFixed(1)} km`;
  }
  return `${Math.round(meters)} m`;
}

function formatArea(areaKm2: number) {
  if (areaKm2 < 1) {
    return `${Math.round(areaKm2 * 1_000_000).toLocaleString()} m²`;
  }
  return `${areaKm2.toFixed(2)} km²`;
}
