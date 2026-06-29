const API_URL = import.meta.env.VITE_API_URL ?? "http://localhost:8000";

export type AddressPayload = {
  formatted_address: string;
  place_id?: string;
  latitude: number;
  longitude: number;
  query_text?: string;
  nickname?: string;
};

export async function createRecentAddress(userId: string, address: AddressPayload) {
  const response = await fetch(`${API_URL}/users/${userId}/recent-addresses`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(address),
  });

  if (!response.ok) {
    throw new Error("Failed to save recent address");
  }

  return response.json();
}

export async function listRecentAddresses(userId: string) {
  const response = await fetch(`${API_URL}/users/${userId}/recent-addresses`);

  if (!response.ok) {
    throw new Error("Failed to load recent addresses");
  }

  return response.json();
}

export async function createSavedAddress(userId: string, address: AddressPayload) {
  const response = await fetch(`${API_URL}/users/${userId}/saved-addresses`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(address),
  });

  if (!response.ok) {
    throw new Error("Failed to save address");
  }

  return response.json();
}

export async function listSavedAddresses(userId: string) {
  const response = await fetch(`${API_URL}/users/${userId}/saved-addresses`);

  if (!response.ok) {
    throw new Error("Failed to load saved addresses");
  }

  return response.json();
}