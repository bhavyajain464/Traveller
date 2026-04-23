import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../lib/auth";
import { apiRequest } from "../lib/api";
import { getActiveSession } from "../lib/tickets";

const PAYMENT_METHOD = "upi";

function BillsPage() {
  const { user } = useAuth();
  const activeSession = getActiveSession();
  const [pendingBills, setPendingBills] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [payingBillID, setPayingBillID] = useState("");
  const [error, setError] = useState("");
  const [successMessage, setSuccessMessage] = useState("");

  useEffect(() => {
    let ignore = false;

    async function loadBills() {
      if (!user?.id) {
        return;
      }

      try {
        setIsLoading(true);
        setError("");
        const payload = await apiRequest("/bills/me/pending");
        if (!ignore) {
          setPendingBills(payload.bills || []);
        }
      } catch (loadError) {
        if (!ignore) {
          setError(loadError.message || "Unable to load your bills.");
        }
      } finally {
        if (!ignore) {
          setIsLoading(false);
        }
      }
    }

    loadBills();
    return () => {
      ignore = true;
    };
  }, [user?.id]);

  async function payBill(bill) {
    try {
      setPayingBillID(bill.id);
      setError("");
      setSuccessMessage("");

      await apiRequest(`/bills/${encodeURIComponent(bill.id)}/pay`, {
        method: "POST",
        body: JSON.stringify({
          payment_id: buildPaymentID(bill.id),
          payment_method: PAYMENT_METHOD
        })
      });

      setPendingBills((current) => current.filter((item) => item.id !== bill.id));
      setSuccessMessage(`Bill for ${formatBillDate(bill.bill_date)} marked as paid.`);
    } catch (payError) {
      setError(payError.message || "Payment failed.");
    } finally {
      setPayingBillID("");
    }
  }

  const totalOutstanding = pendingBills.reduce(
    (sum, bill) => sum + Number(bill.total_fare || 0),
    0
  );

  return (
    <section className="route-page">
      <div className="page-header">
        <div>
          <p className="eyebrow">Billing</p>
          <h2>Review and pay your daily bills</h2>
          <p className="lead">
            Each day rolls up your tracked rides into one amount. Pay the outstanding bill once and you&apos;re ready for the next travel day.
          </p>
        </div>
      </div>

      {error ? <section className="card"><p className="status-error">{error}</p></section> : null}
      {successMessage ? <section className="card"><p className="status-success">{successMessage}</p></section> : null}

      <section className="hero-panel">
        <div>
          <p className="eyebrow">Outstanding</p>
          <h3>{formatCurrency(totalOutstanding)}</h3>
          <p className="lead">
            {pendingBills.length > 0
              ? `${pendingBills.length} unpaid daily bill${pendingBills.length === 1 ? "" : "s"} linked to your account.`
              : "No unpaid bills right now."}
          </p>
          <div className="hero-actions">
            <Link className="primary-link" to="/tickets">
              {activeSession ? "Back to active ticket" : "Go to tickets"}
            </Link>
            <Link className="secondary-link" to="/profile">Open profile</Link>
          </div>
        </div>

        <div className="card api-card">
          <span>Status</span>
          <strong>{pendingBills.length > 0 ? "Payment required before next travel day" : "All caught up"}</strong>
        </div>
      </section>

      <div className="dashboard-grid">
        <section className="card">
          <div className="section-heading">
            <h3>How billing works</h3>
          </div>
          <div className="feature-stack">
            <div>
              <strong>1. Track rides through the day</strong>
              <p>Your QR ticket stays active while boarding and alighting events are attached to the same session.</p>
            </div>
            <div>
              <strong>2. Daily bill is generated once</strong>
              <p>All completed travel for the day is grouped into a single payable amount.</p>
            </div>
            <div>
              <strong>3. Pay once to keep traveling</strong>
              <p>If a bill is still pending, the next day&apos;s check-in is blocked until it&apos;s settled.</p>
            </div>
          </div>
        </section>

        <section className="card">
          <div className="section-heading">
            <h3>Travel status</h3>
          </div>
          {activeSession ? (
            <div className="feature-stack">
              <div>
                <strong>Ticket currently active</strong>
                <p>Your ticket is still open. Finish the ride in Tickets when you&apos;re done so billing can close out cleanly.</p>
              </div>
              <p className="status-muted">Session ID: {activeSession.id}</p>
            </div>
          ) : (
            <div className="feature-stack">
              <div>
                <strong>No active ticket</strong>
                <p>You can start a new ticket from Tickets once any pending daily bill is paid.</p>
              </div>
            </div>
          )}
        </section>
      </div>

      <section className="card">
        <div className="section-heading">
          <h3>Pending bills</h3>
          <span>{isLoading ? "Refreshing..." : `${pendingBills.length} open`}</span>
        </div>

        {isLoading ? (
          <p className="lead">Loading your billing summary...</p>
        ) : pendingBills.length === 0 ? (
          <div className="empty-bills-state">
            <strong>Nothing to pay</strong>
            <p className="lead">
              Your account has no pending daily bills right now.
            </p>
          </div>
        ) : (
          <div className="bills-list">
            {pendingBills.map((bill) => (
              <article key={bill.id} className="bill-card">
                <div className="bill-card-header">
                  <div>
                    <p className="eyebrow">Bill Date</p>
                    <h3>{formatBillDate(bill.bill_date)}</h3>
                  </div>
                  <strong className="bill-amount">{formatCurrency(bill.total_fare)}</strong>
                </div>

                <div className="route-meta-grid">
                  <div>
                    <strong>{bill.total_journeys}</strong>
                    <span>Journeys</span>
                  </div>
                  <div>
                    <strong>{formatDistance(bill.total_distance)}</strong>
                    <span>Distance</span>
                  </div>
                  <div>
                    <strong>{bill.status}</strong>
                    <span>Status</span>
                  </div>
                </div>

                <div className="toolbar">
                  <button
                    type="button"
                    className="primary-button"
                    disabled={payingBillID === bill.id}
                    onClick={() => payBill(bill)}
                  >
                    {payingBillID === bill.id ? "Paying..." : "Pay bill"}
                  </button>
                </div>
              </article>
            ))}
          </div>
        )}
      </section>
    </section>
  );
}

function buildPaymentID(billID) {
  return `pay-${billID.slice(0, 8)}-${Date.now()}`;
}

function formatBillDate(value) {
  return new Date(value).toLocaleDateString("en-IN", {
    weekday: "short",
    day: "numeric",
    month: "short",
    year: "numeric"
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

export default BillsPage;
