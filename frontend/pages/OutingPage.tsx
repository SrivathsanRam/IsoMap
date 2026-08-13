import { useEffect, useMemo, useRef, useState } from "react"
import { useParams } from "react-router-dom"
import L from "leaflet"
import "leaflet/dist/leaflet.css"

import {
  DEFAULT_ISOCHRONE_DEPARTURE_TIME,
  toSingaporeDepartureRFC3339,
} from "../src/lib/departureTime"
import { TimeOfDayControl } from "../src/components/TimeOfDayControl"
import { PlaceSearchInput, type Place } from "../src/components/PlaceSearchInput"
import { intersectMultiPolygons, multiPolygonAreaKm2 } from "../src/lib/geojson"
import {
  getIsochrone,
  getOuting,
  joinOuting,
  updateOutingMember,
  type Outing,
  type OutingMember,
  type MultiPolygonGeometry,
} from "../src/lib/api"

type MemberPolygon = {
  memberId: string
  geometry: MultiPolygonGeometry
}

const singapore: L.LatLngExpression = [1.3521, 103.8198]
const mapboxAccessToken =
  import.meta.env.MAPBOX_ACCESS_TOKEN ?? import.meta.env.VITE_MAPBOX_ACCESS_TOKEN

export default function OutingPage() {
  const { token = "" } = useParams()
  const mapElementRef = useRef<HTMLDivElement>(null)
  const mapRef = useRef<L.Map | null>(null)
  const layerGroupRef = useRef<L.LayerGroup | null>(null)
  const locationMarkerRef = useRef<L.CircleMarker | null>(null)

  const [outing, setOuting] = useState<Outing | null>(null)
  const [currentMember, setCurrentMember] = useState<OutingMember | null>(null)
  const [displayName, setDisplayName] = useState("")
  const [selectedPlace, setSelectedPlace] = useState<Place | null>(null)
  const [minutes, setMinutes] = useState(15)
  const [departureTime, setDepartureTime] = useState(DEFAULT_ISOCHRONE_DEPARTURE_TIME)
  const [searchProximity, setSearchProximity] = useState({ lat: 1.3521, lon: 103.8198 })
  const [polygons, setPolygons] = useState<MemberPolygon[]>([])
  const [overlapGeometry, setOverlapGeometry] = useState<MultiPolygonGeometry | null>(null)
  const [overlapMessage, setOverlapMessage] = useState("")
  const [error, setError] = useState("")
  const [isLoading, setIsLoading] = useState(false)

  const joinUrl = `${window.location.origin}/outings/${token}`

  useEffect(() => {
    if (!mapElementRef.current || mapRef.current) {
      return
    }

    const map = L.map(mapElementRef.current).setView(singapore, 12)
    if (mapboxAccessToken) {
      L.tileLayer(
        `https://api.mapbox.com/styles/v1/mapbox/streets-v12/tiles/{z}/{x}/{y}?access_token=${mapboxAccessToken}`,
        {
          attribution:
            '&copy; <a href="https://www.mapbox.com/about/maps/">Mapbox</a> &copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>',
          maxZoom: 19,
          tileSize: 512,
          zoomOffset: -1,
        },
      ).addTo(map)
    } else {
      L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
        attribution: "&copy; OpenStreetMap contributors",
      }).addTo(map)
    }
    layerGroupRef.current = L.layerGroup().addTo(map)
    mapRef.current = map

    const updateSearchProximity = () => {
      const center = map.getCenter()
      setSearchProximity({ lat: center.lat, lon: center.lng })
    }
    map.on("moveend", updateSearchProximity)

    return () => {
      map.off("moveend", updateSearchProximity)
      map.remove()
      mapRef.current = null
      layerGroupRef.current = null
      locationMarkerRef.current = null
    }
  }, [])

  useEffect(() => {
    loadOuting()
  }, [token])

  useEffect(() => {
    drawPolygons()
  }, [overlapGeometry, polygons])

  const membersWithLocations = useMemo(
    () => outing?.members.filter(hasMemberLocation) ?? [],
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
    setIsLoading(true)
    setError("")
    try {
      const member = await joinOuting(token, displayName.trim())
      setCurrentMember(member)
      setMinutes(member.max_travel_minutes)
      await loadOuting()
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not join outing")
    } finally {
      setIsLoading(false)
    }
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
      clearGeneratedAreas()
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save location")
    } finally {
      setIsLoading(false)
    }
  }

  function selectLocation(place: Place) {
    setSelectedPlace(place)
    const map = mapRef.current
    if (!map) {
      return
    }
    const position: L.LatLngExpression = [Number(place.lat), Number(place.lon)]
    locationMarkerRef.current?.remove()
    locationMarkerRef.current = L.circleMarker(position, {
      radius: 8,
      color: "#1d4ed8",
      fillColor: "#3b82f6",
      fillOpacity: 0.9,
      weight: 3,
    }).addTo(map)
    map.setView(position, 16)
  }

  function clearLocation() {
    setSelectedPlace(null)
    locationMarkerRef.current?.remove()
    locationMarkerRef.current = null
  }

  async function generateOverlap() {
    if (!token) {
      return
    }

    setIsLoading(true)
    setError("")
    setOverlapMessage("")
    try {
      // Refresh immediately before computing so newly joined members and their
      // latest travel limits are included without requiring a page reload.
      const latestOuting = await getOuting(token)
      setOuting(latestOuting)
      const members = latestOuting.members.filter(hasMemberLocation)
      if (members.length < 2) {
        setPolygons([])
        setOverlapGeometry(null)
        setError("At least two members with locations are needed.")
        return
      }
      const nextPolygons = await Promise.all(
        members.map(async (member) => ({
          memberId: member.id,
          geometry: (await getIsochrone(
            { lat: Number(member.latitude), lon: Number(member.longitude) },
            member.max_travel_minutes,
            toSingaporeDepartureRFC3339(departureTime),
          )).geometry,
        })),
      )
      const overlap = intersectMultiPolygons(nextPolygons.map((entry) => entry.geometry))
      setPolygons(nextPolygons)
      setOverlapGeometry(overlap)
      if (overlap) {
        const regionCount = overlap.coordinates.length
        setOverlapMessage(
          `${regionCount} common ${regionCount === 1 ? "area" : "areas"} · ${formatArea(
            multiPolygonAreaKm2(overlap),
          )}`,
        )
      } else {
        setOverlapMessage("There is no area reachable by every member at this time.")
      }
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
	  const layer = L.geoJSON(entry.geometry, {
		style: {
		  color: memberColor(index),
		  fillColor: memberColor(index),
		  fillOpacity: 0.08,
		  weight: 2,
		},
	  }).addTo(layerGroup)
	  bounds.extend(layer.getBounds())
    })

    if (overlapGeometry) {
	  const overlapLayer = L.geoJSON(overlapGeometry, {
		style: {
		  color: "#166534",
		  fillColor: "#22c55e",
		  fillOpacity: 0.5,
		  opacity: 1,
		  weight: 4,
		},
	  }).addTo(layerGroup)
	  overlapLayer.bringToFront()
	  bounds.extend(overlapLayer.getBounds())
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
          <div className="outing-time-control rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <TimeOfDayControl
              id="outing-departure-time"
              value={departureTime}
              onChange={(value) => {
                setDepartureTime(value)
                clearGeneratedAreas()
              }}
            />
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
                disabled={isLoading}
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
                <PlaceSearchInput
                  placeholder="Search your location"
                  value={selectedPlace}
                  proximity={searchProximity}
                  onClear={clearLocation}
                  onSelect={selectLocation}
                />
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

          {overlapMessage && (
            <p
              className={`rounded-lg px-3 py-2 text-sm ${
                overlapGeometry ? "bg-emerald-50 text-emerald-800" : "bg-amber-50 text-amber-800"
              }`}
              role="status"
            >
              {overlapMessage}
            </p>
          )}

          {error && <p className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>}
        </aside>

        <div className="min-h-[640px] overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
          <div ref={mapElementRef} className="h-[640px] w-full" />
        </div>
      </section>
    </main>
  )
  function clearGeneratedAreas() {
    setPolygons([])
    setOverlapGeometry(null)
    setOverlapMessage("")
  }
}

function hasMemberLocation(member: OutingMember): boolean {
  return Number.isFinite(member.latitude) && Number.isFinite(member.longitude)
}

function formatArea(areaKm2: number): string {
  if (areaKm2 < 0.01) {
    return `${Math.round(areaKm2 * 1_000_000).toLocaleString()} m²`
  }
  return `${areaKm2.toFixed(areaKm2 < 10 ? 2 : 1)} km²`
}

function memberColor(index: number) {
  const colors = ["#2563eb", "#7c3aed", "#ea580c", "#0891b2", "#be123c"]
  return colors[index % colors.length]
}
