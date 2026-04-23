import { useEffect, useRef, useState } from "react";
import { Navigate, useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "../lib/auth";

const GOOGLE_CLIENT_ID = import.meta.env.VITE_GOOGLE_CLIENT_ID || "";
const GOOGLE_SCRIPT_ID = "google-identity-services";

function LoginPage() {
  const { isAuthenticated, isLoading, loginWithGoogle } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const buttonRef = useRef(null);
  const retryTimerRef = useRef(null);
  const [error, setError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [buttonRendered, setButtonRendered] = useState(false);

  useEffect(() => {
    if (!GOOGLE_CLIENT_ID || !buttonRef.current) {
      return undefined;
    }

    let cancelled = false;

    const scheduleRetry = () => {
      window.clearTimeout(retryTimerRef.current);
      retryTimerRef.current = window.setTimeout(() => {
        attemptRender();
      }, 300);
    };

    const attemptRender = () => {
      if (cancelled) {
        return;
      }

      if (!window.google?.accounts?.id || !buttonRef.current) {
        scheduleRetry();
        return;
      }

      window.google.accounts.id.initialize({
        client_id: GOOGLE_CLIENT_ID,
        callback: async (response) => {
          try {
            setIsSubmitting(true);
            setError("");
            await loginWithGoogle(response.credential);
            navigate(location.state?.from || "/", { replace: true });
          } catch (submitError) {
            setError(submitError.message || "Google sign-in failed.");
          } finally {
            setIsSubmitting(false);
          }
        }
      });

      buttonRef.current.innerHTML = "";
      window.google.accounts.id.renderButton(buttonRef.current, {
        type: "standard",
        theme: "outline",
        size: "large",
        shape: "pill",
        text: "continue_with",
        width: 320
      });
      setButtonRendered(true);
    };

    const existingScript = document.getElementById(GOOGLE_SCRIPT_ID);
    if (existingScript) {
      attemptRender();
      return () => {
        cancelled = true;
        window.clearTimeout(retryTimerRef.current);
      };
    }

    const script = document.createElement("script");
    script.id = GOOGLE_SCRIPT_ID;
    script.src = "https://accounts.google.com/gsi/client";
    script.async = true;
    script.defer = true;
    script.onload = attemptRender;
    script.onerror = () => {
      if (!cancelled) {
        setError("Unable to load Google sign-in.");
      }
    };
    document.head.appendChild(script);

    return () => {
      cancelled = true;
      window.clearTimeout(retryTimerRef.current);
    };
  }, [location.state?.from, loginWithGoogle, navigate]);

  if (isAuthenticated) {
    return <Navigate to="/" replace />;
  }

  if (isLoading) {
    return (
      <main className="login-shell">
        <section className="login-panel card">
          <p className="lead">Restoring your session...</p>
        </section>
      </main>
    );
  }

  return (
    <main className="login-shell">
      <section className="login-hero">
        <p className="eyebrow">Traveller</p>
        <h1>Plan, board, and travel with one QR.</h1>
        <p className="login-copy">
          Sign in with Google to connect your rider identity to the backend, generate live journey QR codes, and keep the door open for phone-based auth later.
        </p>
        <div className="login-highlights">
          <div className="card">
            <strong>Real backend session</strong>
            <span>Google identity is verified server-side before Traveller issues its own app session token.</span>
          </div>
          <div className="card">
            <strong>Phone-ready foundation</strong>
            <span>The user model now supports Google today and phone login as a follow-up auth provider.</span>
          </div>
        </div>
      </section>

      <section className="login-panel card">
        <div>
          <p className="eyebrow">Sign In</p>
          <h2>Continue to your rider app</h2>
          <p className="lead">Google SSO is live for this build.</p>
        </div>

        {!GOOGLE_CLIENT_ID ? (
          <p className="status-error">
            Missing <code>VITE_GOOGLE_CLIENT_ID</code>. Add your Google OAuth web client ID to enable sign-in.
          </p>
        ) : (
          <div className="login-form">
            <div ref={buttonRef} />
            {!buttonRendered ? <p className="status-muted">Loading Google sign-in...</p> : null}
            {isSubmitting ? <p className="status-muted">Signing you in...</p> : null}
          </div>
        )}

        {error ? <p className="status-error">{error}</p> : null}
      </section>
    </main>
  );
}

export default LoginPage;
