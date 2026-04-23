import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { apiRequest } from "../lib/api";
import { saveActiveJourney } from "../lib/tickets";

function JourneysPage() {
  const [originQuery, setOriginQuery] = useState("Chirag Delhi");
  const [destinationQuery, setDestinationQuery] = useState("Pari Chowk");
  const [originResults, setOriginResults] = useState([]);
  const [destinationResults, setDestinationResults] = useState([]);
  const [originStop, setOriginStop] = useState(null);
  const [destinationStop, setDestinationStop] = useState(null);
  const [options, setOptions] = useState([]);
  const [selectedOption, setSelectedOption] = useState(null);
  const [journeySaved, setJourneySaved] = useState(null);
  const [plannerError, setPlannerError] = useState("");
  const [searchError, setSearchError] = useState("");
  const [isPlanning, setIsPlanning] = useState(false);

  useEffect(() => {
    loadStops(originQuery, setOriginResults, setOriginStop);
    loadStops(destinationQuery, setDestinationResults, setDestinationStop);
  }, []);

  const canPlan = originStop && destinationStop;
  const selectedSummary = useMemo(() => {
    if (!selectedOption) {
      return null;
    }

    const firstTransitLeg = selectedOption.legs.find((leg) => leg.route_id);
    const routeNames = selectedOption.legs
      .filter((leg) => leg.route_id)
      .map((leg) => leg.route_name || leg.route_id);

    return {
      primaryRouteID: firstTransitLeg?.route_id || "",
      routeNames: routeNames.join(" · ")
    };
  }, [selectedOption]);

  async function planJourney() {
    if (!originStop || !destinationStop) {
      setPlannerError("Choose both origin and destination stops first.");
      return;
    }

    setIsPlanning(true);
    setPlannerError("");
    setSelectedOption(null);
    setJourneySaved(null);

    try {
      const query = new URLSearchParams({
        from_lat: String(originStop.latitude),
        from_lon: String(originStop.longitude),
        to_lat: String(destinationStop.latitude),
        to_lon: String(destinationStop.longitude),
        date: new Date().toISOString().slice(0, 10)
      });

      const payload = await apiRequest(`/journeys/plan?${query.toString()}`, { method: "POST" });
      setOptions(payload.options || []);
    } catch (error) {
      setOptions([]);
      setPlannerError(error.message);
    } finally {
      setIsPlanning(false);
    }
  }

  function setCurrentJourney(option) {
    if (!originStop || !destinationStop) {
      return;
    }

    const timestamp = new Date();
    const nextJourney = {
      id: `J-${timestamp.getTime()}`,
      fromStopName: originStop.name,
      toStopName: destinationStop.name,
      createdAtLabel: timestamp.toLocaleString(),
      fromStopID: originStop.id,
      toStopID: destinationStop.id,
      routeSummary: option.legs.filter((leg) => leg.route_id).map((leg) => leg.route_name || leg.route_id).join(" · "),
      duration: option.duration
    };

    saveActiveJourney(nextJourney);
    setSelectedOption(option);
    setJourneySaved(nextJourney);
  }

  return (
    <section className="route-page">
      <div className="page-header">
        <div>
          <h2>Plan Journey</h2>
          <p className="lead">
            Search stops, plan a trip, review route legs, and set your current journey before you check in and mint a live QR.
          </p>
        </div>
      </div>

      <section className="card">
        <div className="planner-grid">
          <StopPicker
            label="From"
            query={originQuery}
            results={originResults}
            selectedStop={originStop}
            onQueryChange={setOriginQuery}
            onSearch={async () => {
              setSearchError("");
              try {
                await loadStops(originQuery, setOriginResults, setOriginStop);
              } catch (error) {
                setSearchError(error.message);
              }
            }}
            onSelect={setOriginStop}
          />

          <StopPicker
            label="To"
            query={destinationQuery}
            results={destinationResults}
            selectedStop={destinationStop}
            onQueryChange={setDestinationQuery}
            onSearch={async () => {
              setSearchError("");
              try {
                await loadStops(destinationQuery, setDestinationResults, setDestinationStop);
              } catch (error) {
                setSearchError(error.message);
              }
            }}
            onSelect={setDestinationStop}
          />
        </div>

        <div className="toolbar">
          <button type="button" className="primary-button" disabled={!canPlan || isPlanning} onClick={planJourney}>
            {isPlanning ? "Planning..." : "Plan Journey"}
          </button>
        </div>

        {searchError ? <p className="status-error">{searchError}</p> : null}
        {plannerError ? <p className="status-error">{plannerError}</p> : null}
      </section>

      {options.length > 0 ? (
        <section className="journey-options">
          {options.map((option, index) => {
            const isSelected = selectedOption === option;

            return (
              <article key={`${option.departure_time}-${index}`} className={isSelected ? "card journey-card selected-card" : "card journey-card"}>
                <div className="section-heading">
                  <h3>Option {index + 1}</h3>
                  <span>{option.duration} min · {option.transfers} transfers</span>
                </div>
                <p className="lead">
                  Departure {formatDateTime(option.departure_time)} and arrival {formatDateTime(option.arrival_time)}
                </p>

                <div className="journey-legs">
                  {option.legs.map((leg, legIndex) => (
                    <div key={`${leg.route_id || leg.mode}-${legIndex}`} className="trip-row">
                      <strong>{leg.mode === "walking" ? "Walking" : leg.route_name || leg.route_id}</strong>
                      <p>{leg.from_stop_name} to {leg.to_stop_name}</p>
                      <small>{formatDateTime(leg.departure_time)} to {formatDateTime(leg.arrival_time)} · {leg.duration} min</small>
                      {leg.route_id ? (
                        <div className="inline-actions">
                          <Link to={`/routes/${encodeURIComponent(leg.route_id)}`}>Open route detail</Link>
                        </div>
                      ) : null}
                    </div>
                  ))}
                </div>

                <div className="toolbar">
                  <button type="button" className="primary-button" onClick={() => setCurrentJourney(option)}>
                    Set As Current Journey
                  </button>
                </div>
              </article>
            );
          })}
        </section>
      ) : null}

      {journeySaved ? (
        <section className="card ticket-card">
          <div className="section-heading">
            <h3>Current journey saved</h3>
            {selectedSummary?.primaryRouteID ? (
              <Link to={`/routes/${encodeURIComponent(selectedSummary.primaryRouteID)}`}>Open main route</Link>
            ) : null}
          </div>

          <div className="ticket-copy">
            <p className="eyebrow">Ready for travel</p>
            <strong>{journeySaved.fromStopName} to {journeySaved.toStopName}</strong>
            <p>{journeySaved.createdAtLabel}</p>
            <p>{journeySaved.routeSummary || selectedSummary?.routeNames || "Walking-only route"}</p>
            <p className="status-muted">
              Check in from Tickets to generate a live QR with the actual time and location.
            </p>
          </div>
        </section>
      ) : null}
    </section>
  );
}

