import { useState } from "react";
import ApiResult from "../components/ApiResult";
import { apiRequest } from "../lib/api";

function VehiclesPage() {
  const [form, setForm] = useState({
    vehicle_id: "demo-vehicle-id",
    route_id: "327H",
    trip_id: "demo-trip-id",
    latitude: "28.6139",
    longitude: "77.2090",
    start_moving_immediately: false
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
      <h2>Vehicle API</h2>
      <div className="card-grid">
        <label className="field"><span>route_id</span><input value={form.route_id} onChange={(e) => set("route_id", e.target.value)} /></label>
        <label className="field"><span>trip_id</span><input value={form.trip_id} onChange={(e) => set("trip_id", e.target.value)} /></label>
        <label className="field"><span>latitude</span><input value={form.latitude} onChange={(e) => set("latitude", e.target.value)} /></label>
        <label className="field"><span>longitude</span><input value={form.longitude} onChange={(e) => set("longitude", e.target.value)} /></label>
        <label className="field"><span>vehicle_id</span><input value={form.vehicle_id} onChange={(e) => set("vehicle_id", e.target.value)} /></label>
        <label className="checkbox"><input type="checkbox" checked={form.start_moving_immediately} onChange={(e) => set("start_moving_immediately", e.target.checked)} />start_moving_immediately</label>
      </div>
      <div className="toolbar">
        <button onClick={() => run(() => apiRequest("/vehicles/mock", { method: "POST", body: JSON.stringify({ route_id: form.route_id, trip_id: form.trip_id || undefined, latitude: Number(form.latitude), longitude: Number(form.longitude), start_moving_immediately: form.start_moving_immediately }) }))}>POST /vehicles/mock</button>
        <button onClick={() => run(() => apiRequest(`/vehicles/${form.vehicle_id}`))}>GET /vehicles/:vehicle_id</button>
      </div>
      <ApiResult data={data} error={error} />
    </section>
  );
}

export default VehiclesPage;
