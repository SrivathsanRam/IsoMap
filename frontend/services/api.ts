import axios, { type AxiosInstance, AxiosError, type AxiosResponse } from "axios";
import type {
  AddressRequest,
  Address,
  AddressSearch,
  ApiResponse,
  Point,
  SavedAddress,
  User,
} from "../types";

const API_BASE_URL = import.meta.env.VITE_API_URL || "http://localhost:8000";

function unwrap<T>(response: AxiosResponse<ApiResponse<T>>): T {
  const data = response.data?.payload?.data;
  if (data === undefined) {
    const message = response.data?.messages?.[0] ?? "Unexpected API response";
    throw new Error(message);
  }
  return data;
}

class ApiService {
  private client: AxiosInstance;
  private userId: string | null = null;

  constructor() {
    this.client = axios.create({
      baseURL: API_BASE_URL,
      withCredentials: true,
      headers: {
        "Content-Type": "application/json",
      },
    });

    // Add interceptor to include X-User-ID header
    this.client.interceptors.request.use((config) => {
      if (this.userId) {
        config.headers["X-User-ID"] = this.userId.toString();
      }
      return config;
    });
  }

  setUserId(id: string | null) {
    this.userId = id;
  }

  getUserId(): string | null {
    return this.userId;
  }

  // Auth
  async loginWithGoogle(credential: string): Promise<User> {
    const response = await this.client.post<ApiResponse<User>>("/auth/google", { credential });
    const user = unwrap(response);
    this.setUserId(user.id);
    return user;
  }

  async logout(): Promise<void> {
    await this.client.post("/auth/logout");
    this.setUserId(null);
  }

  async getCurrentUser(): Promise<User> {
    const response = await this.client.get<ApiResponse<User>>("/auth/me");
    const user = unwrap(response);
    this.setUserId(user.id);
    return user;
  }

  async getUsers(): Promise<User[]> {
    const response = await this.client.get<ApiResponse<User[]>>("/users");
    return unwrap(response);
  }

  async getIsochrone(point: Point): Promise<Point[]> {
    const response = await this.client.post<Point[]>("/isochrone", point);
    return response.data;
  }

  async createRecentAddress(userId: string, data: AddressRequest): Promise<Address> {
    const response = await this.client.post<ApiResponse<Address>>(
      `/users/${userId}/recent-addresses`,
      data
    );
    return unwrap(response);
  }

  async getRecentAddresses(userId: string, limit?: number): Promise<AddressSearch[]> {
    const response = await this.client.get<ApiResponse<AddressSearch[]>>(
      `/users/${userId}/recent-addresses`,
      { params: limit ? { limit } : undefined }
    );
    return unwrap(response);
  }

  async createSavedAddress(userId: string, data: AddressRequest): Promise<Address> {
    const response = await this.client.post<ApiResponse<Address>>(
      `/users/${userId}/saved-addresses`,
      data
    );
    return unwrap(response);
  }

  async getSavedAddresses(userId: string): Promise<SavedAddress[]> {
    const response = await this.client.get<ApiResponse<SavedAddress[]>>(
      `/users/${userId}/saved-addresses`
    );
    return unwrap(response);
  }
}

// Singleton instance
export const api = new ApiService();

// Error handler helper
export function getErrorMessage(error: unknown): string {
  if (axios.isAxiosError(error)) {
    const axiosError = error as AxiosError<ApiResponse<unknown> & { error?: string }>;
    return (
      axiosError.response?.data?.error ||
      axiosError.response?.data?.messages?.[0] ||
      axiosError.message
    );
  }
  return "An unexpected error occurred";
}
