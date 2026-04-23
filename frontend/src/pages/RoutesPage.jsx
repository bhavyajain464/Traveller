import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { apiRequest } from "../lib/api";

function RoutesPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [query, setQuery] = useState("");
  const [submittedQuery, setSubmittedQuery] = useState("");
  const [routes, setRoutes] = useState([]);
  const [routesLoading, setRoutesLoading] = useState(false);
  const [routesError, setRoutesError] = useState("");
  const [detail, setDetail] = useState(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState("");

  useEffect(() => {
    let ignore = false;

    async function loadRoutes() {
      setRoutesLoading(true);
      setRoutesError("");

      try {
        const path = submittedQuery.trim()
          ? `/routes/search?q=${encodeURIComponent(submittedQuery.trim())}&limit=24`
          : "/routes?limit=24";
        const payload = await apiRequest(path);
        if (!ignore) {
          setRoutes(payload.routes || []);
        }
      } catch (err) {
        if (!ignore) {
          setRoutes([]);
          setRoutesError(err.message);
        }
      } finally {
        if (!ignore) {
          setRoutesLoading(false);
        }
      }
    }

    loadRoutes();

    return () => {
      ignore = true;
    };
  }, [submittedQuery]);

  useEffect(() => {
    let ignore = false;

    async function loadDetail() {
      if (!id) {
        setDetail(null);
        setDetailError("");
        setDetailLoading(false);
        return;
      }

      setDetailLoading(true);
      setDetailError("");

      try {
        const payload = await apiRequest(`/routes/${encodeURIComponent(id)}/detail?trip_limit=10`);
        if (!ignore) {
          setDetail(payload);
        }
      } catch (err) {
        if (!ignore) {
          setDetail(null);
          setDetailError(err.message);
        }
      } finally {
        if (!ignore) {
          setDetailLoading(false);
        }
      }
    }

    loadDetail();

    return () => {
      ignore = true;
    };
  }, [id]);

  const onSearch = (event) => {
    event.preventDefault();
    setSubmittedQuery(query);
  };

  const selectedRouteID = detail?.route?.id || id;

  return (
    <section className="route-page">
      <div className="page-header">
        <div>
          <h2>Routes</h2>
          <p className="lead">
            Search routes and inspect stops, trip samples, and GTFS metadata in one place.
          </p>
        </div>
      </div>

      <div className="route-shell">
        <aside className="route-list-panel card">
          <form className="toolbar" onSubmit={onSearch}>
            <input
              placeholder="Search by route id or name"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
            <button type="submit">Search</button>
            <button
              type="button"
              onClick={() => {
                setQuery("");
                setSubmittedQuery("");
              }}
            >
              Reset
            </button>
          </form>

          {routesError ? <p className="status-error">{routesError}</p> : null}
          {routesLoading ? <p className="status-muted">Loading routes…</p> : null}

          <div className="route-list">
            {routes.map((route) => (
              <button
                key={route.id}
                type="button"
                className={route.id === selectedRouteID ? "route-list-item active" : "route-list-item"}
                onClick={() => navigate(`/routes/${encodeURIComponent(route.id)}`)}
              >
                <strong>{route.short_name || route.long_name || route.id}</strong>
                <span>{route.long_name || "No long name available"}</span>
                <small>{route.id}</small>
              </button>
            ))}
          </div>

          {!routesLoading && routes.length === 0 && !routesError ? (
            <p className="status-muted">No routes matched this search.</p>
          ) : null}
        </aside>

        <div className="route-detail-panel">
          {!id ? (
            <div className="card route-empty-state">
              <h3>Select a route</h3>
              <p>Pick a route from the list to open its detail view.</p>
            </div>
          ) : null}

          {detailError ? (
            <div className="card">
              <p className="status-error">{detailError}</p>
            </div>
          ) : null}

          {detailLoading ? (
            <div className="card">
              <p className="status-muted">Loading route detail…</p>
            </div>
          ) : null}

          {detail ? (
            <>
              <section className="card route-hero">
                <div>
                  <p className="eyebrow">{detail.mode}</p>
                  <h3>{detail.route.short_name || detail.route.long_name || detail.route.id}</h3>
                  <p className="route-subtitle">
                    {detail.route.long_name || detail.route.description || "No description available"}
                  </p>
                </div>

                <div className="route-meta-grid">
                  <div>
                    <span>Route ID</span>
                    <strong>{detail.route.id}</strong>
                  </div>
                  <div>
                    <span>Agency</span>
                    <strong>{detail.route.agency_id}</strong>
                  </div>
                  <div>
                    <span>Stops</span>
                    <strong>{detail.stop_count}</strong>
                  </div>
                  <div>
                    <span>Trips</span>
                    <strong>{detail.trip_count}</strong>
                  </div>
                </div>
              </section>

              <section className="card">
                <h3>Route span</h3>
                <p className="lead">
                  {detail.first_stop_name && detail.last_stop_name
                    ? `${detail.first_stop_name} to ${detail.last_stop_name}`
                    : "Stops are available below."}
                </p>
                {detail.route.description ? <p>{detail.route.description}</p> : null}
                {detail.route.url ? (
                  <p>
                    <a href={detail.route.url} target="_blank" rel="noreferrer">
                      Open agency route page
                    </a>
                  </p>
                ) : null}
              </section>

              <div className="route-detail-grid">
                <section className="card">
                  <div className="section-heading">
                    <h3>Stops</h3>
                    <span>{detail.stops.length} shown</span>
                  </div>
                  <div className="stop-sequence">
                    {detail.stops.map((stop, index) => (
                      <div key={stop.id} className="stop-row">
                        <div className="stop-index">{index + 1}</div>
                        <div>
                          <strong>{stop.name}</strong>
                          <p>{stop.code || stop.id}</p>
                        </div>
                        <small>
                          {stop.latitude.toFixed(5)}, {stop.longitude.toFixed(5)}
                        </small>
                      </div>
                    ))}
                  </div>
                </section>

                <section className="card">
                  <div className="section-heading">
                    <h3>Trip samples</h3>
                    <span>{detail.trips.length} shown</span>
                  </div>
                  <div className="trip-list">
                    {detail.trips.map((trip) => (
                      <div key={trip.id} className="trip-row">
                        <strong>{trip.headsign || trip.short_name || trip.id}</strong>
                        <p>{trip.short_name || trip.service_id}</p>
                        <small>
                          {trip.direction_id === null || trip.direction_id === undefined
                            ? "Direction n/a"
                            : `Direction ${trip.direction_id}`}
                        </small>
                      </div>
                    ))}
                  </div>
                  {detail.trip_count > detail.trips.length ? (
                    <p className="status-muted">
                      Showing the first {detail.trips.length} trips out of {detail.trip_count}.
                    </p>
                  ) : null}
                </section>
              </div>
            </>
          ) : null}
        </div>
      </div>

      {detail ? (
        <p className="status-muted">
          API: <code>/routes/{detail.route.id}/detail</code>. Raw stops and trips endpoints still remain available.
        </p>
      ) : (
        <p className="status-muted">
          This screen now uses the consolidated route detail API instead of separate manual calls.
        </p>
      )}

      {id ? (
        <p className="status-muted">
          Shareable URL: <Link to={`/routes/${encodeURIComponent(id)}`}>/routes/{id}</Link>
        </p>
      ) : null}
    </section>
  );
}

export default RoutesPage;
