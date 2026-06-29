import { useEffect, useMemo, useRef, useState } from "react"
import { useParams } from "react-router-dom"
import L from "leaflet"
import "leaflet/dist/leaflet.css"
import { featureCollection, intersect, polygon } from "@turf/turf"

import {
  getIsochrone,
  getOuting,
  joinOuting,
  updateOutingMember,
  type Outing,
  type OutingMember,
  type Point,
} from "../src/lib/api"

type Place = {
  display_name: string
  lat: string
  lon: string
}

type MemberPolygon = {
  memberId: string
  points: Point[]
}

const singapore: L.LatLngExpression = [1.3521, 103.8198]

export default function OutingPage() {
  const { token = "" } = useParams()
  const mapElementRef = useRef<HTMLDivElement>(null)
  const mapRef = useRef<L.Map | null>(null)
  const layerGroupRef = useRef<L.LayerGroup | null>(null)

  const [outing, setOuting] = useState<Outing | null>(null)
  const [currentMember, setCurrentMember] = useState<OutingMember | null>(null)
  const [displayName, setDisplayName] = useState("")
  const [selectedPlace, setSelectedPlace] = useState<Place | null>(null)
  const [minutes, setMinutes] = useState(15)
  const [polygons, setPolygons] = useState<MemberPolygon[]>([])
  const [error, setError] = useState("")
  const [isLoading, setIsLoading] = useState(false)

  const joinUrl = `${window.location.origin}/outings/${token}`

  useEffect(() => {
    if (!mapElementRef.current || mapRef.current) {
      return
    }

    const map = L.map(mapElementRef.current).setView(singapore, 12)
    L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
      attribution: "&copy; OpenStreetMap contributors",
    }).addTo(map)
    layerGroupRef.current = L.layerGroup().addTo(map)
    mapRef.current = map

    return () => {
      map.remove()
      mapRef.current = null
      layerGroupRef.current = null
    }
  }, [])

  useEffect(() => {
    loadOuting()
  }, [token])

  useEffect(() => {
    drawPolygons()
  }, [polygons])

  const membersWithLocations = useMemo(
    () => outing?.members.filter((member) => member.latitude && member.longitude) ?? [],
    [outing],
  )

  async function loadOuting() {
    if (!token) {
      return
    }
    try {
      setOuting(await getOuting(token))
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not load outing")
    }
  }

  async function handleJoin() {
    if (!displayName.trim()) {
      setError("Enter your name to join.")
      return
    }
    setError("")
    const member = await joinOuting(token, displayName.trim())
    setCurrentMember(member)
    setMinutes(member.max_travel_minutes)
    await loadOuting()
  }

  async function saveLocation() {
    if (!currentMember || !selectedPlace) {
      setError("Join the outing and select your location first.")
      return
    }

    setIsLoading(true)
    setError("")
    try {
      await updateOutingMember(token, currentMember.id, {
        location_name: selectedPlace.display_name,
        latitude: Number(selectedPlace.lat),
        longitude: Number(selectedPlace.lon),
        max_travel_minutes: minutes,
      })
      await loadOuting()
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save location")
    } finally {
      setIsLoading(false)
    }
  }

  async function generateOverlap() {
    if (!outing) {
      return
    }
    const members = outing.members.filter((member) => member.latitude && member.longitude)
    if (members.length < 2) {
      setError("At least two members with locations are needed.")
      return
    }

    setIsLoading(true)
    setError("")
    try {
      const nextPolygons = await Promise.all(
        members.map(async (member) => ({
          memberId: member.id,
          points: await getIsochrone(
            { lat: Number(member.latitude), lon: Number(member.longitude) },
            member.max_travel_minutes,
          ),
        })),
      )
      setPolygons(nextPolygons)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not generate overlap")
    } finally {
      setIsLoading(false)
    }
  }

  function drawPolygons() {
    const map = mapRef.current
    const layerGroup = layerGroupRef.current
    if (!map || !layerGroup) {
      return
    }

    layerGroup.clearLayers()
    const bounds = L.latLngBounds([])

    polygons.forEach((entry, index) => {
      const latLngs = entry.points.map((point) => [point.lat, point.lon] as L.LatLngExpression)
      L.polygon(latLngs, {
        color: memberColor(index),
        fillColor: memberColor(index),
        fillOpacity: 0.12,
        weight: 2,
      }).addTo(layerGroup)
      latLngs.forEach((point) => bounds.extend(point))
    })

    const overlap = intersectAll(polygons)
    if (overlap) {
      const rings = overlap.geometry.coordinates
      rings.forEach((ring) => {
        const latLngs = ring.map(([lon, lat]) => [lat, lon] as L.LatLngExpression)
        L.polygon(latLngs, {
          color: "#111827",
          fillColor: "#22c55e",
          fillOpacity: 0.35,
          weight: 3,
        }).addTo(layerGroup)
        latLngs.forEach((point) => bounds.extend(point))
      })
    }

    if (bounds.isValid()) {
      map.fitBounds(bounds, { padding: [40, 40] })
    }
  }

  return (
    <main className="min-h-screen bg-slate-50">
      <section className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-6xl flex-col gap-3 px-6 py-5 md:flex-row md:items-center md:justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-slate-950">{outing?.title ?? "Group outing"}</h1>
            <p className="mt-1 text-sm text-slate-500">Share this link so others can join.</p>
          </div>
          <button
            type="button"
            onClick={() => navigator.clipboard.writeText(joinUrl)}
            className="rounded-lg border border-slate-200 bg-white px-4 py-2 text-sm text-slate-700 hover:bg-slate-50"
          >
            Copy join link
          </button>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-5 px-6 py-6 lg:grid-cols-[360px_1fr]">
        <aside className="space-y-4">
          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <h2 className="font-medium text-slate-950">Join outing</h2>
            <div className="mt-3 grid gap-2">
              <input
                value={displayName}
                onChange={(event) => setDisplayName(event.target.value)}
                placeholder="Your name"
                className="rounded-lg border border-slate-200 px-3 py-2 text-sm"
              />
              <button
                type="button"
                onClick={handleJoin}
                className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white hover:bg-slate-700"
              >
                Join
              </button>
            </div>
          </div>

          {currentMember && (
            <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
              <h2 className="font-medium text-slate-950">Your location</h2>
              <div className="mt-3 grid gap-3">
                <PlaceSearch onSelect={setSelectedPlace} />
                <label className="grid gap-2 text-sm text-slate-600">
                  Max travel time: <strong>{minutes} min</strong>
                  <input
                    type="range"
                    min="5"
                    max="60"
                    step="5"
                    value={minutes}
                    onChange={(event) => setMinutes(Number(event.target.value))}
                  />
                </label>
                <button
                  type="button"
                  onClick={saveLocation}
                  disabled={isLoading}
                  className="rounded-lg bg-blue-600 px-3 py-2 text-sm font-medium text-white hover:bg-blue-500 disabled:opacity-50"
                >
                  Save location
                </button>
              </div>
            </div>
          )}

          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <div className="flex items-center justify-between">
              <h2 className="font-medium text-slate-950">Members</h2>
              <span className="text-sm text-slate-500">{outing?.members.length ?? 0}</span>
            </div>
            <div className="mt-3 grid gap-2">
              {outing?.members.map((member) => (
                <div key={member.id} className="rounded-lg border border-slate-100 p-3">
                  <div className="font-medium text-slate-900">{member.display_name}</div>
                  <div className="mt-1 text-sm text-slate-500">
                    {member.location_name || "No location yet"} · {member.max_travel_minutes} min
                  </div>
                </div>
              ))}
            </div>
          </div>

          <button
            type="button"
            onClick={generateOverlap}
            disabled={isLoading || membersWithLocations.length < 2}
            className="w-full rounded-lg bg-slate-900 px-4 py-3 text-sm font-medium text-white hover:bg-slate-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {isLoading ? "Generating..." : "Find common area"}
          </button>

          {error && <p className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>}
        </aside>

        <div className="min-h-[640px] overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
          <div ref={mapElementRef} className="h-[640px] w-full" />
        </div>
      </section>
    </main>
  )
}

