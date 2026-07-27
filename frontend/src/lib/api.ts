const API_URL = import.meta.env.VITE_API_URL ?? "http://localhost:8000";

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

export async function getIsochrone(point: Point, minutes: number) {
  const response = await fetch(`${API_URL}/isochrone`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ ...point, minutes }),
  });

  if (!response.ok) {
    throw new Error("Failed to load isochrone");
  }

  return (await response.json()) as Point[];
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
