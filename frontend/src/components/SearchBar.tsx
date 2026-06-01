import { useEffect, useState } from "react";

export type Place = {
  display_name: string;
  lat: string;
  lon: string;
};

type SearchBarProps = {
  onSelect: (place: Place) => void;
};

export function SearchBar({ onSelect }: SearchBarProps) {
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
          limit: "10",
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
    onSelect(place);
  }

  return (
    <div className="map-search">
      <input
        value={query}
        onChange={(event) => changeQuery(event.target.value)}
        placeholder="Search Singapore"
      />
      {places.length > 0 && (
        <ul>
          {places.map((place) => (
            <li key={`${place.lat}-${place.lon}`}>
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
