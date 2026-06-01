import { useEffect, useRef } from "react";
import L from "leaflet";
import "leaflet/dist/leaflet.css";
import markerIcon from "leaflet/dist/images/marker-icon.png";
import markerIcon2x from "leaflet/dist/images/marker-icon-2x.png";
import markerShadow from "leaflet/dist/images/marker-shadow.png";
import { SearchBar, type Place } from "./SearchBar";

const api = import.meta.env.VITE_API_URL ?? "http://localhost:8000";
const singapore: L.LatLngExpression = [1.3521, 103.8198];

L.Icon.Default.mergeOptions({
  iconRetinaUrl: markerIcon2x,
  iconUrl: markerIcon,
  shadowUrl: markerShadow,
});

type Point = {
  lat: number;
  lon: number;
};

export function Map() {
  const elementRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<L.Map | null>(null);
  const markerRef = useRef<L.Marker | null>(null);
  const polygonRef = useRef<L.Polygon | null>(null);

  useEffect(() => {
    if (!elementRef.current || mapRef.current) {
      return;
    }

    const map = L.map(elementRef.current).setView(singapore, 12);
    L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
      attribution: "&copy; OpenStreetMap contributors",
    }).addTo(map);
    mapRef.current = map;

    return () => {
      map.remove();
      mapRef.current = null;
    };
  }, []);

  async function select(place: Place) {
    const lat = Number(place.lat);
    const lon = Number(place.lon);
    const map = mapRef.current;
    if (!map) {
      return;
    }

    map.setView([lat, lon], 15);
    markerRef.current?.remove();
    markerRef.current = L.marker([lat, lon]).addTo(map);

    const response = await fetch(`${api}/isochrone`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ lat, lon }),
    });
    if (!response.ok) {
      return;
    }
    const points = (await response.json()) as Point[];
    const polygon = points.map((point) => [point.lat, point.lon] as L.LatLngExpression);

    polygonRef.current?.remove();
    polygonRef.current = L.polygon(polygon, {
      color: "#2563eb",
      fillColor: "#60a5fa",
      fillOpacity: 0.25,
    }).addTo(map);
    map.fitBounds(polygonRef.current.getBounds());
  }

  return (
    <main className="screen">
      <SearchBar onSelect={select} />
      <div ref={elementRef} className="map" />
    </main>
  );
}
