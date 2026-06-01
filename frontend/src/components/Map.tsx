import { useEffect, useState } from "react";

type Place = {
  display_name: string;
  lat: string;
  lon: string;
};

const singaporeMap =
  "https://www.openstreetmap.org/export/embed.html?bbox=103.5935%2C1.1304%2C104.1076%2C1.4756&layer=mapnik";

function mapUrl(place: Place) {
  const lat = Number(place.lat);
  const lon = Number(place.lon);
  const pad = 0.01;

  return `https://www.openstreetmap.org/export/embed.html?bbox=${lon - pad}%2C${lat - pad}%2C${lon + pad}%2C${lat + pad}&layer=mapnik&marker=${lat}%2C${lon}`;
}

export function Map() {
  const [query, setQuery] = useState("");
  const [places, setPlaces] = useState<Place[]>([]);
  const [src, setSrc] = useState(singaporeMap);

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

  function select(place: Place) {
    setQuery(place.display_name);
    setPlaces([]);
    setSrc(mapUrl(place));
  }

  function changeQuery(value: string) {
    setQuery(value);
    if (value.trim().length < 2) {
      setPlaces([]);
    }
  }

  return (
    <main className="screen">
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
      <iframe title="OpenStreetMap Singapore" src={src} />
    </main>
  );
}
