import { Link } from "react-router-dom";
import QRCodePreview from "../components/QRCodePreview";
import { API_BASE_URL } from "../lib/api";
import { clearActiveJourney, clearActiveSession, getActiveJourney, getActiveSession } from "../lib/tickets";

function DashboardPage() {
  const activeJourney = getActiveJourney();
  const activeSession = getActiveSession();

  return (
    <section className="route-page">
      <section className="hero-panel">
        <div>
          <p className="eyebrow">Ready To Travel</p>
          <h2>Check in to generate your live travel QR.</h2>
          <p className="lead">
            Your QR now exists only while a journey session is active. It is created on check-in with your time and location, then destroyed on checkout once billing can be finalized.
          </p>
          <div className="hero-actions">
            <Link className="primary-link" to="/plan">Plan a journey</Link>
            <Link className="secondary-link" to="/tickets">Check in or out</Link>
          </div>
        </div>
        <div className="card api-card">
          <span>Connected backend</span>
          <strong>{API_BASE_URL}</strong>
        </div>
      </section>

      <div className="dashboard-grid">
        <section className="card">
          <div className="section-heading">
            <h3>How this flow works</h3>
          </div>
          <div className="feature-stack">
            <div>
              <strong>1. Choose origin and destination</strong>
              <p>Search stations and stops like a rider, not like a database operator.</p>
            </div>
            <div>
              <strong>2. Pick the best option</strong>
              <p>Review transfers, route legs, and timing before you commit.</p>
            </div>
            <div>
              <strong>3. Check out to end it</strong>
              <p>The live QR is destroyed when your trip ends and the ride can be billed.</p>
            </div>
          </div>
        </section>

        <section className="card planner-health-card">
          <div className="section-heading">
            <h3>Planner architecture</h3>
          </div>
          <div className="planner-status-grid">
            <div>
              <span>Timetable</span>
              <strong>In memory</strong>
            </div>
            <div>
              <span>Search model</span>
              <strong>Round based</strong>
            </div>
            <div>
              <span>Interchanges</span>
              <strong>Footpath edges</strong>
            </div>
            <div>
              <span>Fallback</span>
              <strong>SQL adapter</strong>
            </div>
          </div>
          <p className="status-muted">
            The rider UI now follows the same transition path as the backend plan: snapshot planner first, richer routing behavior next, then end-goal APIs and validation.
          </p>
        </section>

        <section className="card active-ticket-card">
          <div className="section-heading">
            <h3>Active travel status</h3>
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
            <div className="ticket-preview">
              <QRCodePreview value={activeSession.qrCode} size={180} />
              <div>
                <strong>Checked in and traveling</strong>
                <p>{activeSession.checkInTimeLabel}</p>
                <p>{activeSession.checkInLocationLabel}</p>
                <p className="status-muted">QR is active until checkout.</p>
              </div>
            </div>
          ) : activeJourney ? (
            <div>
              <strong>Journey ready to start</strong>
              <p>{activeJourney.fromStopName} to {activeJourney.toStopName}</p>
              <p>{activeJourney.routeSummary || "Journey planned"}</p>
              <p className="status-muted">Go to Tickets and check in to generate the QR.</p>
            </div>
          ) : (
            <p className="lead">No active journey. Plan a trip, then check in to create a live QR.</p>
          )}
        </section>
      </div>
    </section>
  );
}

export default DashboardPage;
