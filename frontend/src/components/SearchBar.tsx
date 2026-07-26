import { useEffect, useState } from "react";
import { CircleDot, Route, History, Bookmark, BookmarkCheck } from "lucide-react";
import { useRecentAddresses } from "../lib/useRecentAddresses";
import { useSavedAddresses } from "../lib/useSavedAddresses";
import { placeToAddressRequest, addressToPlace } from "../lib/placeAdapters";

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
  isLoading: boolean;
  error: string;
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
  isLoading,
  error,
  onModeChange,
  onIsochroneSelect,
  onRoutePlaceChange,
  onRouteSubmit,
  onRouteSelect,
}: SearchBarProps) {
  const { recents, addRecent } = useRecentAddresses();
  const { saved, savePlace, isSaved } = useSavedAddresses();

  const [selectedIsoPlace, setSelectedIsoPlace] = useState<Place | null>(null);
  const [activePanel, setActivePanel] = useState<"recents" | "saved" | null>(null);

  // Central place-selection router: sends the pick to the right handler
  // depending on mode, and to whichever routing field still needs filling.
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

      {/* Recents / Saved shortcuts */}
      <div className="panel-shortcuts">
        <button
          type="button"
          className={activePanel === "recents" ? "panel-shortcut active" : "panel-shortcut"}
          onClick={() => setActivePanel((p) => (p === "recents" ? null : "recents"))}
          title="Recent searches"
        >
          <History size={16} />
          Recent
        </button>
        <button
          type="button"
          className={activePanel === "saved" ? "panel-shortcut active" : "panel-shortcut"}
          onClick={() => setActivePanel((p) => (p === "saved" ? null : "saved"))}
          title="Saved places"
        >
          <Bookmark size={16} />
          Saved
        </button>
      </div>

      {activePanel === "recents" && (
        <ul className="shortcut-panel">
          {recents.length === 0 && <li className="shortcut-empty">No recent searches yet</li>}
          {recents.map((r) => (
            <li key={r.id}>
              <button
                type="button"
                onClick={() => selectPlace(addressToPlace(r.address), r.query_text)}
              >
                <History size={13} className="shortcut-icon" />
                {r.address.formatted_address}
              </button>
            </li>
          ))}
        </ul>
      )}

      {activePanel === "saved" && (
        <ul className="shortcut-panel">
          {saved.length === 0 && <li className="shortcut-empty">No saved places yet</li>}
          {saved.map((s) => (
            <li key={s.id}>
              <button
                type="button"
                onClick={() => selectPlace(addressToPlace(s.address), s.address.formatted_address)}
              >
                <Bookmark size={13} className="shortcut-icon" />
                {s.nickname ? `${s.nickname} — ${s.address.formatted_address}` : s.address.formatted_address}
              </button>
            </li>
          ))}
        </ul>
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
                onClick={() => savePlace(placeToAddressRequest(selectedIsoPlace))}
              >
                {isSaved(
                  parseFloat(selectedIsoPlace.lat),
                  parseFloat(selectedIsoPlace.lon),
                  selectedIsoPlace.display_name
                ) ? (
                  <>
                    <BookmarkCheck size={14} /> Saved
                  </>
                ) : (
                  <>
                    <Bookmark size={14} /> Save place
                  </>
                )}
              </button>
            </div>
          )}

          {isLoading && <p className="map-panel-message">Loading isochrone...</p>}
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

function PlaceInput({
  placeholder,
  onSelect,
}: {
  placeholder: string;
  onSelect: (place: Place, queryText: string) => void;
}) {
  const [query, setQuery] = useState("");
  const [places, setPlaces] = useState<Place[]>([]);

  useEffect(() => {
    if (query.trim().length < 2) {
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
  }, [query]);

  function changeQuery(value: string) {
    setQuery(value);
    if (value.trim().length < 2) {
      setPlaces([]);
    }
  }

  function select(place: Place) {
    setQuery(place.display_name);
    setPlaces([]);
    onSelect(place, query);
  }

  return (
    <div className="place-input">
      <input
        value={query}
        onChange={(event) => changeQuery(event.target.value)}
        placeholder={placeholder}
      />
      {places.length > 0 && (
        <ul>
          {places.map((place) => (
            <li key={`${place.lat}-${place.lon}-${place.display_name}`}>
              <button type="button" onMouseDown={() => select(place)}>
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