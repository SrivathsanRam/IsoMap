import { useEffect, useRef, useState } from "react";
import { CircleDot, Route } from "lucide-react";

const mapboxAccessToken =
  import.meta.env.MAPBOX_ACCESS_TOKEN ?? import.meta.env.VITE_MAPBOX_ACCESS_TOKEN;
const mapboxSearchboxURL = "https://api.mapbox.com/search/searchbox/v1";
const singaporeProximity = "103.8198,1.3521";

export type Place = {
  display_name: string;
  lat: string;
  lon: string;
};

type MapboxSuggestion = {
  name: string;
  name_preferred?: string;
  mapbox_id: string;
  full_address?: string;
  place_formatted?: string;
};

type MapboxSuggestResponse = {
  suggestions: MapboxSuggestion[];
};

type MapboxRetrieveResponse = {
  features: {
    geometry: {
      coordinates: [number, number];
    };
    properties: {
      name: string;
      full_address?: string;
      place_formatted?: string;
    };
  }[];
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
  isochroneMinutes: number;
  routeResults: RouteSummary[];
  activeRouteIndex: number;
  isLoading: boolean;
  error: string;
  onModeChange: (mode: MapToolMode) => void;
  onIsochroneSelect: (place: Place) => void;
  onIsochroneMinutesChange: (minutes: number) => void;
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
  isochroneMinutes,
  routeResults,
  activeRouteIndex,
  isLoading,
  error,
  onModeChange,
  onIsochroneSelect,
  onIsochroneMinutesChange,
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
          <div className="isochrone-slider">
            <div className="isochrone-slider-header">
              <label htmlFor="isochrone-minutes">Travel time</label>
              <strong>{isochroneMinutes} min</strong>
            </div>
            <input
              id="isochrone-minutes"
              type="range"
              min="5"
              max="60"
              step="5"
              value={isochroneMinutes}
              onChange={(event) => onIsochroneMinutesChange(Number(event.target.value))}
            />
            <div className="isochrone-slider-scale" aria-hidden="true">
              <span>5</span>
              <span>45</span>
              <span>90</span>
            </div>
          </div>
          {isLoading && <p className="map-panel-message">Loading isochrone...</p>}
          {error && <p className="map-panel-error">{error}</p>}
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
              onSelect={(place) => onRoutePlaceChange("start", place)}
            />
            <PlaceInput
              placeholder="End location"
              onSelect={(place) => onRoutePlaceChange("end", place)}
            />
            <button
              type="button"
              className="route-submit"
              disabled={!routeSelection.start || !routeSelection.end}
              onClick={onRouteSubmit}
            >
              {isLoading ? "Loading routes..." : "Get routes"}
            </button>
          </div>

          {error && <p className="map-panel-error">{error}</p>}

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

export function PlaceInput({
  placeholder,
  onSelect,
}: {
  placeholder: string;
  onSelect: (place: Place) => void;
}) {
  const [query, setQuery] = useState("");
  const [suggestions, setSuggestions] = useState<MapboxSuggestion[]>([]);
  const sessionTokenRef = useRef(newSessionToken());
  const selectedQueryRef = useRef("");

  useEffect(() => {
    if (query.trim().length < 2) {
      return;
    }
    if (query === selectedQueryRef.current) {
      return;
    }
    if (!mapboxAccessToken) {
      return;
    }

    const controller = new AbortController();
    const timeout = window.setTimeout(async () => {
      try {
        const params = new URLSearchParams({
          q: query.trim(),
          session_token: sessionTokenRef.current,
          proximity: singaporeProximity,
          country: "SG",
          language: "en",
          limit: "8",
          access_token: mapboxAccessToken,
        });
        const response = await fetch(`${mapboxSearchboxURL}/suggest?${params}`, {
          signal: controller.signal,
        });
        if (!response.ok) {
          throw new Error("Could not load suggestions.");
        }
        const data = (await response.json()) as MapboxSuggestResponse;
        setSuggestions(data.suggestions ?? []);
      } catch {
        if (!controller.signal.aborted) {
          setSuggestions([]);
        }
      }
    }, 300);

    return () => {
      controller.abort();
      window.clearTimeout(timeout);
    };
  }, [query]);

  function changeQuery(value: string) {
    selectedQueryRef.current = "";
    setQuery(value);
    if (value.trim().length < 2) {
      setSuggestions([]);
    }
  }

  async function select(suggestion: MapboxSuggestion) {
    if (!mapboxAccessToken) {
      return;
    }

    const params = new URLSearchParams({
      session_token: sessionTokenRef.current,
      access_token: mapboxAccessToken,
    });
    const response = await fetch(
      `${mapboxSearchboxURL}/retrieve/${encodeURIComponent(suggestion.mapbox_id)}?${params}`,
    );
    if (!response.ok) {
      setSuggestions([]);
      return;
    }

    const data = (await response.json()) as MapboxRetrieveResponse;
    const feature = data.features[0];
    if (!feature) {
      setSuggestions([]);
      return;
    }

    const [lon, lat] = feature.geometry.coordinates;
    const displayName = primarySuggestionLabel({
      name: feature.properties.name,
      full_address: feature.properties.full_address,
      place_formatted: feature.properties.place_formatted,
      mapbox_id: suggestion.mapbox_id,
    });
    const place = {
      display_name: displayName,
      lat: String(lat),
      lon: String(lon),
    };

    selectedQueryRef.current = displayName;
    setQuery(place.display_name);
    setSuggestions([]);
    sessionTokenRef.current = newSessionToken();
    onSelect(place);
  }

  return (
    <div className="place-input">
      <input
        value={query}
        onChange={(event) => changeQuery(event.target.value)}
        placeholder={placeholder}
      />
      {suggestions.length > 0 && (
        <ul>
          {suggestions.map((suggestion) => (
            <li key={suggestion.mapbox_id}>
              <button type="button" onClick={() => select(suggestion)}>
                <strong>{primarySuggestionLabel(suggestion)}</strong>
                {secondarySuggestionLabel(suggestion) && (
                  <span>{secondarySuggestionLabel(suggestion)}</span>
                )}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function newSessionToken() {
  return crypto.randomUUID();
}

function primarySuggestionLabel(suggestion: MapboxSuggestion) {
  return suggestion.name_preferred || suggestion.name;
}

function secondarySuggestionLabel(suggestion: MapboxSuggestion) {
  const primary = primarySuggestionLabel(suggestion);
  const secondary = suggestion.full_address || suggestion.place_formatted;
  if (!secondary || secondary === primary) {
    return "";
  }
  return secondary;
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
