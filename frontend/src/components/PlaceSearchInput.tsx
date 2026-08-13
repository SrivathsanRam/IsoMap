import {
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type FocusEvent,
  type KeyboardEvent,
} from "react";
import { LoaderCircle, LocateFixed, MapPin, Search, X } from "lucide-react";
import { searchPlaces } from "../lib/api";

export type Place = {
  display_name: string;
  lat: string;
  lon: string;
  primary_text?: string;
  secondary_text?: string;
  place_id?: string;
  source?: string;
  distance_meters?: number;
};

type Proximity = {
  lat: number;
  lon: number;
};

export type PlaceSearchInputProps = {
  placeholder: string;
  value?: Place | null;
  onSelect: (place: Place, queryText: string) => void;
  onFocus?: () => void;
  onClear?: () => void;
  proximity?: Proximity;
  id?: string;
  ariaLabel?: string;
};

type SearchStatus = "idle" | "loading" | "success" | "error";

const minimumQueryLength = 2;
const searchLimit = 8;
const debounceMilliseconds = 250;
const maximumCacheEntries = 40;
const singaporeBounds = {
  minLat: 1.1304,
  maxLat: 1.478,
  minLon: 103.5935,
  maxLon: 104.1076,
};

export function PlaceSearchInput({
  placeholder,
  value,
  onSelect,
  onFocus,
  onClear,
  proximity,
  id,
  ariaLabel,
}: PlaceSearchInputProps) {
  const generatedID = useId();
  const inputID = id ?? `place-search-${generatedID}`;
  const listboxID = `${inputID}-listbox`;
  const statusID = `${inputID}-status`;
  const rootRef = useRef<HTMLDivElement>(null);
  const cacheRef = useRef(new Map<string, Place[]>());
  const requestSequenceRef = useRef(0);
  const selectedTextRef = useRef(value?.display_name ?? "");
  const externalValueKeyRef = useRef(externalPlaceIdentity(value));

  const [query, setQuery] = useState(value?.display_name ?? "");
  const [places, setPlaces] = useState<Place[]>([]);
  const [status, setStatus] = useState<SearchStatus>("idle");
  const [errorMessage, setErrorMessage] = useState("");
  const [isFocused, setIsFocused] = useState(false);
  const [isOpen, setIsOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(-1);
  const [isLocating, setIsLocating] = useState(false);

  const externalValueKey = externalPlaceIdentity(value);
  useEffect(() => {
    if (externalValueKey === externalValueKeyRef.current) {
      return;
    }

    externalValueKeyRef.current = externalValueKey;
    const displayName = value?.display_name ?? "";
    selectedTextRef.current = displayName;
    setQuery(displayName);
    setPlaces([]);
    setStatus("idle");
    setErrorMessage("");
    setActiveIndex(-1);
  }, [externalValueKey, value?.display_name]);

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    function closeOnOutsidePointer(event: PointerEvent) {
      if (!rootRef.current?.contains(event.target as Node)) {
        setIsOpen(false);
        setActiveIndex(-1);
      }
    }

    document.addEventListener("pointerdown", closeOnOutsidePointer);
    return () => document.removeEventListener("pointerdown", closeOnOutsidePointer);
  }, [isOpen]);

  useEffect(() => {
    const trimmedQuery = query.trim();
    const coordinatePlace = parseCoordinatePlace(trimmedQuery);

    if (!isFocused || trimmedQuery === selectedTextRef.current) {
      return;
    }

    if (coordinatePlace) {
      return scheduleStateUpdate(() => {
        setPlaces([coordinatePlace]);
        setStatus("success");
        setErrorMessage("");
        setActiveIndex(0);
        setIsOpen(true);
      });
    }

    if (trimmedQuery.length < minimumQueryLength) {
      return scheduleStateUpdate(() => {
        setPlaces([]);
        setStatus("idle");
        setErrorMessage("");
        setActiveIndex(-1);
      });
    }

    const cacheKey = [
      trimmedQuery.toLocaleLowerCase("en-SG"),
      proximity ? `${proximity.lat.toFixed(3)},${proximity.lon.toFixed(3)}` : "",
    ].join("|");
    const cachedPlaces = cacheRef.current.get(cacheKey);
    if (cachedPlaces) {
      return scheduleStateUpdate(() => {
        setPlaces(cachedPlaces);
        setStatus("success");
        setErrorMessage("");
        setActiveIndex(cachedPlaces.length > 0 ? 0 : -1);
        setIsOpen(true);
      });
    }

    const controller = new AbortController();
    const requestSequence = requestSequenceRef.current + 1;
    requestSequenceRef.current = requestSequence;
    const timeout = window.setTimeout(async () => {
      setStatus("loading");
      setErrorMessage("");
      setIsOpen(true);

      try {
        const results = await searchPlaces(
          trimmedQuery,
          controller.signal,
          searchLimit,
          proximity,
        );
        if (requestSequence !== requestSequenceRef.current) {
          return;
        }

        const validPlaces = normalizePlaces(results);
        cacheResult(cacheRef.current, cacheKey, validPlaces);
        setPlaces(validPlaces);
        setStatus("success");
        setActiveIndex(validPlaces.length > 0 ? 0 : -1);
      } catch (error) {
        if (controller.signal.aborted || requestSequence !== requestSequenceRef.current) {
          return;
        }

        setPlaces([]);
        setStatus("error");
        setErrorMessage(
          error instanceof Error && error.message
            ? error.message
            : "Location search is unavailable. Please try again.",
        );
        setActiveIndex(-1);
      }
    }, debounceMilliseconds);

    return () => {
      controller.abort();
      window.clearTimeout(timeout);
    };
  }, [isFocused, proximity, query]);

  const displayedPlaces = useMemo(
    () => places.map((place) => addDistance(place, proximity)),
    [places, proximity],
  );
  const hasSearchableQuery =
    query.trim().length >= minimumQueryLength || parseCoordinatePlace(query.trim()) !== null;
  const showResults = isOpen && (hasSearchableQuery || errorMessage !== "");
  const activeOptionID =
    activeIndex >= 0 && activeIndex < displayedPlaces.length
      ? `${inputID}-option-${activeIndex}`
      : undefined;

  function changeQuery(event: ChangeEvent<HTMLInputElement>) {
    const nextQuery = event.target.value;
    selectedTextRef.current = "";
    requestSequenceRef.current += 1;
    setQuery(nextQuery);
    setErrorMessage("");
    setStatus("idle");
    setActiveIndex(-1);
    setIsOpen(true);
  }

  function focusInput() {
    setIsFocused(true);
    onFocus?.();
    if (hasSearchableQuery && query.trim() !== selectedTextRef.current) {
      setIsOpen(true);
    }
  }

  function blurInput(event: FocusEvent<HTMLInputElement>) {
    if (!rootRef.current?.contains(event.relatedTarget as Node | null)) {
      setIsFocused(false);
    }
  }

  function clearInput() {
    requestSequenceRef.current += 1;
    selectedTextRef.current = "";
    setQuery("");
    setPlaces([]);
    setStatus("idle");
    setErrorMessage("");
    setActiveIndex(-1);
    setIsOpen(false);
    onClear?.();
  }

  function useCurrentLocation() {
    if (!navigator.geolocation) {
      setStatus("error");
      setErrorMessage("Location access is not supported by this browser.");
      setIsOpen(true);
      return;
    }

    setIsLocating(true);
    setStatus("idle");
    setErrorMessage("");
    navigator.geolocation.getCurrentPosition(
      ({ coords }) => {
        setIsLocating(false);
        if (!isInSingapore(coords.latitude, coords.longitude)) {
          setStatus("error");
          setErrorMessage("Your current location is outside Singapore.");
          setIsOpen(true);
          return;
        }

        const latitudeText = formatCoordinate(coords.latitude);
        const longitudeText = formatCoordinate(coords.longitude);
        selectPlace({
          display_name: "Current location",
          primary_text: "Current location",
          secondary_text: `${latitudeText}, ${longitudeText}`,
          place_id: `geolocation:${latitudeText},${longitudeText}`,
          lat: String(coords.latitude),
          lon: String(coords.longitude),
          source: "geolocation",
        });
      },
      () => {
        setIsLocating(false);
        setStatus("error");
        setErrorMessage("Could not access your location. Check browser permissions and try again.");
        setIsOpen(true);
      },
      { enableHighAccuracy: true, timeout: 10_000, maximumAge: 60_000 },
    );
  }

  function selectPlace(place: Place) {
    const submittedQuery = query.trim();
    const selectedPlace = addDistance(place, proximity);
    selectedTextRef.current = selectedPlace.display_name;
    externalValueKeyRef.current = externalPlaceIdentity(selectedPlace);
    setQuery(selectedPlace.display_name);
    setPlaces([]);
    setStatus("idle");
    setErrorMessage("");
    setActiveIndex(-1);
    setIsOpen(false);
    onSelect(selectedPlace, submittedQuery);
  }

  function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === "Escape") {
      if (isOpen) {
        event.preventDefault();
        setIsOpen(false);
        setActiveIndex(-1);
      }
      return;
    }

    if (displayedPlaces.length === 0) {
      return;
    }

    if (event.key === "ArrowDown") {
      event.preventDefault();
      setIsOpen(true);
      setActiveIndex((current) => (current + 1) % displayedPlaces.length);
      return;
    }

    if (event.key === "ArrowUp") {
      event.preventDefault();
      setIsOpen(true);
      setActiveIndex((current) =>
        current <= 0 ? displayedPlaces.length - 1 : current - 1,
      );
      return;
    }

    if (event.key === "Enter" && isOpen) {
      const place = displayedPlaces[activeIndex >= 0 ? activeIndex : 0];
      if (place) {
        event.preventDefault();
        selectPlace(place);
      }
    }
  }

  return (
    <div className="place-input place-search-input" ref={rootRef}>
      <div className="place-search-field">
        <Search className="place-search-leading-icon" size={18} aria-hidden="true" />
        <input
          id={inputID}
          className="place-search-control"
          type="search"
          autoComplete="off"
          spellCheck="false"
          value={query}
          placeholder={placeholder}
          role="combobox"
          aria-label={ariaLabel ?? placeholder}
          aria-autocomplete="list"
          aria-controls={listboxID}
          aria-describedby={statusID}
          aria-expanded={showResults}
          aria-activedescendant={activeOptionID}
          onChange={changeQuery}
          onFocus={focusInput}
          onBlur={blurInput}
          onKeyDown={handleKeyDown}
        />
        <span className="place-search-actions">
          {(status === "loading" || isLocating) && (
            <LoaderCircle
              className="place-search-spinner"
              size={17}
              aria-label="Searching"
            />
          )}
          {!query && status !== "loading" && !isLocating && (
            <button
              type="button"
              className="place-search-locate"
              aria-label="Use current location"
              title="Use current location"
              onClick={useCurrentLocation}
            >
              <LocateFixed size={17} aria-hidden="true" />
            </button>
          )}
          {query && status !== "loading" && !isLocating && (
            <button
              type="button"
              className="place-search-clear"
              aria-label="Clear location search"
              onClick={clearInput}
            >
              <X size={17} aria-hidden="true" />
            </button>
          )}
        </span>
      </div>

      {showResults && (
        <ul id={listboxID} className="place-search-results" role="listbox">
          {displayedPlaces.map((place, index) => {
            const labels = placeLabels(place);
            return (
              <li
                id={`${inputID}-option-${index}`}
                key={placeIdentity(place) || `${place.display_name}-${index}`}
                role="option"
                aria-selected={activeIndex === index}
                className={activeIndex === index ? "active" : undefined}
                onMouseDown={(event) => event.preventDefault()}
                onMouseEnter={() => setActiveIndex(index)}
                onClick={() => selectPlace(place)}
              >
                <MapPin className="place-search-result-icon" size={17} aria-hidden="true" />
                <span className="place-search-result-labels">
                  <strong>{labels.primary}</strong>
                  {labels.secondary && <span>{labels.secondary}</span>}
                </span>
                {Number.isFinite(place.distance_meters) && (
                  <small className="place-search-distance">
                    {formatDistance(place.distance_meters as number)}
                  </small>
                )}
              </li>
            );
          })}

          {status === "loading" && displayedPlaces.length === 0 && (
            <li className="place-search-state" role="presentation">
              Searching Singapore...
            </li>
          )}
          {status === "success" && displayedPlaces.length === 0 && (
            <li className="place-search-state" role="presentation">
              No places found. Try a place name, address, postal code, or coordinates.
            </li>
          )}
          {status === "error" && (
            <li className="place-search-state place-search-error" role="presentation">
              {errorMessage}
            </li>
          )}
        </ul>
      )}

      <span id={statusID} className="place-search-sr-status" role="status" aria-live="polite">
        {screenReaderStatus(status, displayedPlaces.length)}
      </span>
    </div>
  );
}

