import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import QRCodePreview from "../components/QRCodePreview";
import { apiRequest } from "../lib/api";

function SessionsPage() {
  const [activeSession, setActiveSession] = useState(null);
  const [recentSessions, setRecentSessions] = useState([]);
  const [pendingBills, setPendingBills] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let ignore = false;

    async function loadSessionsView() {
      try {
        setIsLoading(true);
        setError("");

        const [activePayload, sessionsPayload, billsPayload] = await Promise.all([
          apiRequest("/sessions/me/active"),
          apiRequest("/sessions/me?limit=12"),
          apiRequest("/bills/me/pending")
        ]);

        if (ignore) {
          return;
        }

        setActiveSession(activePayload.sessions?.[0] || null);
        setRecentSessions(sessionsPayload.sessions || []);
        setPendingBills(billsPayload.bills || []);
      } catch (loadError) {
        if (!ignore) {
          setError(loadError.message || "Unable to load your sessions.");
        }
      } finally {
        if (!ignore) {
          setIsLoading(false);
        }
      }
    }

    loadSessionsView();
    return () => {
      ignore = true;
    };
  }, []);

  return (
    <section className="route-page">
      <div className="page-header">
        <div>
          <p className="eyebrow">Sessions</p>
          <h2>Your ticket and travel timeline</h2>
          <p className="lead">
            See whether your QR ticket is active, what happened in recent sessions, and whether any unpaid daily bill is blocking the next travel day.
          </p>
        </div>
      </div>

      {error ? <section className="card"><p className="status-error">{error}</p></section> : null}

      <section className="hero-panel">
        <div>
          <p className="eyebrow">Current status</p>
          <h3>{activeSession ? "Ticket active" : "No active ticket"}</h3>
          <p className="lead">
            {activeSession
              ? "Your QR is live and can keep collecting tracked boardings until you end the ride."
              : "You are not currently checked in. Start a ticket from Tickets when you begin traveling."}
          </p>
          <div className="hero-actions">
            <Link className="primary-link" to="/tickets">
              {activeSession ? "Open active ticket" : "Start a new ticket"}
            </Link>
            <Link className="secondary-link" to="/bills">Review bills</Link>
          </div>
        </div>

        <div className="card api-card">
          <span>Billing gate</span>
          <strong>
            {pendingBills.length > 0
              ? `${pendingBills.length} pending bill${pendingBills.length === 1 ? "" : "s"}`
              : "No pending bills"}
          </strong>
        </div>
      </section>

      <div className="dashboard-grid">
        <section className="card">
          <div className="section-heading">
            <h3>Active session</h3>
          </div>

          {isLoading ? (
            <p className="lead">Loading your current ticket...</p>
          ) : activeSession ? (
            <div className="ticket-preview tracking-ticket-preview">
              <div className="feature-stack">
                <div>
                  <strong>Checked in</strong>
                  <p>{formatDateTime(activeSession.check_in_time)}</p>
                </div>
                <div>
                  <strong>Current state</strong>
                  <p>{formatStatus(activeSession.status)}</p>
                </div>
                <div>
                  <strong>Tracked fare so far</strong>
                  <p>{formatCurrency(activeSession.total_fare)}</p>
                </div>
                <div>
                  <strong>Distance so far</strong>
                  <p>{formatDistance(activeSession.total_distance)}</p>
                </div>
                <p className="status-muted">Session ID: {activeSession.id}</p>
              </div>

              <div className="tracking-qr-panel">
                <QRCodePreview value={activeSession.qr_code} size={200} />
                <p className="status-muted">This QR identifies the active session while tracking runs in the background.</p>
              </div>
            </div>
          ) : (
            <div className="empty-bills-state">
              <strong>No active session</strong>
              <p className="lead">Your next check-in will create a new QR ticket and tracking session.</p>
            </div>
          )}
        </section>

        <section className="card">
          <div className="section-heading">
            <h3>What this page shows</h3>
          </div>
          <div className="feature-stack">
            <div>
              <strong>Active ticket</strong>
              <p>Your live QR and the current session status.</p>
            </div>
            <div>
              <strong>Recent sessions</strong>
              <p>Completed and active sessions in reverse chronological order.</p>
            </div>
            <div>
              <strong>Billing blockers</strong>
              <p>Any unpaid daily bill that could stop the next travel day.</p>
            </div>
          </div>
        </section>
      </div>

      <section className="card">
        <div className="section-heading">
          <h3>Recent sessions</h3>
          <span>{isLoading ? "Refreshing..." : `${recentSessions.length} loaded`}</span>
        </div>

        {isLoading ? (
          <p className="lead">Loading your recent travel timeline...</p>
        ) : recentSessions.length === 0 ? (
          <div className="empty-bills-state">
            <strong>No sessions yet</strong>
            <p className="lead">Your checked-in journeys will start appearing here once you begin traveling.</p>
          </div>
        ) : (
          <div className="session-timeline">
            {recentSessions.map((session) => (
              <article key={session.id} className="session-card">
                <div className="session-card-header">
                  <div>
                    <p className="eyebrow">{formatStatus(session.status)}</p>
                    <h3>{formatDateTime(session.check_in_time)}</h3>
                  </div>
                  <strong className="bill-amount">{formatCurrency(session.total_fare)}</strong>
                </div>

                <div className="route-meta-grid">
                  <div>
                    <strong>{formatDistance(session.total_distance)}</strong>
                    <span>Distance</span>
                  </div>
                  <div>
                    <strong>{session.routes_used?.length || 0}</strong>
                    <span>Tracked routes</span>
                  </div>
                  <div>
                    <strong>{session.check_out_time ? formatDateTime(session.check_out_time) : "Still open"}</strong>
                    <span>Checkout</span>
                  </div>
                </div>

                <div className="feature-stack">
                  <div>
                    <strong>Check-in location</strong>
                    <p>{formatCoordinates(session.check_in_lat, session.check_in_lon)}</p>
                  </div>
                  {session.check_out_lat && session.check_out_lon ? (
                    <div>
                      <strong>Check-out location</strong>
                      <p>{formatCoordinates(session.check_out_lat, session.check_out_lon)}</p>
                    </div>
                  ) : null}
                  <p className="status-muted">Session ID: {session.id}</p>
                </div>
              </article>
            ))}
          </div>
        )}
      </section>
    </section>
  );
}

function formatDateTime(value) {
  return new Date(value).toLocaleString("en-IN");
}

function formatCurrency(value) {
  return new Intl.NumberFormat("en-IN", {
    style: "currency",
    currency: "INR",
    maximumFractionDigits: 2
  }).format(Number(value || 0));
}

function formatDistance(value) {
  return `${Number(value || 0).toFixed(1)} km`;
}

function formatStatus(value) {
  return String(value || "unknown").replaceAll("_", " ").replace(/^\w/, (char) => char.toUpperCase());
}

function formatCoordinates(lat, lon) {
  return `${Number(lat).toFixed(5)}, ${Number(lon).toFixed(5)}`;
}

export default SessionsPage;
