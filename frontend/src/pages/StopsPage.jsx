import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { apiRequest } from "../lib/api";

function StopsPage() {
  const [query, setQuery] = useState("New Delhi");
  const [submittedQuery, setSubmittedQuery] = useState("New Delhi");
  const [stopID, setStopID] = useState("");
  const [stops, setStops] = useState([]);
  const [stopsLoading, setStopsLoading] = useState(false);
  const [stopsError, setStopsError] = useState("");
  const [selectedStop, setSelectedStop] = useState(null);
  const [selectedStopError, setSelectedStopError] = useState("");
  const [departures, setDepartures] = useState([]);
  const [departuresLoading, setDeparturesLoading] = useState(false);
  const [departuresError, setDeparturesError] = useState("");

  useEffect(() => {
    let ignore = false;

    async function loadStops() {
      setStopsLoading(true);
      setStopsError("");

      try {
        const path = submittedQuery.trim()
          ? `/stops/search?q=${encodeURIComponent(submittedQuery.trim())}&limit=20`
          : "/stops?limit=20";
        const payload = await apiRequest(path);
        if (!ignore) {
          setStops(payload.stops || []);
          if (!stopID && payload.stops?.length) {
            setStopID(payload.stops[0].id);
          }
        }
      } catch (err) {
        if (!ignore) {
          setStops([]);
          setStopsError(err.message);
        }
      } finally {
        if (!ignore) {
          setStopsLoading(false);
        }
      }
    }

    loadStops();

    return () => {
      ignore = true;
    };
  }, [submittedQuery]);

  useEffect(() => {
    let ignore = false;

    async function loadStopDetail() {
      if (!stopID) {
        setSelectedStop(null);
        setDepartures([]);
        return;
      }

      setSelectedStopError("");
      setDeparturesError("");
      setDeparturesLoading(true);

      try {
        const [stopPayload, departuresPayload] = await Promise.all([
          apiRequest(`/stops/${encodeURIComponent(stopID)}`),
          apiRequest(`/stops/${encodeURIComponent(stopID)}/departures?limit=12`)
        ]);

        if (!ignore) {
          setSelectedStop(stopPayload);
          setDepartures(departuresPayload.departures || []);
        }
      } catch (err) {
        if (!ignore) {
          setSelectedStop(null);
          setDepartures([]);
          setSelectedStopError(err.message);
          setDeparturesError(err.message);
        }
      } finally {
        if (!ignore) {
          setDeparturesLoading(false);
        }
      }
    }

    loadStopDetail();

    return () => {
      ignore = true;
    };
  }, [stopID]);

  return (
    <section className="route-page">
      <div className="page-header">
        <div>
          <h2>Stops</h2>
          <p className="lead">
            Browse stops, inspect upcoming departures, and jump from each departure into the matching route detail view.
          </p>
        </div>
      </div>

      <div className="route-shell">
        <aside className="route-list-panel card">
          <form
            className="toolbar"
            onSubmit={(event) => {
              event.preventDefault();
              setSubmittedQuery(query);
            }}
          >
            <input
              placeholder="Search by stop name or code"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
            <button type="submit">Search</button>
          </form>

          {stopsError ? <p className="status-error">{stopsError}</p> : null}
          {stopsLoading ? <p className="status-muted">Loading stops…</p> : null}

          <div className="route-list">
            {stops.map((stop) => (
              <button
                key={stop.id}
                type="button"
                className={stop.id === stopID ? "route-list-item active" : "route-list-item"}
                onClick={() => setStopID(stop.id)}
              >
                <strong>{stop.name}</strong>
                <span>{stop.code || stop.id}</span>
                <small>{stop.latitude.toFixed(5)}, {stop.longitude.toFixed(5)}</small>
              </button>
            ))}
          </div>
        </aside>

        <div className="route-detail-panel">
          {!stopID ? (
            <div className="card route-empty-state">
              <h3>Select a stop</h3>
              <p>Pick a stop from the list to inspect departures.</p>
            </div>
          ) : null}

          {selectedStopError ? (
            <div className="card">
              <p className="status-error">{selectedStopError}</p>
            </div>
          ) : null}

          {selectedStop ? (
            <>
              <section className="card route-hero">
                <div>
                  <p className="eyebrow">stop</p>
                  <h3>{selectedStop.name}</h3>
                  <p className="route-subtitle">{selectedStop.code || selectedStop.id}</p>
                </div>
                <div className="route-meta-grid">
                  <div>
                    <span>Stop ID</span>
                    <strong>{selectedStop.id}</strong>
                  </div>
                  <div>
                    <span>Zone</span>
                    <strong>{selectedStop.zone_id || "n/a"}</strong>
                  </div>
                  <div>
                    <span>Type</span>
                    <strong>{selectedStop.location_type}</strong>
                  </div>
                  <div>
                    <span>Wheelchair</span>
                    <strong>{selectedStop.wheelchair_boarding}</strong>
                  </div>
                </div>
              </section>

              <section className="card">
                <h3>Departures</h3>
                {departuresError ? <p className="status-error">{departuresError}</p> : null}
                {departuresLoading ? <p className="status-muted">Loading departures…</p> : null}

                <div className="trip-list">
                  {departures.map((departure) => (
                    <div key={`${departure.trip_id}-${departure.departure_time}`} className="trip-row">
                      <strong>{departure.route_short_name || departure.route_long_name || departure.route_id}</strong>
                      <p>
                        {departure.departure_time} to {departure.arrival_time}
                      </p>
                      <small>{departure.headsign || departure.trip_id}</small>
                      <div className="inline-actions">
                        <Link to={`/routes/${encodeURIComponent(departure.route_id)}`}>Open route detail</Link>
                      </div>
                    </div>
                  ))}
                </div>

                {!departuresLoading && departures.length === 0 && !departuresError ? (
                  <p className="status-muted">No departures returned for this stop right now.</p>
                ) : null}
              </section>
            </>
          ) : null}
        </div>
      </div>
    </section>
  );
}

export default StopsPage;