function placeIdentity(place: Place | null | undefined) {
  if (!place) {
    return "";
  }
  return place.place_id || `${place.lat},${place.lon}|${place.display_name}`;
}

function externalPlaceIdentity(place: Place | null | undefined) {
  if (!place) {
    return "";
  }
  return `${placeIdentity(place)}|${place.display_name}`;
}

function normalizePlaces(results: Place[]) {
  const seen = new Set<string>();
  const places: Place[] = [];

  for (const place of results) {
    const latitude = Number(place.lat);
    const longitude = Number(place.lon);
    if (
      !place.display_name?.trim() ||
      !Number.isFinite(latitude) ||
      !Number.isFinite(longitude) ||
      latitude < -90 ||
      latitude > 90 ||
      longitude < -180 ||
      longitude > 180
    ) {
      continue;
    }

    const normalizedPlace = {
      ...place,
      display_name: place.display_name.trim(),
      lat: String(latitude),
      lon: String(longitude),
    };
    const key = placeIdentity(normalizedPlace);
    if (!seen.has(key)) {
      seen.add(key);
      places.push(normalizedPlace);
    }
  }

  return places.slice(0, searchLimit);
}

function cacheResult(cache: Map<string, Place[]>, key: string, places: Place[]) {
  if (cache.size >= maximumCacheEntries) {
    const oldestKey = cache.keys().next().value;
    if (oldestKey) {
      cache.delete(oldestKey);
    }
  }
  cache.set(key, places);
}

