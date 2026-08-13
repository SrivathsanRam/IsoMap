import { useState } from "react";
import { Bookmark, BookmarkCheck, CircleDot, History, Route } from "lucide-react";
import { addressToPlace, placeToAddressRequest } from "../lib/placeAdapters";
import { useRecentAddresses } from "../lib/useRecentAddresses";
import { useSavedAddresses } from "../lib/useSavedAddresses";
import { PlaceSearchInput, type Place } from "./PlaceSearchInput";
import { TimeOfDayControl } from "./TimeOfDayControl";

export { PlaceSearchInput as PlaceInput };
export type { Place };

export type MapToolMode = "isochrone" | "routing";

type RouteSelection = {
  start: Place;
  end: Place;
};

type SearchBarProps = {
  mode: MapToolMode;
  routeSelection: Partial<RouteSelection>;
  isochronePlace: Place | null;
  searchProximity: { lat: number; lon: number };
  isochroneAreaKm2: number | null;
  isochroneMinutes: number;
  isochroneDepartureTime: string;
  isochroneTransitUsed: boolean | null;
  routeResults: RouteSummary[];
  activeRouteIndex: number;
  isLoading: boolean;
  error: string;
  onModeChange: (mode: MapToolMode) => void;
  onIsochroneSelect: (place: Place) => void;
  onIsochroneClear: () => void;
  onIsochroneMinutesChange: (minutes: number) => void;
  onIsochroneDepartureTimeChange: (departureTime: string) => void;
  onRoutePlaceChange: (field: keyof RouteSelection, place: Place) => void;
  onRoutePlaceClear: (field: keyof RouteSelection) => void;
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
  isochronePlace,
  searchProximity,
  isochroneAreaKm2,
  isochroneMinutes,
  isochroneDepartureTime,
  isochroneTransitUsed,
  routeResults,
  activeRouteIndex,
  isLoading,
  error,
  onModeChange,
  onIsochroneSelect,
  onIsochroneClear,
  onIsochroneMinutesChange,
  onIsochroneDepartureTimeChange,
  onRoutePlaceChange,
  onRoutePlaceClear,
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

  const [activePanel, setActivePanel] = useState<"recents" | "saved" | null>(null);
  const [routeTargetField, setRouteTargetField] = useState<keyof RouteSelection>("start");
  const selectedIsoSaved =
    isochronePlace !== null &&
    isSaved(
      Number(isochronePlace.lat),
      Number(isochronePlace.lon),
      isochronePlace.display_name,
    );

  function selectPlace(
    place: Place,
    queryText: string,
    explicitRouteField?: keyof RouteSelection,
  ) {
    addRecent(placeToAddressRequest(place, queryText));

    if (mode === "isochrone") {
      onIsochroneSelect(place);
    } else {
      const field = explicitRouteField ?? routeTargetField;
      setRouteTargetField(field === "start" ? "end" : "start");
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
          <PlaceSearchInput
            placeholder="Search Singapore"
            value={isochronePlace}
            proximity={searchProximity}
            onClear={onIsochroneClear}
            onSelect={(place, queryText) => selectPlace(place, queryText)}
          />

          {isochronePlace && (
            <div className="selected-place-row">
              <p className="selected-place-name">{isochronePlace.display_name}</p>
              <button
                type="button"
                className="save-place-button"
                disabled={!canUseSaved || isSaving || selectedIsoSaved}
                onClick={() => savePlace(placeToAddressRequest(isochronePlace))}
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
          {isochronePlace && !canUseSaved && (
            <p className="map-panel-message">Sign in to save this place to the database.</p>
          )}

          <TimeOfDayControl
            id="isochrone-departure-time"
            value={isochroneDepartureTime}
            onChange={onIsochroneDepartureTimeChange}
          />

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
                <div>
                  <dt>Network</dt>
                  <dd>{isochroneTransitUsed ? "Transit + walking" : "Walking only"}</dd>
                </div>
              </dl>
            </div>
          )}
        </>
      ) : (
        <>
          <div className="route-inputs">
            <PlaceSearchInput
              placeholder="Start location"
              value={routeSelection.start ?? null}
              proximity={searchProximity}
              onFocus={() => setRouteTargetField("start")}
              onClear={() => onRoutePlaceClear("start")}
              onSelect={(place, queryText) => selectPlace(place, queryText, "start")}
            />
            <PlaceSearchInput
              placeholder="End location"
              value={routeSelection.end ?? null}
              proximity={searchProximity}
              onFocus={() => setRouteTargetField("end")}
              onClear={() => onRoutePlaceClear("end")}
              onSelect={(place, queryText) => selectPlace(place, queryText, "end")}
            />
            <TimeOfDayControl
              id="route-departure-time"
              value={isochroneDepartureTime}
              onChange={onIsochroneDepartureTimeChange}
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