function StopPicker({ label, query, results, selectedStop, onQueryChange, onSearch, onSelect }) {
  return (
    <section className="card">
      <div className="section-heading">
        <h3>{label}</h3>
        {selectedStop ? <span>{selectedStop.name}</span> : null}
      </div>

      <div className="toolbar">
        <input value={query} onChange={(event) => onQueryChange(event.target.value)} placeholder={`Search ${label.toLowerCase()} stop`} />
        <button type="button" onClick={onSearch}>Search</button>
      </div>

      <div className="route-list compact-list">
        {results.map((stop) => (
          <button
            key={stop.id}
            type="button"
            className={selectedStop?.id === stop.id ? "route-list-item active" : "route-list-item"}
            onClick={() => onSelect(stop)}
          >
            <strong>{stop.name}</strong>
            <span>{stop.code || stop.id}</span>
            <small>{stop.latitude.toFixed(5)}, {stop.longitude.toFixed(5)}</small>
          </button>
        ))}
      </div>
    </section>
  );
}

async function loadStops(query, setResults, setSelectedStop) {
  const payload = await apiRequest(`/stops/search?q=${encodeURIComponent(query)}&limit=6`);
  const stops = payload.stops || [];
  setResults(stops);
  setSelectedStop(stops[0] || null);
}

function formatDateTime(value) {
  if (!value) {
    return "n/a";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString();
}

export default JourneysPage;
