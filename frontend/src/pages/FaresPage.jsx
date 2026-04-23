import { useState } from "react";
import ApiResult from "../components/ApiResult";
import { apiRequest } from "../lib/api";

function FaresPage() {
  const [routeID, setRouteID] = useState("327H");
  const [fromStop, setFromStop] = useState("3280");
  const [toStop, setToStop] = useState("3282");
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
      <h2>Fare API</h2>
      <div className="toolbar">
        <input placeholder="Route ID" value={routeID} onChange={(e) => setRouteID(e.target.value)} />
        <input placeholder="From stop ID" value={fromStop} onChange={(e) => setFromStop(e.target.value)} />
        <input placeholder="To stop ID" value={toStop} onChange={(e) => setToStop(e.target.value)} />
      </div>
      <div className="toolbar">
        <button onClick={() => run(() => apiRequest(`/fares/calculate?route_id=${routeID}&from_stop_id=${fromStop}&to_stop_id=${toStop}`))}>GET /fares/calculate</button>
        <button onClick={() => run(() => apiRequest(`/fares/routes/${routeID}?from_stop_id=${fromStop}&to_stop_id=${toStop}`))}>GET /fares/routes/:id</button>
      </div>
      <ApiResult data={data} error={error} />
    </section>
  );
}

export default FaresPage;
