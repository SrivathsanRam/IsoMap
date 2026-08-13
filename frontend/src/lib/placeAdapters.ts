import type { Place } from "../components/PlaceSearchInput";
import type { Address, AddressRequest } from "@/types";

export function placeToAddressRequest(place: Place, queryText?: string): AddressRequest {
  return {
    formatted_address: place.display_name,
    ...(place.place_id ? { place_id: place.place_id } : {}),
    latitude: parseFloat(place.lat),
    longitude: parseFloat(place.lon),
    ...(queryText ? { query_text: queryText } : {}),
  };
}

export function addressToPlace(address: Address): Place {
  return {
    display_name: address.formatted_address,
    place_id: address.place_id ?? undefined,
    primary_text: address.formatted_address,
    lat: String(address.latitude),
    lon: String(address.longitude),
  };
}
