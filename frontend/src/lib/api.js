const RAW_API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080";
const API_ROOT_URL = RAW_API_BASE_URL.replace(/\/api\/v1\/?$/, "");
const API_V1_BASE_URL = `${API_ROOT_URL}/api/v1`;
const API_BASE_URL = API_V1_BASE_URL;
const AUTH_TOKEN_KEY = "traveller.auth.token";

export function getAuthToken() {
  try {
    return window.localStorage.getItem(AUTH_TOKEN_KEY) || "";
  } catch {
    return "";
  }
}

export function setAuthToken(token) {
  try {
    if (!token) {
      window.localStorage.removeItem(AUTH_TOKEN_KEY);
      return;
    }

    window.localStorage.setItem(AUTH_TOKEN_KEY, token);
  } catch {
    // Ignore storage failures and keep the in-memory request working.
  }
}

export function clearAuthToken() {
  setAuthToken("");
}

export async function apiRequest(path, options = {}) {
  const token = getAuthToken();
  const headers = {
    ...(options.body ? { "Content-Type": "application/json" } : {}),
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...(options.headers || {})
  };

  const response = await fetch(buildApiURL(path), {
    ...options,
    headers
  });

  const contentType = response.headers.get("content-type") || "";
  const payload = contentType.includes("application/json")
    ? await response.json()
    : await response.text();

  if (!response.ok) {
    const errorMessage =
      typeof payload === "string" ? payload : payload.error || JSON.stringify(payload);
    throw new Error(errorMessage || `Request failed (${response.status})`);
  }

  return payload;
}

function buildApiURL(path) {
  if (/^https?:\/\//.test(path)) {
    return path;
  }

  if (path.startsWith("/v3/") || path === "/health") {
    return `${API_ROOT_URL}${path}`;
  }

  return `${API_V1_BASE_URL}${path}`;
}

export { API_ROOT_URL, API_V1_BASE_URL, API_BASE_URL, AUTH_TOKEN_KEY };
