import type { ApiErrorBody } from "@rexio-pay/shared-types";

/**
 * Typed fetch wrapper for the Go backend. All calls go through the same-origin
 * /api/backend proxy (the backend has no CORS middleware, and the proxy injects
 * the merchant API key from an httpOnly cookie).
 */

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly param?: string;

  constructor(status: number, code: string, message: string, param?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.param = param;
  }
}

export function isPlanLimitError(err: unknown): err is ApiError {
  return err instanceof ApiError && err.code === "plan_limit_exceeded";
}

type QueryValue = string | number | boolean | null | undefined;

export function buildQuery(
  params: Record<string, QueryValue | QueryValue[]>,
): string {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null) continue;
    if (Array.isArray(value)) {
      for (const item of value) search.append(key, String(item));
    } else {
      search.set(key, String(value));
    }
  }
  const qs = search.toString();
  return qs ? `?${qs}` : "";
}

async function parseBody<T>(res: Response): Promise<T> {
  const text = await res.text();
  if (!text) return undefined as T;
  try {
    return JSON.parse(text) as T;
  } catch {
    throw new ApiError(res.status, "internal_error", "Invalid response from server");
  }
}

/**
 * Request against the Go backend via the same-origin proxy.
 * Throws ApiError with the backend's {error:{code,message,param}} shape on failure.
 */
export async function apiRequest<T>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const res = await fetch(`/api/backend${path}`, {
    ...init,
    headers: {
      ...(init.body ? { "Content-Type": "application/json" } : {}),
      ...init.headers,
    },
    cache: "no-store",
  });

  if (!res.ok) {
    const body = await parseBody<ApiErrorBody>(res).catch(() => undefined);
    throw new ApiError(
      res.status,
      body?.error.code ?? "internal_error",
      body?.error.message ?? `Request failed with status ${res.status}`,
      body?.error.param,
    );
  }
  return parseBody<T>(res);
}

export const api = {
  get: <T>(path: string, init?: RequestInit) =>
    apiRequest<T>(path, { ...init, method: "GET" }),
  post: <T>(path: string, body?: unknown, init?: RequestInit) =>
    apiRequest<T>(path, {
      ...init,
      method: "POST",
      body: body === undefined ? undefined : JSON.stringify(body),
    }),
  patch: <T>(path: string, body: unknown, init?: RequestInit) =>
    apiRequest<T>(path, {
      ...init,
      method: "PATCH",
      body: JSON.stringify(body),
    }),
  delete: <T>(path: string, init?: RequestInit) =>
    apiRequest<T>(path, { ...init, method: "DELETE" }),
};