function scheduleStateUpdate(update: () => void) {
  const timeout = window.setTimeout(update, 0);
  return () => window.clearTimeout(timeout);
}

function parseCoordinatePlace(query: string): Place | null {
  const coordinateMatch = query.match(
    /^\s*\(?\s*([+-]?\d{1,3}(?:\.\d+)?)\s*[,\s]\s*([+-]?\d{1,3}(?:\.\d+)?)\s*\)?\s*$/,
  );
  if (!coordinateMatch) {
    return null;
  }

  let latitude = Number(coordinateMatch[1]);
  let longitude = Number(coordinateMatch[2]);
  if (Math.abs(latitude) > 90 && Math.abs(longitude) <= 90) {
    [latitude, longitude] = [longitude, latitude];
  }
  if (
    !Number.isFinite(latitude) ||
    !Number.isFinite(longitude) ||
    Math.abs(latitude) > 90 ||
    Math.abs(longitude) > 180 ||
    !isInSingapore(latitude, longitude)
  ) {
    return null;
  }

  const latitudeText = formatCoordinate(latitude);
  const longitudeText = formatCoordinate(longitude);
  return {
    display_name: `${latitudeText}, ${longitudeText}`,
    primary_text: "Use these coordinates",
    secondary_text: `${latitudeText}, ${longitudeText}`,
    place_id: `coordinates:${latitudeText},${longitudeText}`,
    lat: String(latitude),
    lon: String(longitude),
    source: "coordinates",
  };
}

