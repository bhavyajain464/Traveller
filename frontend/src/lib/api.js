const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080/api/v1";
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

  const response = await fetch(`${API_BASE_URL}${path}`, {
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

export { API_BASE_URL, AUTH_TOKEN_KEY };
