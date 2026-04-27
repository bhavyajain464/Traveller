import { useEffect, useMemo, useState } from "react";
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
  }, [submittedQuery, stopID]);

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

  const nextDeparture = departures[0] || null;
  const departureSummary = useMemo(() => {
    if (!departures.length) {
      return {
        count: 0,
        primary: "No live departures loaded",
        secondary: "Try another stop or search term."
      };
    }

    return {
      count: departures.length,
      primary: nextDeparture
        ? `${nextDeparture.route_short_name || nextDeparture.route_long_name || nextDeparture.route_id} leaves at ${nextDeparture.departure_time}`
        : "Departures loaded",
      secondary: "Use route detail if you want to inspect the line before you travel."
    };
  }, [departures, nextDeparture]);

  return (
    <section className="route-page">
      <section className="home-hero card">
        <div className="home-hero-copy">
          <p className="eyebrow">Departures</p>
          <h2>Check what leaves from your stop right now.</h2>
          <p className="lead">
            This page is your rider board: search a stop, scan the next departures, and jump into the right route before you commit to the trip.
          </p>

          <div className="hero-actions">
            <Link className="primary-link" to="/plan">Plan a journey</Link>
            <Link className="secondary-link" to="/tickets">Open tickets</Link>
          </div>

          <div className="home-signal-row">
            <div className="home-signal-pill">
              <span>Stop results</span>
              <strong>{stops.length}</strong>
            </div>
            <div className="home-signal-pill">
              <span>Departures loaded</span>
              <strong>{departureSummary.count}</strong>
            </div>
            <div className="home-signal-pill">
              <span>Next departure</span>
              <strong>{nextDeparture ? nextDeparture.departure_time : "Waiting"}</strong>
            </div>
          </div>
        </div>

        <div className="home-hero-panel">
          <div className="section-heading">
            <h3>Current board</h3>
          </div>
          <div className="feature-stack">
            <div>
              <strong>{selectedStop?.name || "Select a stop"}</strong>
              <p>{selectedStop?.code || selectedStop?.id || "Search for a stop to load departures."}</p>
            </div>
            <div>
              <strong>{departureSummary.primary}</strong>
              <p>{departureSummary.secondary}</p>
            </div>
            {nextDeparture ? (
              <div>
                <strong>Best quick action</strong>
                <p>Open the route for {nextDeparture.route_short_name || nextDeparture.route_long_name || nextDeparture.route_id} if you want more context before boarding.</p>
              </div>
            ) : null}
          </div>
        </div>
      </section>

      <div className="route-shell">
        <aside className="route-list-panel card">
          <div className="section-heading">
            <h3>Find a stop</h3>
          </div>

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
          {stopsLoading ? <p className="status-muted">Loading nearby stop boards…</p> : null}

          <div className="route-list compact-list">
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
              <p>Pick a stop from the list to see the departure board.</p>
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
                  <p className="eyebrow">Selected stop</p>
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
                    <span>Wheelchair</span>
                    <strong>{selectedStop.wheelchair_boarding}</strong>
                  </div>
                  <div>
                    <span>Board state</span>
                    <strong>{departures.length ? "Live" : "Quiet"}</strong>
                  </div>
                </div>
              </section>

              <section className="card">
                <div className="section-heading">
                  <h3>Next departures</h3>
                  <span>{departuresLoading ? "Refreshing..." : `${departures.length} listed`}</span>
                </div>

                {departuresError ? <p className="status-error">{departuresError}</p> : null}
                {departuresLoading ? <p className="status-muted">Loading departures…</p> : null}

                <div className="trip-list">
                  {departures.map((departure) => (
                    <div key={`${departure.trip_id}-${departure.departure_time}`} className="trip-row departure-card">
                      <div className="departure-card-head">
                        <div>
                          <p className="eyebrow">Route</p>
                          <strong>{departure.route_short_name || departure.route_long_name || departure.route_id}</strong>
                        </div>
                        <span className="departure-time-pill">{departure.departure_time}</span>
                      </div>
                      <p>
                        Arrives by {departure.arrival_time}
                      </p>
                      <small>{departure.headsign || departure.trip_id}</small>
                      <div className="inline-actions">
                        <Link to={`/routes/${encodeURIComponent(departure.route_id)}`}>Open route detail</Link>
                        <Link to="/plan">Plan a full trip</Link>
                      </div>
                    </div>
                  ))}
                </div>

                {!departuresLoading && departures.length === 0 && !departuresError ? (
                  <div className="empty-bills-state">
                    <strong>No departures returned right now</strong>
                    <p className="lead">Try another nearby stop or search a larger station.</p>
                  </div>
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