function PlaceSearch({ onSelect }: { onSelect: (place: Place) => void }) {
  const [query, setQuery] = useState("")
  const [places, setPlaces] = useState<Place[]>([])

  useEffect(() => {
    if (query.trim().length < 2) {
      setPlaces([])
      return
    }

    const controller = new AbortController()
    const timeout = window.setTimeout(async () => {
      const params = new URLSearchParams({
        q: query,
        format: "jsonv2",
        "accept-language": "en",
        bounded: "1",
        countrycodes: "sg",
        limit: "6",
        viewbox: "103.5935,1.4756,104.1076,1.1304",
      })
      const response = await fetch(`https://nominatim.openstreetmap.org/search?${params}`, {
        signal: controller.signal,
      })
      setPlaces(await response.json())
    }, 300)

    return () => {
      controller.abort()
      window.clearTimeout(timeout)
    }
  }, [query])

  function select(place: Place) {
    setQuery(place.display_name)
    setPlaces([])
    onSelect(place)
  }

  return (
    <div className="relative">
      <input
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        placeholder="Search your location"
        className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm"
      />
      {places.length > 0 && (
        <div className="absolute z-[1000] mt-1 max-h-56 w-full overflow-auto rounded-lg border border-slate-200 bg-white shadow-lg">
          {places.map((place) => (
            <button
              key={`${place.lat}-${place.lon}-${place.display_name}`}
              type="button"
              onClick={() => select(place)}
              className="block w-full px-3 py-2 text-left text-sm text-slate-700 hover:bg-slate-50"
            >
              {place.display_name}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

function intersectAll(polygons: MemberPolygon[]) {
  const features = polygons
    .map((entry) => {
      const ring = entry.points.map((point) => [point.lon, point.lat])
      if (ring.length > 0) {
        const first = ring[0]
        const last = ring[ring.length - 1]
        if (first[0] !== last[0] || first[1] !== last[1]) {
          ring.push(first)
        }
      }
      return polygon([ring])
    })

  if (features.length < 2) {
    return null
  }

  let overlap = intersect(featureCollection([features[0], features[1]]))
  for (let index = 2; overlap && index < features.length; index += 1) {
    overlap = intersect(featureCollection([overlap, features[index]]))
  }

  if (!overlap || overlap.geometry.type !== "Polygon") {
    return null
  }

  return overlap
}

function memberColor(index: number) {
  const colors = ["#2563eb", "#7c3aed", "#ea580c", "#0891b2", "#be123c"]
  return colors[index % colors.length]
}
