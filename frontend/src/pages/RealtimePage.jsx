import { useState } from "react";
import ApiResult from "../components/ApiResult";
import { apiRequest } from "../lib/api";

function RealtimePage() {
  const [stopID, setStopID] = useState("49");
  const [tripID, setTripID] = useState("demo-trip-id");
  const [data, setData] = useState(null);
  const [error, setError] = useState("");

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
      <h2>Realtime API</h2>
      <div className="toolbar">
        <input placeholder="Stop ID" value={stopID} onChange={(e) => setStopID(e.target.value)} />
        <button onClick={() => run(() => apiRequest(`/realtime/stops/${stopID}`))}>GET /realtime/stops/:id</button>
      </div>
      <div className="toolbar">
        <input placeholder="Trip ID" value={tripID} onChange={(e) => setTripID(e.target.value)} />
        <button onClick={() => run(() => apiRequest(`/realtime/trips/${tripID}`))}>GET /realtime/trips/:id</button>
      </div>
      <ApiResult data={data} error={error} />
    </section>
  );
}

export default RealtimePage;
