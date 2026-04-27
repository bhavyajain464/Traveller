import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import QRCodePreview from "../components/QRCodePreview";
import { useAuth } from "../lib/auth";
import { apiRequest } from "../lib/api";
import {
  clearActiveJourney,
  clearActiveSession,
  getActiveJourney,
  getActiveSession,
  saveActiveSession
} from "../lib/tickets";

const DELHI_FALLBACK_COORDS = {
  latitude: Number(import.meta.env.VITE_DELHI_FALLBACK_LAT || 28.632896),
  longitude: Number(import.meta.env.VITE_DELHI_FALLBACK_LON || 77.219574)
};

const FORCE_DELHI_LOCATION = import.meta.env.VITE_FORCE_DELHI_LOCATION === "true";
const TRACKING_POLL_MS = 15000;

function TicketsPage() {
  const { user } = useAuth();
  const activeJourney = getActiveJourney();
  const [activeSession, setActiveSession] = useState(() => getActiveSession());
  const [boardings, setBoardings] = useState([]);
  const [activeBoarding, setActiveBoarding] = useState(null);
  const [pendingBills, setPendingBills] = useState([]);
  const [trackingMessage, setTrackingMessage] = useState("");
  const [error, setError] = useState("");
  const [busyAction, setBusyAction] = useState("");
  const overdueBills = pendingBills.filter((bill) => isBillOverdue(bill.bill_date));

  useEffect(() => {
    let ignore = false;

    async function loadPageState() {
      if (!user?.id) {
        return;
      }

      try {
        const [sessionPayload, billsPayload] = await Promise.all([
          apiRequest("/sessions/me/active"),
          apiRequest("/bills/me/pending")
        ]);

        if (ignore) {
          return;
        }

        const session = sessionPayload.sessions?.[0];
        setPendingBills(billsPayload.bills || []);

        if (!session) {
          clearActiveSession();
          setActiveSession(null);
          setBoardings([]);
          setActiveBoarding(null);
          return;
        }

        const hydrated = toSessionState(session);
        saveActiveSession(hydrated);
        setActiveSession(hydrated);
        await refreshBoardingState(hydrated.id, ignore);
      } catch {
        // Preserve local state if the refresh fails.
      }
    }

    loadPageState();
    return () => {
      ignore = true;
    };
  }, [user?.id]);

  useEffect(() => {
    if (!activeSession?.id) {
      return undefined;
    }

    let cancelled = false;

    async function runTrackingTick() {
      const position = await getCurrentPosition();

      try {
        const heartbeat = await apiRequest("/boardings/tracking-heartbeat", {
          method: "POST",
          body: JSON.stringify({
            latitude: position.coords.latitude,
            longitude: position.coords.longitude,
            timestamp: new Date().toISOString()
          })
        });

        if (cancelled) {
          return;
        }

        setBoardings(heartbeat.boardings || []);
        setActiveBoarding(heartbeat.active_boarding || null);
        setTrackingMessage(heartbeat.message || "");
      } catch {
        // Leave the last good tracking state in place if background sync fails.
      }
    }

    runTrackingTick();
    const intervalID = window.setInterval(runTrackingTick, TRACKING_POLL_MS);

    return () => {
      cancelled = true;
      window.clearInterval(intervalID);
    };
  }, [activeSession?.id]);

  async function refreshBoardingState(sessionID, ignore = false) {
    const snapshot = await getBoardingSnapshot(sessionID);
    if (ignore) {
      return snapshot;
    }

    setBoardings(snapshot.boardings);
    setActiveBoarding(snapshot.activeBoarding);
    return snapshot;
  }

  async function checkIn() {
    try {
      setBusyAction("checkin");
      setError("");
      const position = await getCurrentPosition();
      const payload = await apiRequest("/sessions/checkin", {
        method: "POST",
        body: JSON.stringify({
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
          stop_id: activeJourney?.fromStopID || undefined
        })
      });

      const nextSession = toSessionState(payload.session, payload.qr_code);
      saveActiveSession(nextSession);
      setActiveSession(nextSession);
      setBoardings([]);
      setActiveBoarding(null);
      setTrackingMessage("Ticket is active. Tracking is now watching for your next boarding.");
      await refreshPendingBills();
    } catch (checkInError) {
      setError(checkInError.message || "Check-in failed.");
      await refreshPendingBills();
    } finally {
      setBusyAction("");
    }
  }

  async function checkOut() {
    if (!activeSession) {
      return;
    }

    try {
      setBusyAction("checkout");
      setError("");
      const position = await getCurrentPosition();
      await apiRequest("/sessions/checkout", {
        method: "POST",
        body: JSON.stringify({
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
          stop_id: activeJourney?.toStopID || undefined
        })
      });

      clearActiveSession();
      clearActiveJourney();
      setActiveSession(null);
      setBoardings([]);
      setActiveBoarding(null);
      setTrackingMessage("");
      await refreshPendingBills();
    } catch (checkOutError) {
      setError(checkOutError.message || "Check-out failed.");
    } finally {
      setBusyAction("");
    }
  }

  async function refreshPendingBills() {
    if (!user?.id) {
      return;
    }

    try {
      const payload = await apiRequest("/bills/me/pending");
      setPendingBills(payload.bills || []);
    } catch {
      // Ignore billing refresh errors on the rider page.
    }
  }

  const trackingState = getTrackingState({
    activeSession,
    activeBoarding,
    boardings,
    overdueBills
  });

  return (
    <section className="route-page">
      <div className="page-header">
        <div>
          <h2>Tickets</h2>
          <p className="lead">
            Your QR ticket stays active for inspection while background tracking determines the boardings that feed billing.
          </p>
        </div>
      </div>

      {error ? <section className="card"><p className="status-error">{error}</p></section> : null}

      {overdueBills.length > 0 && !activeSession ? (
        <section className="card selected-card">
          <p className="eyebrow">Billing Gate</p>
          <h3>Overdue bill pending</h3>
          <p className="lead">
            New travel is blocked until the outstanding bill from a previous day is paid.
          </p>
          <div className="route-meta-grid">
            <div>
              <strong>{overdueBills.length}</strong>
              <span>Overdue bill{overdueBills.length === 1 ? "" : "s"}</span>
            </div>
            <div>
              <strong>{formatCurrency(sumPendingBills(overdueBills))}</strong>
              <span>Outstanding total</span>
            </div>
          </div>
          <div className="hero-actions">
            <Link className="primary-link" to="/bills">Review bills</Link>
            <Link className="secondary-link" to="/profile">Open profile</Link>
          </div>
        </section>
      ) : null}

      {activeSession ? (
        <section className="card ticket-card full-ticket-card tracking-ticket-card">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Active Ticket</p>
              <h3>{trackingState.label}</h3>
            </div>
            <button
              type="button"
              className="ghost-button"
              onClick={() => {
                clearActiveSession();
                window.location.reload();
              }}
            >
              Reset local view
            </button>
          </div>

          <div className="tracking-status-banner">
            <strong>{trackingState.label}</strong>
            <span>{trackingState.description}</span>
          </div>

          <div className="ticket-preview tracking-ticket-preview">
            <div className="ticket-copy">
              <p className="eyebrow">Session-linked QR</p>
              <strong>{user?.name}</strong>
              <p>{activeSession.checkInTimeLabel}</p>
              <p>{activeSession.checkInLocationLabel}</p>
              <p className="status-muted">Session ID: {activeSession.id}</p>
              {trackingMessage ? <p className="status-muted">{trackingMessage}</p> : null}
            </div>

            <div className="tracking-qr-panel">
              <QRCodePreview value={activeSession.qrCode} size={220} />
              <p className="status-muted">Show this QR to inspectors while tracking runs in the background.</p>
            </div>
          </div>

          <div className="route-meta-grid">
            <div>
              <strong>{boardings.length}</strong>
              <span>Total tracked segments</span>
            </div>
            <div>
              <strong>{activeBoarding?.RouteID || "Waiting"}</strong>
              <span>Current route</span>
            </div>
            <div>
              <strong>{activeBoarding?.VehicleID || "No vehicle yet"}</strong>
              <span>Current vehicle</span>
            </div>
          </div>

          {boardings.length > 0 ? (
            <div className="tracking-segment-list">
              {boardings.map((boarding, index) => (
                <div key={boarding.id} className="tracking-segment-item">
                  <strong>Segment {index + 1}: {boarding.route_id}</strong>
                  <span>
                    Boarded {formatDateTime(boarding.boarding_time)}
                    {boarding.alighting_time ? `, alighted ${formatDateTime(boarding.alighting_time)}` : ", currently active"}
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <p className="status-muted">
              No vehicle matched yet. The ticket is live and tracking will attach boardings as movement data arrives.
            </p>
          )}

          <div className="toolbar">
            <button type="button" className="primary-button" disabled={busyAction === "checkout"} onClick={checkOut}>
              {busyAction === "checkout" ? "Ending tracking..." : "End Tracking And Close Ticket"}
            </button>
            <Link className="secondary-link" to="/plan">Plan a trip</Link>
            <Link className="secondary-link" to="/departures">Browse departures</Link>
          </div>
        </section>
      ) : (
        <section className="card empty-state-card">
          <h3>No active ticket yet</h3>
          <p className="lead">
            Start tracking once, get a QR ticket for inspection, and let boardings and alightings drive billing in the background.
          </p>
          {activeJourney ? (
            <p className="status-muted">
              Planned journey: {activeJourney.fromStopName} to {activeJourney.toStopName}
            </p>
          ) : null}
          <div className="hero-actions">
            <button
              type="button"
              className="primary-button"
              disabled={busyAction === "checkin" || overdueBills.length > 0}
              onClick={checkIn}
            >
              {busyAction === "checkin" ? "Starting tracking..." : "Start Tracking And Generate Ticket"}
            </button>
            <Link className="secondary-link" to="/plan">Plan a trip</Link>
            <Link className="secondary-link" to="/departures">Browse departures</Link>
          </div>
        </section>
      )}
    </section>
  );
}

async function getBoardingSnapshot(sessionID) {
  const [boardingsPayload, activePayload] = await Promise.all([
    apiRequest(`/boardings/sessions/${encodeURIComponent(sessionID)}`),
    apiRequest(`/boardings/sessions/${encodeURIComponent(sessionID)}/active`)
  ]);

  return {
    boardings: boardingsPayload.boardings || [],
    activeBoarding: activePayload.active ? activePayload.boarding : null
  };
}

function getTrackingState({ activeSession, activeBoarding, boardings, overdueBills }) {
  if (!activeSession && overdueBills.length > 0) {
    return {
      label: "Overdue bill pending",
      description: "Settle unpaid bills from previous days before starting the next travel day."
    };
  }

  if (!activeSession) {
    return {
      label: "Ready",
      description: "Start tracking when you begin moving."
    };
  }

  if (activeBoarding && boardings.length > 1) {
    return {
      label: "Transferred",
      description: "Your QR is still valid and tracking has linked more than one segment to this session."
    };
  }

  if (activeBoarding) {
    return {
      label: "On vehicle",
      description: `Tracking is attached to route ${activeBoarding.RouteID} for billing.`
    };
  }

  if (boardings.length > 0) {
    return {
      label: "Ride ended",
      description: "Your last segment has ended. Stay checked in if you expect another transfer."
    };
  }

  return {
    label: "Checked in",
    description: "Your QR ticket is live and tracking is waiting for the first boarding match."
  };
}

function toSessionState(session, qrCodeOverride) {
  return {
    id: session.id,
    qrCode: qrCodeOverride || session.qr_code,
    checkInTimeLabel: new Date(session.check_in_time).toLocaleString(),
    checkInLocationLabel: `${Number(session.check_in_lat).toFixed(5)}, ${Number(session.check_in_lon).toFixed(5)}`
  };
}
function sumPendingBills(bills) {
  return bills.reduce((total, bill) => total + Number(bill.total_fare || 0), 0);
}

function formatCurrency(value) {
  return new Intl.NumberFormat("en-IN", {
    style: "currency",
    currency: "INR",
    maximumFractionDigits: 2
  }).format(value || 0);
}

function formatDateTime(value) {
  return new Date(value).toLocaleString();
}

function isBillOverdue(billDateValue) {
  if (!billDateValue) {
    return false;
  }

  const billDate = new Date(billDateValue);
  const todayStart = new Date();
  todayStart.setHours(0, 0, 0, 0);
  return billDate < todayStart;
}

function getCurrentPosition() {
  if (FORCE_DELHI_LOCATION) {
    return Promise.resolve(createPosition(DELHI_FALLBACK_COORDS.latitude, DELHI_FALLBACK_COORDS.longitude));
  }

  return new Promise((resolve) => {
    if (!navigator.geolocation) {
      resolve(createPosition(DELHI_FALLBACK_COORDS.latitude, DELHI_FALLBACK_COORDS.longitude));
      return;
    }

    navigator.geolocation.getCurrentPosition((position) => {
      const coords = position.coords;

      if (isWithinDelhi(coords.latitude, coords.longitude)) {
        resolve(position);
        return;
      }

      resolve(createPosition(DELHI_FALLBACK_COORDS.latitude, DELHI_FALLBACK_COORDS.longitude));
    }, () => {
      resolve(createPosition(DELHI_FALLBACK_COORDS.latitude, DELHI_FALLBACK_COORDS.longitude));
    }, { enableHighAccuracy: true, timeout: 10000 });
  });
}

function createPosition(latitude, longitude) {
  return {
    coords: {
      latitude,
      longitude
    }
  };
}

function isWithinDelhi(latitude, longitude) {
  return latitude >= 28.4 && latitude <= 28.95 && longitude >= 76.8 && longitude <= 77.45;
}

export default TicketsPage;
