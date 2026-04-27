import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../lib/auth";
import { apiRequest } from "../lib/api";

const PROFILE_TABS = [
  { id: "overview", label: "Overview" },
  { id: "tickets", label: "Tickets & Bills" },
  { id: "journeys", label: "Journey History" },
  { id: "account", label: "Account" }
];

function ProfilePage() {
  const { user } = useAuth();
  const [activeTab, setActiveTab] = useState("overview");
  const [activeSession, setActiveSession] = useState(null);
  const [recentSessions, setRecentSessions] = useState([]);
  const [pendingBills, setPendingBills] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let ignore = false;

    async function loadProfile() {
      if (!user?.id) {
        return;
      }

      try {
        setIsLoading(true);
        setError("");

        const [activePayload, sessionsPayload, billsPayload] = await Promise.all([
          apiRequest("/sessions/me/active"),
          apiRequest("/sessions/me?limit=6"),
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
          setError(loadError.message || "Unable to load your profile right now.");
        }
      } finally {
        if (!ignore) {
          setIsLoading(false);
        }
      }
    }

    loadProfile();
    return () => {
      ignore = true;
    };
  }, [user?.id]);

  const outstandingAmount = useMemo(
    () => pendingBills.reduce((sum, bill) => sum + Number(bill.total_fare || 0), 0),
    [pendingBills]
  );
  const completedSessions = recentSessions.filter((session) => session.status !== "active");

  return (
    <section className="route-page">
      <section className="profile-hero card">
        <div className="profile-hero-copy">
          <p className="eyebrow">Profile</p>
          <h2>Your personal travel area</h2>
          <p className="lead">
            Built around the end-goal rider flow: plan a journey, check departures, travel with one ticket, and keep billing and account actions in one calm place.
          </p>

          <div className="profile-status-grid">
            <div className="profile-status-tile">
              <span>Travel status</span>
              <strong>{activeSession ? "Ticket active" : "Ready to travel"}</strong>
            </div>
            <div className="profile-status-tile">
              <span>Pending bills</span>
              <strong>{pendingBills.length ? `${pendingBills.length} open` : "All paid"}</strong>
            </div>
            <div className="profile-status-tile">
              <span>Recent trips</span>
              <strong>{completedSessions.length}</strong>
            </div>
          </div>
        </div>

        <div className="profile-hero-panel">
          <strong>{user?.name}</strong>
          <span>{user?.email || user?.phone || "Signed in rider"}</span>
          <p>
            {activeSession
              ? "You already have an active travel session. Open Tickets to show your QR or finish the ride."
              : "Start with a route search, then move straight into a live ticket when you are ready to board."}
          </p>
          <div className="hero-actions">
            <Link className="primary-link" to={activeSession ? "/tickets" : "/plan"}>
              {activeSession ? "Open active ticket" : "Plan a journey"}
            </Link>
            <Link className="secondary-link" to="/departures">Check departures</Link>
          </div>
        </div>
      </section>

      <section className="profile-tab-strip card">
        <div className="profile-tabs" role="tablist" aria-label="Profile sections">
          {PROFILE_TABS.map((tab) => (
            <button
              key={tab.id}
              type="button"
              className={activeTab === tab.id ? "profile-tab active" : "profile-tab"}
              onClick={() => setActiveTab(tab.id)}
            >
              {tab.label}
            </button>
          ))}
        </div>
      </section>

      {error ? <section className="card"><p className="status-error">{error}</p></section> : null}

      {activeTab === "overview" ? (
        <div className="dashboard-grid">
          <section className="card">
            <div className="section-heading">
              <h3>Today&apos;s travel</h3>
            </div>
            {isLoading ? (
              <p className="lead">Loading your travel snapshot...</p>
            ) : (
              <div className="feature-stack">
                <div>
                  <strong>{activeSession ? "Ticket already running" : "No active ticket yet"}</strong>
                  <p>
                    {activeSession
                      ? `Started ${formatDateTime(activeSession.check_in_time)} and ready to show at inspection.`
                      : "Plan first, then create a QR ticket when you begin traveling."}
                  </p>
                </div>
                <div>
                  <strong>{pendingBills.length ? "Payment needed before your next travel day" : "Billing is clear"}</strong>
                  <p>
                    {pendingBills.length
                      ? `${formatCurrency(outstandingAmount)} is still waiting in daily bills.`
                      : "You do not have any unpaid daily bill right now."}
                  </p>
                </div>
                <div>
                  <strong>Useful next step</strong>
                  <p>
                    {activeSession
                      ? "Open Tickets to manage your live journey and QR."
                      : "Use Plan or Departures to line up your next trip."}
                  </p>
                </div>
              </div>
            )}
          </section>

          <section className="card">
            <div className="section-heading">
              <h3>Quick actions</h3>
            </div>
            <div className="profile-action-grid">
              <Link className="profile-action-card" to="/plan">
                <strong>Plan a trip</strong>
                <span>Search connections and save the best option before you check in.</span>
              </Link>
              <Link className="profile-action-card" to="/departures">
                <strong>Browse departures</strong>
                <span>See what is leaving nearby and jump into a route when the timing works.</span>
              </Link>
              <Link className="profile-action-card" to="/tickets">
                <strong>Open tickets</strong>
                <span>Show your QR, track a live session, and end the ride when you arrive.</span>
              </Link>
              <Link className="profile-action-card" to="/bills">
                <strong>Review bills</strong>
                <span>Pay once for the day and clear any blockers before the next trip.</span>
              </Link>
            </div>
          </section>
        </div>
      ) : null}

      {activeTab === "tickets" ? (
        <div className="dashboard-grid">
          <section className="card">
            <div className="section-heading">
              <h3>Ticket status</h3>
            </div>
            {isLoading ? (
              <p className="lead">Loading your ticket status...</p>
            ) : activeSession ? (
              <div className="feature-stack">
                <div>
                  <strong>Active QR session</strong>
                  <p>Checked in at {formatDateTime(activeSession.check_in_time)}.</p>
                </div>
                <div>
                  <strong>Session state</strong>
                  <p>{formatStatus(activeSession.status)}</p>
                </div>
                <div>
                  <strong>Current fare tracker</strong>
                  <p>{formatCurrency(activeSession.total_fare)}</p>
                </div>
                <div className="hero-actions">
                  <Link className="primary-link" to="/tickets">Open tickets</Link>
                </div>
              </div>
            ) : (
              <div className="empty-bills-state">
                <strong>No active ticket</strong>
                <p className="lead">Your next check-in will create a new QR ticket linked to your trip.</p>
              </div>
            )}
          </section>

          <section className="card">
            <div className="section-heading">
              <h3>Billing snapshot</h3>
            </div>
            {isLoading ? (
              <p className="lead">Loading billing details...</p>
            ) : (
              <div className="feature-stack">
                <div>
                  <strong>{pendingBills.length ? formatCurrency(outstandingAmount) : formatCurrency(0)}</strong>
                  <p>{pendingBills.length ? "Outstanding across pending daily bills." : "No unpaid bill at the moment."}</p>
                </div>
                <div>
                  <strong>{pendingBills.length ? "Travel may be blocked later" : "Free to start the next day"}</strong>
                  <p>
                    {pendingBills.length
                      ? "Settle the pending daily bill before your next check-in."
                      : "No billing blocker is currently attached to your account."}
                  </p>
                </div>
                <div className="hero-actions">
                  <Link className="primary-link" to="/bills">Open bills</Link>
                  <Link className="secondary-link" to="/tickets">Back to tickets</Link>
                </div>
              </div>
            )}
          </section>
        </div>
      ) : null}

      {activeTab === "journeys" ? (
        <section className="card">
          <div className="section-heading">
            <h3>Recent journeys</h3>
            <span>{isLoading ? "Refreshing..." : `${recentSessions.length} loaded`}</span>
          </div>

          {isLoading ? (
            <p className="lead">Loading your recent journeys...</p>
          ) : recentSessions.length === 0 ? (
            <div className="empty-bills-state">
              <strong>No journeys yet</strong>
              <p className="lead">Once you start traveling with tickets, your completed journeys will appear here.</p>
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
                      <span>Routes used</span>
                    </div>
                    <div>
                      <strong>{session.check_out_time ? formatDateTime(session.check_out_time) : "Still active"}</strong>
                      <span>Checkout</span>
                    </div>
                  </div>
                </article>
              ))}
            </div>
          )}
        </section>
      ) : null}

      {activeTab === "account" ? (
        <div className="dashboard-grid">
          <section className="card">
            <div className="section-heading">
              <h3>Account details</h3>
            </div>
            <div className="feature-stack">
              <div>
                <strong>Name</strong>
                <p>{user?.name || "Traveller rider"}</p>
              </div>
              <div>
                <strong>Primary contact</strong>
                <p>{user?.email || user?.phone || "No contact saved"}</p>
              </div>
              <div>
                <strong>Sign-in method</strong>
                <p>{formatProvider(user?.authProvider)}</p>
              </div>
            </div>
          </section>

          <section className="card">
            <div className="section-heading">
              <h3>Travel controls</h3>
            </div>
            <div className="feature-stack">
              <div>
                <strong>Planner-first journey flow</strong>
                <p>Use Plan to choose a connection, then Tickets to start a live ride with one QR.</p>
              </div>
              <div>
                <strong>Daily billing</strong>
                <p>Completed rides are grouped into one daily bill instead of charging each boarding separately.</p>
              </div>
              <div>
                <strong>Inspection-ready ticket</strong>
                <p>Your QR stays available throughout the ride, even while route tracking runs in the background.</p>
              </div>
            </div>
          </section>
        </div>
      ) : null}
    </section>
  );
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

function formatStatus(value) {
  return String(value || "unknown").replaceAll("_", " ").replace(/^\w/, (char) => char.toUpperCase());
}

function formatProvider(value) {
  if (!value) {
    return "Unknown";
  }

  if (value === "google") {
    return "Google";
  }

  if (value === "phone") {
    return "Phone number";
  }

  return formatStatus(value);
}

export default ProfilePage;
