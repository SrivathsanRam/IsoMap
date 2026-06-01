import { useState } from "react";
import { SearchBar, type Place } from "./SearchBar";

const singaporeMap =
  "https://www.openstreetmap.org/export/embed.html?bbox=103.5935%2C1.1304%2C104.1076%2C1.4756&layer=mapnik";

function mapUrl(place: Place) {
  const lat = Number(place.lat);
  const lon = Number(place.lon);
  const pad = 0.01;

  return `https://www.openstreetmap.org/export/embed.html?bbox=${lon - pad}%2C${lat - pad}%2C${lon + pad}%2C${lat + pad}&layer=mapnik&marker=${lat}%2C${lon}`;
}

export function Map() {
  const [src, setSrc] = useState(singaporeMap);

  function select(place: Place) {
    setSrc(mapUrl(place));
  }

  return (
    <main className="screen">
      <SearchBar onSelect={select} />
      <iframe title="OpenStreetMap Singapore" src={src} />
    </main>
  );
}
