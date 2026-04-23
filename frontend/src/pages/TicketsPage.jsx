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

function TicketsPage() {
  const { user } = useAuth();
  const activeJourney = getActiveJourney();
  const [activeSession, setActiveSession] = useState(() => getActiveSession());
  const [error, setError] = useState("");
  const [busyAction, setBusyAction] = useState("");

  useEffect(() => {
    let ignore = false;

    async function loadActiveSession() {
      if (!user?.id) {
        return;
      }

      try {
        const payload = await apiRequest(`/sessions/users/${encodeURIComponent(user.id)}/active`);
        if (ignore) {
          return;
        }
        const session = payload.sessions?.[0];
        if (session) {
          const hydrated = toSessionState(session);
          saveActiveSession(hydrated);
          setActiveSession(hydrated);
        } else {
          clearActiveSession();
          setActiveSession(null);
        }
      } catch {
        // Keep the locally stored state if the refresh fails.
      }
    }

    loadActiveSession();
    return () => {
      ignore = true;
    };
  }, [user?.id]);

  async function checkIn() {
    try {
      setBusyAction("checkin");
      setError("");
      const position = await getCurrentPosition();
      const payload = await apiRequest("/sessions/checkin", {
        method: "POST",
        body: JSON.stringify({
          user_id: user.id,
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
          stop_id: activeJourney?.fromStopID || undefined
        })
      });

      const nextSession = toSessionState(payload.session, payload.qr_code);
      saveActiveSession(nextSession);
      setActiveSession(nextSession);
    } catch (checkInError) {
      setError(checkInError.message || "Check-in failed.");
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
          session_id: activeSession.id,
          qr_code: activeSession.qrCode,
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
          stop_id: activeJourney?.toStopID || undefined
        })
      });

      clearActiveSession();
      clearActiveJourney();
      setActiveSession(null);
    } catch (checkOutError) {
      setError(checkOutError.message || "Check-out failed.");
    } finally {
      setBusyAction("");
    }
  }

  return (
    <section className="route-page">
      <div className="page-header">
        <div>
          <h2>Tickets</h2>
          <p className="lead">
            Your live QR is created when you check in. It includes the session tied to your time and location, and it disappears when you check out.
          </p>
        </div>
      </div>

      {error ? <section className="card"><p className="status-error">{error}</p></section> : null}

      {activeSession ? (
        <section className="card ticket-card full-ticket-card">
          <div className="section-heading">
            <h3>Live travel QR</h3>
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

          <div className="ticket-preview">
            <QRCodePreview value={activeSession.qrCode} size={260} />
            <div className="ticket-copy">
              <p className="eyebrow">Active until checkout</p>
              <strong>{user?.name}</strong>
              <p>{activeSession.checkInTimeLabel}</p>
              <p>{activeSession.checkInLocationLabel}</p>
              <p className="status-muted">Session ID: {activeSession.id}</p>
            </div>
          </div>

          <div className="toolbar">
            <button type="button" className="primary-button" disabled={busyAction === "checkout"} onClick={checkOut}>
              {busyAction === "checkout" ? "Checking out..." : "Check Out And Destroy QR"}
            </button>
          </div>
        </section>
      ) : (
        <section className="card empty-state-card">
          <h3>No live QR yet</h3>
          <p className="lead">
            Check in to create a live QR using your current time and location. That QR should exist only while the journey session is active.
          </p>
          {activeJourney ? (
            <p className="status-muted">
              Planned journey: {activeJourney.fromStopName} to {activeJourney.toStopName}
            </p>
          ) : null}
          <div className="hero-actions">
            <button type="button" className="primary-button" disabled={busyAction === "checkin"} onClick={checkIn}>
              {busyAction === "checkin" ? "Checking in..." : "Check In And Generate QR"}
            </button>
            <Link className="secondary-link" to="/plan">Plan a trip</Link>
            <Link className="secondary-link" to="/departures">Browse departures</Link>
          </div>
        </section>
      )}
    </section>
  );
}

function toSessionState(session, qrCodeOverride) {
  return {
    id: session.id,
    qrCode: qrCodeOverride || session.qr_code,
    checkInTimeLabel: new Date(session.check_in_time).toLocaleString(),
    checkInLocationLabel: `${Number(session.check_in_lat).toFixed(5)}, ${Number(session.check_in_lon).toFixed(5)}`
  };
}

function getCurrentPosition() {
  return new Promise((resolve, reject) => {
    if (!navigator.geolocation) {
      reject(new Error("Geolocation is not available in this browser."));
      return;
    }

    navigator.geolocation.getCurrentPosition(resolve, () => {
      reject(new Error("Location permission is required to check in or out."));
    }, { enableHighAccuracy: true, timeout: 10000 });
  });
}

export default TicketsPage;
