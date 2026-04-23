import { NavLink, Outlet } from "react-router-dom";
import { useAuth } from "../lib/auth";
import { clearActiveJourney } from "../lib/tickets";

const links = [
  { to: "/", label: "Home" },
  { to: "/plan", label: "Plan" },
  { to: "/departures", label: "Departures" },
  { to: "/tickets", label: "Tickets" },
  { to: "/profile", label: "Profile" }
];

function Layout() {
  const { user, logout } = useAuth();

  return (
    <div className="layout">
      <aside className="sidebar">
        <h1>Traveller Rider</h1>
        <p>Plan, ticket, and travel through one connected flow.</p>
        <div className="sidebar-user card">
          <strong>{user?.name}</strong>
          <span>{user?.email || user?.phone || "Signed in"}</span>
          <button
            type="button"
            className="ghost-button"
            onClick={() => {
              clearActiveJourney();
              logout();
            }}
          >
            Log out
          </button>
        </div>
        <nav>
          {links.map((link) => (
            <NavLink
              key={link.to}
              to={link.to}
              className={({ isActive }) => (isActive ? "nav-link active" : "nav-link")}
              end={link.to === "/"}
            >
              {link.label}
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-footnote">
          <span>Advanced operational tools live under Profile.</span>
        </div>
      </aside>
      <main className="content">
        <Outlet />
      </main>
    </div>
  );
}

export default Layout;