function formatCoordinate(coordinate: number) {
  return coordinate.toFixed(6).replace(/\.?0+$/, "");
}

function isInSingapore(latitude: number, longitude: number) {
  return latitude >= singaporeBounds.minLat &&
    latitude <= singaporeBounds.maxLat &&
    longitude >= singaporeBounds.minLon &&
    longitude <= singaporeBounds.maxLon;
}

function addDistance(place: Place, proximity?: Proximity): Place {
  if (Number.isFinite(place.distance_meters) || !proximity) {
    return place;
  }

  const latitude = Number(place.lat);
  const longitude = Number(place.lon);
  if (!Number.isFinite(latitude) || !Number.isFinite(longitude)) {
    return place;
  }

  return {
    ...place,
    distance_meters: haversineMeters(proximity.lat, proximity.lon, latitude, longitude),
  };
}

function haversineMeters(latA: number, lonA: number, latB: number, lonB: number) {
  const earthRadiusMeters = 6_371_008.8;
  const latitudeDelta = toRadians(latB - latA);
  const longitudeDelta = toRadians(lonB - lonA);
  const startLatitude = toRadians(latA);
  const endLatitude = toRadians(latB);
  const haversine =
    Math.sin(latitudeDelta / 2) ** 2 +
    Math.cos(startLatitude) * Math.cos(endLatitude) *
      Math.sin(longitudeDelta / 2) ** 2;
  return 2 * earthRadiusMeters * Math.asin(Math.min(1, Math.sqrt(haversine)));
}

function toRadians(value: number) {
  return (value * Math.PI) / 180;
}

function placeLabels(place: Place) {
  const primary = place.primary_text?.trim() || place.display_name.split(",")[0].trim();
  const candidateSecondary = place.secondary_text?.trim() || place.display_name.trim();
  return {
    primary,
    secondary: candidateSecondary === primary ? "" : candidateSecondary,
  };
}

function formatDistance(meters: number) {
  if (meters < 1_000) {
    return `${Math.max(1, Math.round(meters / 10) * 10)} m`;
  }
  return `${(meters / 1_000).toFixed(meters < 10_000 ? 1 : 0)} km`;
}

function screenReaderStatus(status: SearchStatus, resultCount: number) {
  if (status === "loading") {
    return "Searching for places.";
  }
  if (status === "error") {
    return "Location search failed.";
  }
  if (status === "success") {
    return resultCount === 0
      ? "No places found."
      : `${resultCount} ${resultCount === 1 ? "place" : "places"} found. Use arrow keys to review.`;
  }
  return "";
}
