import { useState } from "react";
import ApiResult from "../components/ApiResult";
import { apiRequest } from "../lib/api";

function BoardingsPage() {
  const [form, setForm] = useState({
    session_id: "demo-session-id",
    qr_code: "demo-qr-code",
    route_id: "327H",
    boarding_id: "demo-boarding-id",
    latitude: "28.6139",
    longitude: "77.2090",
    boarding_stop_id: "3280",
    alighting_stop_id: "3282"
  });
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
      <h2>Route Boardings API</h2>
      <div className="card-grid">
        {Object.entries(form).map(([key, value]) => (
          <label key={key} className="field">
            <span>{key}</span>
            <input value={value} onChange={(e) => set(key, e.target.value)} />
          </label>
        ))}
      </div>
      <div className="toolbar">
        <button onClick={() => run(() => apiRequest("/boardings/board", { method: "POST", body: JSON.stringify({ session_id: form.session_id || undefined, qr_code: form.qr_code || undefined, route_id: form.route_id, boarding_stop_id: form.boarding_stop_id || undefined, latitude: Number(form.latitude), longitude: Number(form.longitude) }) }))}>POST /boardings/board</button>
        <button onClick={() => run(() => apiRequest("/boardings/auto-board", { method: "POST", body: JSON.stringify({ session_id: form.session_id || undefined, qr_code: form.qr_code || undefined, latitude: Number(form.latitude), longitude: Number(form.longitude) }) }))}>POST /boardings/auto-board</button>
        <button onClick={() => run(() => apiRequest("/boardings/alight", { method: "POST", body: JSON.stringify({ boarding_id: form.boarding_id, alighting_stop_id: form.alighting_stop_id || undefined, latitude: Number(form.latitude), longitude: Number(form.longitude) }) }))}>POST /boardings/alight</button>
        <button onClick={() => run(() => apiRequest("/boardings/continuous-location", { method: "POST", body: JSON.stringify({ session_id: form.session_id || undefined, qr_code: form.qr_code || undefined, latitude: Number(form.latitude), longitude: Number(form.longitude) }) }))}>POST /boardings/continuous-location</button>
      </div>
      <div className="toolbar">
        <button onClick={() => run(() => apiRequest(`/boardings/sessions/${form.session_id}`))}>GET /boardings/sessions/:session_id</button>
        <button onClick={() => run(() => apiRequest(`/boardings/sessions/${form.session_id}/active`))}>GET /boardings/sessions/:session_id/active</button>
      </div>
      <ApiResult data={data} error={error} />
    </section>
  );
}

export default BoardingsPage;
