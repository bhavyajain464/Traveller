import { Link } from "react-router-dom";
import { useAuth } from "../lib/auth";

function ProfilePage() {
  const { user } = useAuth();

  return (
    <section className="route-page">
      <div className="page-header">
        <div>
          <h2>Profile</h2>
          <p className="lead">
            Your account, rider shortcuts, and the advanced tools that stay out of the main travel flow.
          </p>
        </div>
      </div>

      <div className="dashboard-grid">
        <section className="card">
          <p className="eyebrow">Account</p>
          <h3>{user?.name}</h3>
          <p>{user?.email || user?.phone || "No primary contact available"}</p>
          <p className="status-muted">Signed in with {user?.authProvider || "unknown"} and ready for backend-backed sessions.</p>
        </section>

        <section className="card">
          <p className="eyebrow">Travel shortcuts</p>
          <div className="feature-stack">
            <div>
              <strong>Plan your next trip</strong>
              <p><Link to="/plan">Open trip planner</Link></p>
            </div>
            <div>
              <strong>Check station departures</strong>
              <p><Link to="/departures">Browse stops and departures</Link></p>
            </div>
            <div>
              <strong>See your active QR</strong>
              <p><Link to="/tickets">Open tickets</Link></p>
            </div>
          </div>
        </section>
      </div>

      <section className="card">
        <div className="section-heading">
          <h3>Advanced tools</h3>
          <span>Hidden from the main rider nav</span>
        </div>
        <div className="tool-links">
          <Link to="/routes">Routes</Link>
          <Link to="/fares">Fares</Link>
          <Link to="/sessions">Sessions</Link>
          <Link to="/boardings">Boardings</Link>
          <Link to="/bills">Bills</Link>
          <Link to="/realtime">Realtime</Link>
          <Link to="/vehicles">Vehicles</Link>
        </div>
      </section>
    </section>
  );
}

export default ProfilePage;
