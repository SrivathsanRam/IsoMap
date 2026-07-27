import { useEffect, useRef, useState } from "react";
import { Bookmark, BookmarkCheck, CircleDot, History, Route } from "lucide-react";
import { addressToPlace, placeToAddressRequest } from "../lib/placeAdapters";
import { useRecentAddresses } from "../lib/useRecentAddresses";
import { useSavedAddresses } from "../lib/useSavedAddresses";

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
  const {
    recents,
    addRecent,
    error: recentError,
    isEnabled: canUseRecents,
  } = useRecentAddresses();
  const {
    saved,
    savePlace,
    isSaved,
    isSaving,
    error: savedError,
    isEnabled: canUseSaved,
  } = useSavedAddresses();

  const [selectedIsoPlace, setSelectedIsoPlace] = useState<Place | null>(null);
  const [activePanel, setActivePanel] = useState<"recents" | "saved" | null>(null);
  const selectedIsoSaved =
    selectedIsoPlace !== null &&
    isSaved(
      Number(selectedIsoPlace.lat),
      Number(selectedIsoPlace.lon),
      selectedIsoPlace.display_name,
    );

  function selectPlace(place: Place, queryText: string) {
    addRecent(placeToAddressRequest(place, queryText));

    if (mode === "isochrone") {
      setSelectedIsoPlace(place);
      onIsochroneSelect(place);
    } else {
      const field: keyof RouteSelection = routeSelection.start ? "end" : "start";
      onRoutePlaceChange(field, place);
    }

    setActivePanel(null);
  }

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

      <div className="panel-shortcuts">
        <button
          type="button"
          className={activePanel === "recents" ? "panel-shortcut active" : "panel-shortcut"}
          onClick={() => setActivePanel((panel) => (panel === "recents" ? null : "recents"))}
          title="Recent searches"
        >
          <History size={16} />
          Recent
        </button>
        <button
          type="button"
          className={activePanel === "saved" ? "panel-shortcut active" : "panel-shortcut"}
          onClick={() => setActivePanel((panel) => (panel === "saved" ? null : "saved"))}
          title="Saved places"
        >
          <Bookmark size={16} />
          Saved
        </button>
      </div>

      {activePanel === "recents" && (
        <section className="shortcut-panel" aria-label="Recent searches">
          <div className="shortcut-panel-header">
            <strong>Recent searches</strong>
            <span>{canUseRecents ? "Synced to your account" : "Sign in to sync"}</span>
          </div>
          {recentError && <p className="shortcut-error">{recentError}</p>}
          {!canUseRecents && (
            <p className="shortcut-empty">Sign in to save recent searches to the database.</p>
          )}
          {canUseRecents && recents.length === 0 && (
            <p className="shortcut-empty">No recent searches yet.</p>
          )}
          {recents.map((recent) => (
            <article key={recent.id} className="shortcut-item">
              <button
                type="button"
                onClick={() =>
                  selectPlace(addressToPlace(recent.address), recent.query_text)
                }
              >
                <History size={13} className="shortcut-icon" />
                <span>
                  <strong>{recent.address.formatted_address}</strong>
                  {recent.query_text && <small>Search: {recent.query_text}</small>}
                </span>
              </button>
            </article>
          ))}
        </section>
      )}

      {activePanel === "saved" && (
        <section className="shortcut-panel" aria-label="Saved places">
          <div className="shortcut-panel-header">
            <strong>Saved places</strong>
            <span>{canUseSaved ? "Stored in the database" : "Sign in to save"}</span>
          </div>
          {savedError && <p className="shortcut-error">{savedError}</p>}
          {!canUseSaved && (
            <p className="shortcut-empty">Sign in to save places to the database.</p>
          )}
          {canUseSaved && saved.length === 0 && (
            <p className="shortcut-empty">No saved places yet.</p>
          )}
          {saved.map((savedPlace) => (
            <article key={savedPlace.id} className="shortcut-item">
              <button
                type="button"
                onClick={() =>
                  selectPlace(
                    addressToPlace(savedPlace.address),
                    savedPlace.address.formatted_address,
                  )
                }
              >
                <Bookmark size={13} className="shortcut-icon" />
                <span>
                  <strong>{savedPlace.nickname || savedPlace.address.formatted_address}</strong>
                  {savedPlace.nickname && <small>{savedPlace.address.formatted_address}</small>}
                </span>
              </button>
            </article>
          ))}
        </section>
      )}

      {mode === "isochrone" ? (
        <>
          <PlaceInput
            placeholder="Search Singapore"
            onSelect={(place, queryText) => selectPlace(place, queryText)}
          />

          {selectedIsoPlace && (
            <div className="selected-place-row">
              <p className="selected-place-name">{selectedIsoPlace.display_name}</p>
              <button
                type="button"
                className="save-place-button"
                disabled={!canUseSaved || isSaving || selectedIsoSaved}
                onClick={() => savePlace(placeToAddressRequest(selectedIsoPlace))}
              >
                {selectedIsoSaved ? (
                  <>
                    <BookmarkCheck size={14} /> Saved
                  </>
                ) : (
                  <>
                    <Bookmark size={14} /> {isSaving ? "Saving..." : "Save place"}
                  </>
                )}
              </button>
            </div>
          )}
          {selectedIsoPlace && !canUseSaved && (
            <p className="map-panel-message">Sign in to save this place to the database.</p>
          )}

          <div className="isochrone-slider">
            <div className="isochrone-slider-header">
              <label htmlFor="isochrone-minutes">Travel time</label>
              <strong>{isochroneMinutes} min</strong>
            </div>
            <input
              id="isochrone-minutes"
              type="range"
              min="5"
              max="90"
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
              onSelect={(place, queryText) => selectPlace(place, queryText)}
            />
            <PlaceInput
              placeholder="End location"
              onSelect={(place, queryText) => selectPlace(place, queryText)}
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
  onSelect: (place: Place, queryText: string) => void;
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

    const submittedQuery = query.trim();
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
    onSelect(place, submittedQuery);
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
