const JOURNEY_STORAGE_KEY = "traveller.active.journey";
const SESSION_STORAGE_KEY = "traveller.active.session";

export function getActiveJourney() {
  try {
    const value = window.localStorage.getItem(JOURNEY_STORAGE_KEY);
    return value ? JSON.parse(value) : null;
  } catch {
    return null;
  }
}

export function saveActiveJourney(journey) {
  window.localStorage.setItem(JOURNEY_STORAGE_KEY, JSON.stringify(journey));
}

export function clearActiveJourney() {
  window.localStorage.removeItem(JOURNEY_STORAGE_KEY);
}

export function getActiveSession() {
  try {
    const value = window.localStorage.getItem(SESSION_STORAGE_KEY);
    return value ? JSON.parse(value) : null;
  } catch {
    return null;
  }
}

export function saveActiveSession(session) {
  window.localStorage.setItem(SESSION_STORAGE_KEY, JSON.stringify(session));
}

export function clearActiveSession() {
  window.localStorage.removeItem(SESSION_STORAGE_KEY);
}
