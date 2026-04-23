import { useState } from "react";
import ApiResult from "../components/ApiResult";
import { apiRequest } from "../lib/api";

const initial = {
  user_id: "demo-user-id",
  latitude: "28.6139",
  longitude: "77.2090",
  stop_id: "49",
  session_id: "demo-session-id",
  qr_code: "demo-qr-code",
  route_id: "327H"
};

function SessionsPage() {
  const [form, setForm] = useState(initial);
  const [data, setData] = useState(null);
  const [error, setError] = useState("");

  const set = (key, value) => setForm((s) => ({ ...s, [key]: value }));
  const run = async (req) => {
    try {
      setError("");
      setData(await req());
    } catch (err) {
      setError(err.message);
    }
  };

  return (
    <section>
      <h2>Journey Sessions API</h2>
      <div className="card-grid">
        {Object.keys(initial).map((key) => (
          <label key={key} className="field">
            <span>{key}</span>
            <input value={form[key]} onChange={(e) => set(key, e.target.value)} />
          </label>
        ))}
      </div>
      <div className="toolbar">
        <button onClick={() => run(() => apiRequest("/sessions/checkin", { method: "POST", body: JSON.stringify({ user_id: form.user_id, latitude: Number(form.latitude), longitude: Number(form.longitude), stop_id: form.stop_id || undefined }) }))}>POST /sessions/checkin</button>
        <button onClick={() => run(() => apiRequest("/sessions/checkout", { method: "POST", body: JSON.stringify({ session_id: form.session_id || undefined, qr_code: form.qr_code || undefined, latitude: Number(form.latitude), longitude: Number(form.longitude), stop_id: form.stop_id || undefined }) }))}>POST /sessions/checkout</button>
        <button onClick={() => run(() => apiRequest("/sessions/validate-qr", { method: "POST", body: JSON.stringify({ qr_code: form.qr_code, route_id: form.route_id }) }))}>POST /sessions/validate-qr</button>
        <button onClick={() => run(() => apiRequest(`/sessions/users/${form.user_id}/active`))}>GET /sessions/users/:user_id/active</button>
      </div>
      <ApiResult data={data} error={error} />
    </section>
  );
}

export default SessionsPage;
