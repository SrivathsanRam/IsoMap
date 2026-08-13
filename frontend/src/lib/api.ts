const API_URL = import.meta.env.VITE_API_URL ?? "http://localhost:8000";
const mapboxAccessToken =
  import.meta.env.MAPBOX_ACCESS_TOKEN ?? import.meta.env.VITE_MAPBOX_ACCESS_TOKEN;
const mapboxGeocodingURL = "https://api.mapbox.com/search/geocode/v6/forward";

export type { IsochroneFeature, MultiPolygonGeometry } from "../../types";
import type { IsochroneFeature } from "../../types";

export type AddressPayload = {
  formatted_address: string;
  place_id?: string;
  latitude: number;
  longitude: number;
  query_text?: string;
  nickname?: string;
};

export type OutingMember = {
  id: string;
  outing_id: string;
  user_id?: string | null;
  display_name: string;
  location_name: string;
  latitude?: number | null;
  longitude?: number | null;
  max_travel_minutes: number;
  created_at: string;
  updated_at: string;
};

export type Outing = {
  id: string;
  join_token: string;
  title: string;
  created_by_user_id?: string | null;
  created_at: string;
  updated_at: string;
  members: OutingMember[];
};

export type Point = {
  lat: number;
  lon: number;
};

export type PlaceSearchResult = {
  display_name: string;
  lat: string;
  lon: string;
  primary_text?: string;
  secondary_text?: string;
  place_id?: string;
  source?: string;
};

type SearchProximity = {
  lat: number;
  lon: number;
};

export type PresetLocationPayload = {
  name: string;
  address: string;
  latitude: number;
  longitude: number;
};

export type PresetOverlayPayload = {
  id: string;
  name: string;
  source: "community";
  locations: PresetLocationPayload[];
  upvotes: number;
  downvotes: number;
};

export async function createRecentAddress(userId: string, address: AddressPayload) {
  const response = await fetch(`${API_URL}/users/${userId}/recent-addresses`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(address),
  });

  if (!response.ok) {
    throw new Error("Failed to save recent address");
  }

  return response.json();
}

export async function listRecentAddresses(userId: string) {
  const response = await fetch(`${API_URL}/users/${userId}/recent-addresses`, {
    credentials: "include",
  });

  if (!response.ok) {
    throw new Error("Failed to load recent addresses");
  }

  return response.json();
}

export async function createSavedAddress(userId: string, address: AddressPayload) {
  const response = await fetch(`${API_URL}/users/${userId}/saved-addresses`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(address),
  });

  if (!response.ok) {
    throw new Error("Failed to save address");
  }

  return response.json();
}

export async function listSavedAddresses(userId: string) {
  const response = await fetch(`${API_URL}/users/${userId}/saved-addresses`, {
    credentials: "include",
  });

  if (!response.ok) {
    throw new Error("Failed to load saved addresses");
  }

  return response.json();
}

export async function createOuting(title: string) {
  const response = await fetch(`${API_URL}/outings`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ title }),
  });

  if (!response.ok) {
    throw new Error("Failed to create outing");
  }

  return (await response.json()) as Outing;
}

export async function getOuting(token: string) {
  const response = await fetch(`${API_URL}/outings/${token}`, {
    credentials: "include",
  });

  if (!response.ok) {
    throw new Error("Failed to load outing");
  }

  return (await response.json()) as Outing;
}

export async function joinOuting(token: string, displayName: string) {
  const response = await fetch(`${API_URL}/outings/${token}/members`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ display_name: displayName }),
  });

  if (!response.ok) {
    throw new Error("Failed to join outing");
  }

  return (await response.json()) as OutingMember;
}

export async function updateOutingMember(
  token: string,
  memberId: string,
  data: {
    location_name: string;
    latitude: number;
    longitude: number;
    max_travel_minutes: number;
  },
) {
  const response = await fetch(`${API_URL}/outings/${token}/members/${memberId}`, {
    method: "PUT",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });

  if (!response.ok) {
    throw new Error("Failed to update member");
  }

  return (await response.json()) as OutingMember;
}

export async function getIsochrone(point: Point, minutes: number, departureTime?: string) {
  const response = await fetch(`${API_URL}/isochrone`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      ...point,
      minutes,
      ...(departureTime ? { departure_time: departureTime } : {}),
    }),
  });

  if (!response.ok) {
    const error = (await response.json().catch(() => null)) as { message?: string } | null;
    throw new Error(error?.message ?? "Failed to load isochrone");
  }

  return (await response.json()) as IsochroneFeature;
}

