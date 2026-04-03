import { useAuthStore } from "../stores/auth_store.ts";
import { ApiError } from "./api_models.ts";
import type { RequestOptions } from "./api_models.ts";

let isRefreshing = false;
let failedQueue: Array<{
  resolve: () => void;
  reject: (err: unknown) => void;
}> = [];

const processQueue = (error: unknown) => {
  failedQueue.forEach((p) => (error ? p.reject(error) : p.resolve()));
  failedQueue = [];
};

function appendParam(
  searchParams: URLSearchParams,
  key: string,
  value: unknown,
): void {
  if (value === null || value === undefined) return;
  if (Array.isArray(value)) {
    value.forEach((item, i) => appendParam(searchParams, `${key}[${i}]`, item));
  } else if (typeof value === "object") {
    Object.entries(value as Record<string, unknown>).forEach(([k, v]) =>
      appendParam(searchParams, `${key}[${k}]`, v),
    );
  } else {
    searchParams.append(key, String(value));
  }
}

function buildUrl(path: string, params?: Record<string, unknown>): string {
  const url = new URL(`/api/${path}`, window.location.origin);
  if (params) {
    Object.entries(params).forEach(([k, v]) =>
      appendParam(url.searchParams, k, v),
    );
  }
  return url.pathname + url.search;
}

async function doFetch(
  method: string,
  path: string,
  body: unknown,
  options: RequestOptions,
): Promise<Response> {
  const url = buildUrl(path, options.params);
  const headers: Record<string, string> = { ...options.headers };
  let fetchBody: BodyInit | undefined;

  if (body instanceof FormData) {
    fetchBody = body;
  } else if (body instanceof Blob) {
    fetchBody = body;
    if (body.type) headers["Content-Type"] = body.type;
  } else if (body !== null && body !== undefined) {
    headers["Content-Type"] = "application/json";
    fetchBody = JSON.stringify(body);
  }

  return fetch(url, {
    method,
    headers,
    body: fetchBody,
    credentials: "include",
  });
}

async function handleResponse<T>(
  response: Response,
  options: RequestOptions,
): Promise<{ data: T }> {
  if (options.responseType === "blob") {
    return { data: (await response.blob()) as T };
  }
  const text = await response.text();
  return { data: (text ? JSON.parse(text) : null) as T };
}

async function request<T>(
  method: string,
  path: string,
  body: unknown,
  options: RequestOptions,
  isRetry = false,
): Promise<{ data: T }> {
  let response: Response;

  try {
    response = await doFetch(method, path, body, options);
  } catch {
    throw new ApiError("Network Error", 0, null, true);
  }

  if (response.status === 401 && !isRetry) {
    if (isRefreshing) {
      await new Promise<void>((resolve, reject) =>
        failedQueue.push({ resolve, reject }),
      );
      return request<T>(method, path, body, options, true);
    }

    isRefreshing = true;

    try {
      await doFetch(method, path, body, options);
      processQueue(null);
      isRefreshing = false;
      return await request<T>(method, path, body, options, true);
    } catch (err) {
      processQueue(err);
      isRefreshing = false;
      throw err;
    }
  }

  if (!response.ok) {
    let data: unknown = null;
    try {
      data = await response.json();
    } catch {
      // ignore parse errors
    }

    if (response.status === 401) {
      const auth = useAuthStore();
      const isAuthEndpoint = /\/auth\/(current|logout|login)/.test(path);
      if (!isAuthEndpoint && auth.isAuthenticated) {
        await auth.logoutUser();
      }
    }

    const msg =
      (data as { message?: string } | null)?.message ?? response.statusText;
    throw new ApiError(msg, response.status, data);
  }

  return handleResponse<T>(response, options);
}

const apiClient = {
  get: <T = any>(path: string, options?: RequestOptions) =>
    request<T>("GET", path, undefined, options ?? {}),
  post: <T = any>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>("POST", path, body, options ?? {}),
  put: <T = any>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>("PUT", path, body, options ?? {}),
  patch: <T = any>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>("PATCH", path, body, options ?? {}),
  delete: <T = any>(path: string, options?: RequestOptions) =>
    request<T>("DELETE", path, undefined, options ?? {}),
};

export default apiClient;
