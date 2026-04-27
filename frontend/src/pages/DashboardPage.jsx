import { useEffect, useMemo, useState } from "react";
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

function DashboardPage() {
  const { user } = useAuth();
  const activeJourney = getActiveJourney();
  const [activeSession, setActiveSession] = useState(() => getActiveSession());
  const [pendingBills, setPendingBills] = useState([]);
  const [recentSessions, setRecentSessions] = useState([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    let ignore = false;

    async function loadHomeState() {
      if (!user?.id) {
        return;
      }

      try {
        setIsLoading(true);

        const [activePayload, billsPayload, sessionsPayload] = await Promise.all([
          apiRequest("/sessions/me/active"),
          apiRequest("/bills/me/pending"),
          apiRequest("/sessions/me?limit=4")
        ]);

        if (ignore) {
          return;
        }

        const nextSession = activePayload.sessions?.[0] || null;
        setActiveSession(nextSession ? toSessionState(nextSession) : null);
        if (nextSession) {
          saveActiveSession(toSessionState(nextSession));
        }
        setPendingBills(billsPayload.bills || []);
        setRecentSessions(sessionsPayload.sessions || []);
      } catch {
        // Keep the local snapshot if the API refresh fails.
      } finally {
        if (!ignore) {
          setIsLoading(false);
        }
      }
    }

    loadHomeState();
    return () => {
      ignore = true;
    };
  }, [user?.id]);

  const outstandingAmount = useMemo(
    () => pendingBills.reduce((sum, bill) => sum + Number(bill.total_fare || 0), 0),
    [pendingBills]
  );
  const lastCompletedSession = recentSessions.find((session) => session.status !== "active");
  const heroTitle = activeSession
    ? "Your ticket is live and ready to show."
    : activeJourney
      ? "Your next trip is lined up."
      : "Travel in one connected flow.";
  const heroCopy = activeSession
    ? "Keep your QR ready during inspection while the session stays active until checkout."
    : activeJourney
      ? "You already picked a journey. Move into Tickets when you are ready to start traveling."
      : "Plan a route, check departures, start one live ticket, and settle the day as a single bill.";

  return (
    <section className="route-page">
      <section className="home-hero card">
        <div className="home-hero-copy">
          <p className="eyebrow">Home</p>
          <h2>{heroTitle}</h2>
          <p className="lead">{heroCopy}</p>

          <div className="hero-actions">
            <Link className="primary-link" to={activeSession ? "/tickets" : "/plan"}>
              {activeSession ? "Open active ticket" : "Plan a journey"}
            </Link>
            <Link className="secondary-link" to="/departures">Check departures</Link>
          </div>

          <div className="home-signal-row">
            <div className="home-signal-pill">
              <span>Status</span>
              <strong>{activeSession ? "Traveling now" : "Ready to start"}</strong>
            </div>
            <div className="home-signal-pill">
              <span>Billing</span>
              <strong>{pendingBills.length ? `${pendingBills.length} open` : "All clear"}</strong>
            </div>
            <div className="home-signal-pill">
              <span>Recent trips</span>
              <strong>{recentSessions.filter((session) => session.status !== "active").length}</strong>
            </div>
          </div>
        </div>

        <div className="home-hero-panel">
          <div className="section-heading">
            <h3>Right now</h3>
            {activeSession || activeJourney ? (
              <button
                type="button"
                className="ghost-button"
                onClick={() => {
                  clearActiveSession();
                  clearActiveJourney();
                  window.location.reload();
                }}
              >
                Clear local state
              </button>
            ) : null}
          </div>

          {activeSession ? (
            <div className="ticket-preview home-ticket-preview">
              <QRCodePreview value={activeSession.qrCode} size={170} />
              <div className="feature-stack">
                <div>
                  <strong>Checked in</strong>
                  <p>{activeSession.checkInTimeLabel}</p>
                </div>
                <div>
                  <strong>Starting point</strong>
                  <p>{activeSession.checkInLocationLabel}</p>
                </div>
                <div>
                  <strong>Next step</strong>
                  <p>Keep the QR handy and check out when you finish.</p>
                </div>
              </div>
            </div>
          ) : activeJourney ? (
            <div className="feature-stack">
              <div>
                <strong>Saved journey</strong>
                <p>{activeJourney.fromStopName} to {activeJourney.toStopName}</p>
              </div>
              <div>
                <strong>Selected route</strong>
                <p>{activeJourney.routeSummary || "Journey planned"}</p>
              </div>
              <div>
                <strong>Next step</strong>
                <p>Open Tickets and check in to turn this into a live trip.</p>
              </div>
            </div>
          ) : (
            <div className="feature-stack">
              <div>
                <strong>No trip active</strong>
                <p>Start with a route search or open departures near the stop you are heading to.</p>
              </div>
            </div>
          )}
        </div>
      </section>

      <div className="dashboard-grid">
        <section className="card">
          <div className="section-heading">
            <h3>Your travel snapshot</h3>
          </div>
          <div className="route-meta-grid">
            <div>
              <strong>{activeSession ? "Active" : "Idle"}</strong>
              <span>Ticket state</span>
            </div>
            <div>
              <strong>{formatCurrency(outstandingAmount)}</strong>
              <span>Pending bills</span>
            </div>
            <div>
              <strong>{lastCompletedSession ? formatDistance(lastCompletedSession.total_distance) : "0.0 km"}</strong>
              <span>Last trip distance</span>
            </div>
            <div>
              <strong>{lastCompletedSession ? formatDateTime(lastCompletedSession.check_in_time) : "No recent trip"}</strong>
              <span>Last activity</span>
            </div>
          </div>
          {isLoading ? <p className="status-muted">Refreshing your latest account activity…</p> : null}
        </section>

        <section className="card">
          <div className="section-heading">
            <h3>Start from what you need</h3>
          </div>
          <div className="home-action-grid">
            <Link className="profile-action-card" to="/plan">
              <strong>Plan a route</strong>
              <span>Compare connections, save one, and move into ticketing with less friction.</span>
            </Link>
            <Link className="profile-action-card" to="/departures">
              <strong>See departures</strong>
              <span>Jump straight into stop-level timings when you already know where you are.</span>
            </Link>
            <Link className="profile-action-card" to="/tickets">
              <strong>Manage ticket</strong>
              <span>Show your QR, track the live ride, and finish the session when you arrive.</span>
            </Link>
            <Link className="profile-action-card" to="/profile">
              <strong>Open profile</strong>
              <span>Review bills, journey history, and account details in one personal area.</span>
            </Link>
          </div>
        </section>

        <section className="card">
          <div className="section-heading">
            <h3>How Traveller works</h3>
          </div>
          <div className="feature-stack">
            <div>
              <strong>1. Plan before you board</strong>
              <p>Choose a connection like a rider, with stops, transfers, and timing that match the trip you want.</p>
            </div>
            <div>
              <strong>2. Carry one live ticket</strong>
              <p>Check in once and keep the QR ready while the session captures the ride in the background.</p>
            </div>
            <div>
              <strong>3. Settle the day once</strong>
              <p>Completed travel rolls into daily billing, so payment feels like one clean end-of-day step.</p>
            </div>
          </div>
        </section>

        <section className="card">
          <div className="section-heading">
            <h3>Before your next ride</h3>
          </div>
          <div className="feature-stack">
            <div>
              <strong>{pendingBills.length ? "Pay your pending bill" : "Billing is clear"}</strong>
              <p>
                {pendingBills.length
                  ? "A pending bill can block the next travel day, so clear it from Bills before you check in again."
                  : "You do not have any unpaid daily bill blocking the next ride."}
              </p>
            </div>
            <div>
              <strong>{activeJourney ? "Journey already saved" : "No journey saved yet"}</strong>
              <p>
                {activeJourney
                  ? "Your last selected connection is ready in local state and can feed straight into ticket creation."
                  : "Saving a planned connection makes the move into ticketing much smoother."}
              </p>
            </div>
            <div className="hero-actions">
              <Link className="primary-link" to={pendingBills.length ? "/bills" : "/plan"}>
                {pendingBills.length ? "Review bills" : "Start planning"}
              </Link>
            </div>
          </div>
        </section>
      </div>
    </section>
  );
}

function toSessionState(session) {
  return {
    id: session.id,
    qrCode: session.qr_code,
    checkInTimeLabel: formatDateTime(session.check_in_time),
    checkInLocationLabel: session.check_in_stop_id || "Location captured at check-in"
  };
}

function formatDateTime(value) {
  if (!value) {
    return "Not available";
  }

  return new Date(value).toLocaleString("en-IN", {
    day: "numeric",
    month: "short",
    hour: "numeric",
    minute: "2-digit"
  });
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

export default DashboardPage;