export async function searchPlaces(
  query: string,
  signal: AbortSignal,
  limit = 8,
  proximity?: SearchProximity,
): Promise<PlaceSearchResult[]> {
  const normalizedLimit = Math.min(10, Math.max(1, Math.round(limit)));
  const params = new URLSearchParams({
    q: query.trim(),
    limit: String(normalizedLimit),
  });
  let response: Response;
  try {
    response = await fetch(`${API_URL}/places/search?${params}`, { signal });
  } catch (error) {
    if (signal.aborted || !mapboxAccessToken) {
      throw error;
    }
    return searchPlacesWithMapbox(query, normalizedLimit, signal, proximity);
  }

  if (!response.ok) {
    const data = (await response.json().catch(() => null)) as { message?: string } | null;
    if (
      mapboxAccessToken &&
      [429, 502, 503, 504].includes(response.status)
    ) {
      return searchPlacesWithMapbox(query, normalizedLimit, signal, proximity);
    }
    throw new Error(data?.message ?? "Location search is temporarily unavailable.");
  }

  const data = (await response.json()) as {
    places?: Array<Omit<PlaceSearchResult, "lat" | "lon"> & {
      id?: string;
      lat: string | number;
      lon: string | number;
    }>;
  };
  return (data.places ?? []).map(({ id, ...place }) => ({
    ...place,
    place_id: place.place_id ?? id,
    lat: String(place.lat),
    lon: String(place.lon),
  }));
}

async function searchPlacesWithMapbox(
  query: string,
  limit: number,
  signal: AbortSignal,
  proximity?: SearchProximity,
): Promise<PlaceSearchResult[]> {
  if (!mapboxAccessToken) {
    throw new Error("Location search is temporarily unavailable.");
  }
  const params = new URLSearchParams({
    q: query.trim(),
    access_token: mapboxAccessToken,
    autocomplete: "true",
    permanent: "true",
    country: "sg",
    language: "en",
    bbox: "103.5935,1.1304,104.1076,1.4780",
    limit: String(limit),
  });
  if (proximity && Number.isFinite(proximity.lat) && Number.isFinite(proximity.lon)) {
    params.set("proximity", `${proximity.lon},${proximity.lat}`);
  }

  const response = await fetch(`${mapboxGeocodingURL}?${params}`, { signal });
  if (!response.ok) {
    throw new Error("Location search is temporarily unavailable.");
  }

  const data = (await response.json()) as {
    features?: Array<{
      id?: string;
      geometry?: { coordinates?: [number, number] };
      properties?: {
        mapbox_id?: string;
        name?: string;
        name_preferred?: string;
        full_address?: string;
        place_formatted?: string;
        coordinates?: {
          routable_points?: Array<{ latitude?: number; longitude?: number }>;
        };
      };
    }>;
  };

  return (data.features ?? []).flatMap((feature): PlaceSearchResult[] => {
    const properties = feature.properties;
    const routablePoint = properties?.coordinates?.routable_points?.find(
      (candidate) =>
        Number.isFinite(candidate.latitude) && Number.isFinite(candidate.longitude),
    );
    const longitude = routablePoint?.longitude ?? feature.geometry?.coordinates?.[0];
    const latitude = routablePoint?.latitude ?? feature.geometry?.coordinates?.[1];
    const primaryText = properties?.name_preferred || properties?.name;
    const displayName = properties?.full_address || properties?.place_formatted || primaryText;
    if (
      !primaryText ||
      !displayName ||
      !Number.isFinite(latitude) ||
      !Number.isFinite(longitude)
    ) {
      return [];
    }

    const mapboxID = properties?.mapbox_id || feature.id || `${latitude},${longitude}`;
    return [{
      display_name: displayName,
      primary_text: primaryText,
      secondary_text: displayName === primaryText ? undefined : displayName,
      place_id: `mapbox:${mapboxID}`,
      lat: String(latitude),
      lon: String(longitude),
      source: "mapbox",
    }];
  });
}

export type PresetSort = "new" | "top" | "trending";

export async function listCommunityPresets(query = "", sort: PresetSort = "new") {
  const params = new URLSearchParams();
  if (query.trim()) {
    params.set("q", query.trim());
  }
  if (sort !== "new") {
    params.set("sort", sort);
  }
  const response = await fetch(`${API_URL}/presets${params.size > 0 ? `?${params}` : ""}`);

  if (!response.ok) {
    throw new Error("Failed to load community presets");
  }

  return (await response.json()) as PresetOverlayPayload[];
}

export async function createCommunityPreset(data: {
  name: string;
  locations: PresetLocationPayload[];
}) {
  const response = await fetch(`${API_URL}/presets`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });

  if (!response.ok) {
    throw new Error("Failed to create community preset");
  }

  return (await response.json()) as PresetOverlayPayload;
}

export async function voteCommunityPreset(
  presetId: string,
  vote: "up" | "down" | "",
  previousVote: "up" | "down" | "",
) {
  const response = await fetch(`${API_URL}/presets/${presetId}/vote`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ vote, previous_vote: previousVote }),
  });

  if (!response.ok) {
    throw new Error("Failed to update preset vote");
  }

  return (await response.json()) as PresetOverlayPayload;
}
