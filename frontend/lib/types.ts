export interface URLItem {
  id: number;
  slug: string;
  short_url: string;
  original_url: string;
  title: string | null;
  favicon_url: string | null;
  is_active: boolean;
  is_expired: boolean;
  click_count: number;
  expires_at: string | null;
  created_at: string;
}

export interface URLListResponse {
  data: URLItem[];
  next_cursor: string | null;
}

export interface User {
  id: number;
  email: string;
  created_at: string;
}

export interface AuthResponse {
  token: string;
  expires_at: string;
  user: User;
}

export interface CreateURLPayload {
  original_url: string;
  custom_slug?: string;
  prefix?: string;
  title?: string;
  expires_at?: string;
}

export interface AnalyticsSummary {
  total_clicks: number;
  clicks_24h: number;
  series: { date: string; count: number }[];
  top_referrers: { label: string; count: number }[];
  devices: { label: string; count: number }[];
}

export interface ApiErrorBody {
  error: { code: string; message: string };
}

export type ApiKeyScope = "create" | "read" | "manage";

export interface ApiKey {
  id: number;
  name: string;
  key_prefix: string;
  scopes: ApiKeyScope[];
  created_at: string;
  expires_at?: string | null;
  last_used_at?: string | null;
  revoked_at?: string | null;
}

export interface ApiKeyCreateResponse {
  key: string; // raw secret, shown once
  data: ApiKey;
}
