export interface User {
  id: string;
  name: string;
  email?: string;
  created_at?: string;
  updated_at?: string;
}

export interface ApiResponse<T> {
  payload?: {
    data?: T;
    meta?: unknown;
  };
  messages?: string[];
  errorCode?: number;
}

export interface Point {
  lat: number;
  lon: number;
}

export interface IsochroneRequest extends Point {}

export interface Address {
  id: string;
  formatted_address: string;
  place_id: string;
  latitude: number;
  longitude: number;
  created_at?: string;
}

export interface AddressRequest {
  formatted_address: string;
  place_id?: string;
  latitude: number;
  longitude: number;
  query_text?: string;
  nickname?: string;
}

export interface AddressSearch {
  id: string;
  user_id: string;
  address_id: string;
  query_text: string;
  searched_at: string;
  address: Address;
}

export interface SavedAddress {
  id: string;
  user_id: string;
  address_id: string;
  nickname: string;
  saved_at: string;
  address: Address;
}
