import axios, { type AxiosInstance, AxiosError, type AxiosResponse } from "axios";
import type {
    User,
    Topic,
    Post,
    Comment,
    LoginResponse,
    CreateTopicRequest,
    UpdateTopicRequest,
    CreatePostRequest,
    UpdatePostRequest,
    CreateCommentRequest,
    UpdateCommentRequest,
} from "../types";

// Use environment variable for API URL (set in .env.development and .env.production)
const API_BASE_URL = import.meta.env.VITE_API_URL || "http://localhost:8000";

type ApiResponse<T> = {
  payload?: { data?: T };
  messages?: string[];
  errorCode?: number;
};

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
  private userId: number | null = null;

  constructor() {
    this.client = axios.create({
      baseURL: API_BASE_URL,
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

  setUserId(id: number | null) {
    this.userId = id;
  }

  getUserId(): number | null {
    return this.userId;
  }

  // Auth
  async login(username: string): Promise<LoginResponse> {
    const response = await this.client.post<ApiResponse<LoginResponse>>("/auth/login", {
      username,
    });
    const data = unwrap(response);
    this.setUserId(data.user.id);
    return data;
  }

  async logout(): Promise<void> {
    await this.client.post("/auth/logout");
    this.setUserId(null);
  }

  async getCurrentUser(): Promise<User> {
    const response = await this.client.get<ApiResponse<User>>("/users/me");
    return unwrap(response);
  }

  // Topics
  async getTopics(): Promise<Topic[]> {
    const response = await this.client.get<ApiResponse<Topic[]>>("/topics");
    return unwrap(response);
  }

  async getTopic(id: number): Promise<Topic> {
    const response = await this.client.get<ApiResponse<Topic>>(`/topics/${id}`);
    return unwrap(response);
  }

  async createTopic(data: CreateTopicRequest): Promise<Topic> {
    const response = await this.client.post<ApiResponse<Topic>>("/topics", data);
    return unwrap(response);
  }

  async updateTopic(id: number, data: UpdateTopicRequest): Promise<Topic> {
    const response = await this.client.put<ApiResponse<Topic>>(`/topics/${id}`, data);
    return unwrap(response);
  }

  async deleteTopic(id: number): Promise<void> {
    await this.client.delete(`/topics/${id}`);
  }

  // Posts
  async getPosts(topicId?: number): Promise<Post[]> {
    const params = topicId ? { topic_id: topicId } : {};
    const response = await this.client.get<ApiResponse<Post[]>>("/posts", { params });
    return unwrap(response);
  }

  async getPost(id: number): Promise<Post> {
    const response = await this.client.get<ApiResponse<Post>>(`/posts/${id}`);
    return unwrap(response);
  }

  async createPost(data: CreatePostRequest): Promise<Post> {
    const response = await this.client.post<ApiResponse<Post>>("/posts", data);
    return unwrap(response);
  }

  async updatePost(id: number, data: UpdatePostRequest): Promise<Post> {
    const response = await this.client.put<ApiResponse<Post>>(`/posts/${id}`, data);
    return unwrap(response);
  }

  async deletePost(id: number): Promise<void> {
    await this.client.delete(`/posts/${id}`);
  }

  // Comments
  async getComments(postId: number): Promise<Comment[]> {
    const response = await this.client.get<ApiResponse<Comment[]>>("/comments", {
      params: { post_id: postId },
    });
    return unwrap(response);
  }

  async createComment(data: CreateCommentRequest): Promise<Comment> {
    const response = await this.client.post<ApiResponse<Comment>>("/comments", data);
    return unwrap(response);
  }

  async updateComment(id: number, data: UpdateCommentRequest): Promise<Comment> {
    const response = await this.client.put<ApiResponse<Comment>>(`/comments/${id}`, data);
    return unwrap(response);
  }

  async deleteComment(id: number): Promise<void> {
    await this.client.delete(`/comments/${id}`);
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
