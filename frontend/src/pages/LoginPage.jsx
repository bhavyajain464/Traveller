import { useState } from "react";
import { Navigate, useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "../lib/auth";

function LoginPage() {
  const { isAuthenticated, login } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [form, setForm] = useState({
    name: "Bhavya Jain",
    phone: "+91 98765 43210"
  });
  const [error, setError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  if (isAuthenticated) {
    return <Navigate to="/" replace />;
  }

  const onSubmit = async (event) => {
    event.preventDefault();

    if (!form.name.trim() || !form.phone.trim()) {
      setError("Name and phone are required.");
      return;
    }

    try {
      setIsSubmitting(true);
      setError("");
      await login(form);
      navigate(location.state?.from || "/", { replace: true });
    } catch (submitError) {
      setError(submitError.message || "Login failed.");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <main className="login-shell">
      <section className="login-hero">
        <p className="eyebrow">Traveller</p>
        <h1>Plan, board, and travel with one QR.</h1>
        <p className="login-copy">
          This frontend-first prototype mirrors the core SBB flow: sign in, plan a journey, generate a travel token, and keep your active pass ready while you move.
        </p>
        <div className="login-highlights">
          <div className="card">
            <strong>Stop-aware planning</strong>
            <span>Search stations and stops, then plan directly from the GTFS-backed API.</span>
          </div>
          <div className="card">
            <strong>Travel QR</strong>
            <span>Generate a frontend travel ticket now, then connect it to backend sessions and billing next.</span>
          </div>
        </div>
      </section>

      <section className="login-panel card">
        <div>
          <p className="eyebrow">Sign In</p>
          <h2>Continue to your rider app</h2>
          <p className="lead">Local auth for now. We can wire real backend login later.</p>
        </div>

        <form className="login-form" onSubmit={onSubmit}>
          <label className="field">
            <span>Full name</span>
            <input
              value={form.name}
              onChange={(event) => setForm((state) => ({ ...state, name: event.target.value }))}
            />
          </label>

          <label className="field">
            <span>Phone number</span>
            <input
              value={form.phone}
              onChange={(event) => setForm((state) => ({ ...state, phone: event.target.value }))}
            />
          </label>

          {error ? <p className="status-error">{error}</p> : null}

          <button type="submit" className="primary-button" disabled={isSubmitting}>
            {isSubmitting ? "Signing in..." : "Log In"}
          </button>
        </form>
      </section>
    </main>
  );
}

export default LoginPage;
