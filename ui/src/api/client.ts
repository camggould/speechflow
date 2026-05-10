import type { ApiError as ApiErrorBody } from "@/api/types.gen";

const API_BASE = "/api/v1";

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, body: ApiErrorBody | { message: string; error?: string }) {
    super(body.message || `HTTP ${status}`);
    this.name = "ApiError";
    this.status = status;
    this.code = (body as ApiErrorBody).error ?? "http_error";
  }
}

/**
 * Thin fetch wrapper around the speechflow JSON API. All paths are appended
 * to `/api/v1` so callers pass e.g. `/sessions` not `/api/v1/sessions`.
 */
export async function apiFetch<T>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const headers = new Headers(init.headers);
  if (init.body) {
    headers.set("Content-Type", "application/json");
  }

  const res = await fetch(`${API_BASE}${path}`, { ...init, headers });

  if (!res.ok) {
    let body: ApiErrorBody;
    try {
      body = (await res.json()) as ApiErrorBody;
    } catch {
      body = { error: "http_error", message: `HTTP ${res.status} ${res.statusText}` };
    }
    throw new ApiError(res.status, body);
  }

  if (res.status === 204) {
    return undefined as T;
  }

  return (await res.json()) as T;
}
