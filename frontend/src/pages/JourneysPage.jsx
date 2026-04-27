import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { apiRequest } from "../lib/api";
import { saveActiveJourney } from "../lib/tickets";

const PLANNER_SIGNALS = [
  { label: "Planner", value: "In-memory snapshot" },
  { label: "Search", value: "Round-based routing" },
  { label: "Interchange", value: "Footpath aware" }
];

function JourneysPage() {
  const [originQuery, setOriginQuery] = useState("Chirag Delhi");
  const [destinationQuery, setDestinationQuery] = useState("Pari Chowk");
  const [originResults, setOriginResults] = useState([]);
  const [destinationResults, setDestinationResults] = useState([]);
  const [originPlace, setOriginPlace] = useState(null);
  const [destinationPlace, setDestinationPlace] = useState(null);
  const [options, setOptions] = useState([]);
  const [selectedOption, setSelectedOption] = useState(null);
  const [plannerMeta, setPlannerMeta] = useState(null);
  const [journeySaved, setJourneySaved] = useState(null);
  const [plannerError, setPlannerError] = useState("");
  const [searchError, setSearchError] = useState("");
  const [isPlanning, setIsPlanning] = useState(false);
  const [isSearchingOrigin, setIsSearchingOrigin] = useState(false);
  const [isSearchingDestination, setIsSearchingDestination] = useState(false);
  const [isSelectingOrigin, setIsSelectingOrigin] = useState(false);
  const [isSelectingDestination, setIsSelectingDestination] = useState(false);

  useEffect(() => {
    const timeoutID = window.setTimeout(async () => {
      if (originQuery.trim().length < 2) {
        setOriginResults([]);
        return;
      }

      setIsSearchingOrigin(true);
      try {
        setSearchError("");
        await loadPlaces(originQuery, setOriginResults);
      } catch (error) {
        setSearchError(error.message);
      } finally {
        setIsSearchingOrigin(false);
      }
    }, 250);

    return () => window.clearTimeout(timeoutID);
  }, [originQuery]);

  useEffect(() => {
    const timeoutID = window.setTimeout(async () => {
      if (destinationQuery.trim().length < 2) {
        setDestinationResults([]);
        return;
      }

      setIsSearchingDestination(true);
      try {
        setSearchError("");
        await loadPlaces(destinationQuery, setDestinationResults);
      } catch (error) {
        setSearchError(error.message);
      } finally {
        setIsSearchingDestination(false);
      }
    }, 250);

    return () => window.clearTimeout(timeoutID);
  }, [destinationQuery]);

  const canPlan = originPlace && destinationPlace;
  const selectedSummary = useMemo(() => summarizeOption(selectedOption), [selectedOption]);
  const plannerHighlights = useMemo(() => buildPlannerHighlights(options, selectedOption), [options, selectedOption]);
  const serviceNotice = useMemo(() => buildServiceNotice(options, plannerMeta), [options, plannerMeta]);

  async function planJourney() {
    if (!originPlace || !destinationPlace) {
      setPlannerError("Choose both origin and destination places first.");
      return;
    }

    setIsPlanning(true);
    setPlannerError("");
    setSelectedOption(null);
    setJourneySaved(null);

    try {
      const query = new URLSearchParams({
        from: String(originPlace.id),
        to: String(destinationPlace.id),
        time: new Date().toISOString(),
        mode: "departure",
        results: "5"
      });

      const payload = await apiRequest(`/v3/journey?${query.toString()}`);
      setPlannerMeta(payload.meta || null);
      const nextOptions = mapV3ConnectionsToJourneyOptions(payload.connections || []);
      setOptions(nextOptions);
      setSelectedOption(nextOptions[0] || null);
      if (!nextOptions.length) {
        setPlannerError("No journey options came back for this stop pair.");
      }
    } catch (error) {
      setOptions([]);
      setPlannerMeta(null);
      setPlannerError(error.message);
    } finally {
      setIsPlanning(false);
    }
  }

  function setCurrentJourney(option) {
    if (!originPlace || !destinationPlace) {
      return;
    }

    const timestamp = new Date();
    const summary = summarizeOption(option);
    const nextJourney = {
      id: `J-${timestamp.getTime()}`,
      fromStopName: originPlace.title,
      toStopName: destinationPlace.title,
      createdAtLabel: timestamp.toLocaleString(),
      fromStopID: originPlace.feature_type === "transit_stop" ? originPlace.id : undefined,
      toStopID: destinationPlace.feature_type === "transit_stop" ? destinationPlace.id : undefined,
      fromLatitude: originPlace.latitude,
      fromLongitude: originPlace.longitude,
      toLatitude: destinationPlace.latitude,
      toLongitude: destinationPlace.longitude,
      routeSummary: summary.routeNames || "Walking-only route",
      duration: option.duration
    };

    saveActiveJourney(nextJourney);
    setSelectedOption(option);
    setJourneySaved(nextJourney);
  }

  return (
    <section className="route-page">
      <section className="planner-hero card">
        <div className="planner-hero-copy">
          <p className="eyebrow">Journey Planner</p>
          <h2>Plan on the same lines as the end-goal routing stack.</h2>
          <p className="lead">
            This screen now reflects the backend shift to an in-memory timetable snapshot, round-based search, and interchange footpaths. You can inspect a connection the way a rider would, while still seeing the planner signals we care about during the transition.
          </p>
          <div className="planner-signal-row">
            {PLANNER_SIGNALS.map((signal) => (
              <div key={signal.label} className="planner-signal-pill">
                <span>{signal.label}</span>
                <strong>{signal.value}</strong>
              </div>
            ))}
          </div>
        </div>

        <div className="planner-status-card">
          <div className="planner-status-header">
            <strong>Routing status</strong>
            <span>{options.length ? `${options.length} options` : "Awaiting query"}</span>
          </div>
          <div className="planner-status-grid">
            {plannerHighlights.map((item) => (
              <div key={item.label}>
                <span>{item.label}</span>
                <strong>{item.value}</strong>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="card planner-form-card">
        <div className="planner-grid">
          <StopPicker
            label="From"
            query={originQuery}
            results={originResults}
            selectedPlace={originPlace}
            onQueryChange={setOriginQuery}
            onSearch={async () => {
              setSearchError("");
              try {
                setIsSearchingOrigin(true);
                await loadPlaces(originQuery, setOriginResults);
              } catch (error) {
                setSearchError(error.message);
              } finally {
                setIsSearchingOrigin(false);
              }
            }}
            onSelect={async (suggestion) => {
              setSearchError("");
              setIsSelectingOrigin(true);
              try {
                const resolved = await resolvePlace(suggestion.id);
                setOriginPlace(resolved);
              } catch (error) {
                setSearchError(error.message);
              } finally {
                setIsSelectingOrigin(false);
              }
            }}
            isSelecting={isSelectingOrigin}
            isSearching={isSearchingOrigin}
          />

          <StopPicker
            label="To"
            query={destinationQuery}
            results={destinationResults}
            selectedPlace={destinationPlace}
            onQueryChange={setDestinationQuery}
            onSearch={async () => {
              setSearchError("");
              try {
                setIsSearchingDestination(true);
                await loadPlaces(destinationQuery, setDestinationResults);
              } catch (error) {
                setSearchError(error.message);
              } finally {
                setIsSearchingDestination(false);
              }
            }}
            onSelect={async (suggestion) => {
              setSearchError("");
              setIsSelectingDestination(true);
              try {
                const resolved = await resolvePlace(suggestion.id);
                setDestinationPlace(resolved);
              } catch (error) {
                setSearchError(error.message);
              } finally {
                setIsSelectingDestination(false);
              }
            }}
            isSelecting={isSelectingDestination}
            isSearching={isSearchingDestination}
          />
        </div>

        <div className="planner-route-ribbon">
          <div>
            <span>Origin</span>
            <strong>{originPlace?.title || "Select a place"}</strong>
          </div>
          <div className="planner-route-arrow">→</div>
          <div>
            <span>Destination</span>
            <strong>{destinationPlace?.title || "Select a place"}</strong>
          </div>
        </div>

        <div className="toolbar">
          <button type="button" className="primary-button" disabled={!canPlan || isPlanning} onClick={planJourney}>
            {isPlanning ? "Searching rounds..." : "Plan Journey"}
          </button>
        </div>

        {searchError ? <p className="status-error">{searchError}</p> : null}
        {plannerError ? <p className="status-error">{plannerError}</p> : null}
        {serviceNotice ? (
          <div className="planner-warning-banner">
            <strong>Timetable fallback in effect</strong>
            <p>{serviceNotice}</p>
          </div>
        ) : null}
      </section>

      {options.length > 0 ? (
        <section className="journey-options">
          {options.map((option, index) => {
            const isSelected = selectedOption === option;
            const summary = summarizeOption(option);

            return (
              <article
                key={`${option.departure_time}-${index}`}
                className={isSelected ? "card journey-card selected-card" : "card journey-card"}
              >
                <div className="journey-card-top">
                  <div className="section-heading">
                    <h3>Connection {index + 1}</h3>
                    <span>{formatDateTime(summary.windowStart)} → {formatDateTime(summary.windowEnd)}</span>
                  </div>
                  <div className="journey-summary-chips">
                    <JourneyChip label="Duration" value={`${option.duration} min`} />
                    <JourneyChip label="Transfers" value={`${option.transfers}`} />
                    <JourneyChip label="Walking" value={`${option.walking_time} min`} />
                    <JourneyChip label="Transit" value={summary.transitLabel} />
                  </div>
                </div>

                <div className="journey-legs timeline-legs">
                  {option.legs.map((leg, legIndex) => {
                    const tone = legTone(leg);
                    return (
                      <div key={`${leg.route_id || leg.mode}-${legIndex}`} className={`trip-row planner-leg planner-leg-${tone}`}>
                        <div className="planner-leg-head">
                          <div>
                            <p className="planner-leg-kicker">{legModeLabel(leg)}</p>
                            <strong>{legTitle(leg)}</strong>
                          </div>
                          <span>{leg.duration} min</span>
                        </div>
                        <p>{leg.from_stop_name} to {leg.to_stop_name}</p>
                        <small>{formatDateTime(leg.departure_time)} to {formatDateTime(leg.arrival_time)}</small>
                        {leg.route_id ? (
                          <div className="inline-actions">
                            <Link to={`/routes/${encodeURIComponent(leg.route_id)}`}>Open route detail</Link>
                          </div>
                        ) : null}
                      </div>
                    );
                  })}
                </div>

                <div className="toolbar">
                  <button type="button" className="primary-button" onClick={() => setCurrentJourney(option)}>
                    Set As Current Journey
                  </button>
                  <button type="button" className="ghost-button" onClick={() => setSelectedOption(option)}>
                    Focus this connection
                  </button>
                </div>
              </article>
            );
          })}
        </section>
      ) : null}

      {journeySaved ? (
        <section className="card ticket-card planner-saved-card">
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

function StopPicker({ label, query, results, selectedPlace, onQueryChange, onSearch, onSelect, isSelecting, isSearching }) {
  return (
    <section className="card planner-picker-card">
      <div className="section-heading">
        <h3>{label}</h3>
        {selectedPlace ? <span>{selectedPlace.title}</span> : null}
      </div>

      <div className="toolbar">
        <input value={query} onChange={(event) => onQueryChange(event.target.value)} placeholder={`Search ${label.toLowerCase()} place`} />
        <button type="button" onClick={onSearch} disabled={isSelecting || isSearching}>
          {isSelecting ? "Picking..." : isSearching ? "Searching..." : "Search"}
        </button>
      </div>

      <div className="route-list compact-list">
        {!results.length && query.trim().length >= 2 && !isSearching ? (
          <p className="status-muted">No places found yet. Try a more specific search.</p>
        ) : null}
        {results.map((suggestion) => (
          <button
            key={`${suggestion.provider}:${suggestion.id}`}
            type="button"
            className={selectedPlace?.id === suggestion.id ? "route-list-item active" : "route-list-item"}
            onClick={() => onSelect(suggestion)}
            disabled={isSelecting}
          >
            <strong>{suggestion.title}</strong>
            <span>{suggestion.feature_type || suggestion.provider}</span>
            <small>{suggestion.subtitle || suggestion.id}</small>
          </button>
        ))}
      </div>
    </section>
  );
}

function JourneyChip({ label, value }) {
  return (
    <div className="journey-chip">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function summarizeOption(option) {
  if (!option) {
    return null;
  }

  const transitLegs = option.legs.filter((leg) => leg.route_id);
  const firstLeg = option.legs[0];
  const lastLeg = option.legs[option.legs.length - 1];
  const firstTransitLeg = transitLegs[0];
  const routeNames = transitLegs.map((leg) => leg.route_name || leg.route_id).join(" · ");

  return {
    primaryRouteID: firstTransitLeg?.route_id || "",
    routeNames,
    transitLabel: transitLegs.length ? `${transitLegs.length} legs` : "Walk only",
    windowStart: firstLeg?.departure_time || option.departure_time,
    windowEnd: lastLeg?.arrival_time || option.arrival_time
  };
}

function buildPlannerHighlights(options, selectedOption) {
  const focus = selectedOption || options[0] || null;
  const transitLegs = focus ? focus.legs.filter((leg) => leg.route_id) : [];
  const footpaths = focus ? focus.legs.filter((leg) => isFootpathLeg(leg)) : [];

  return [
    { label: "Best duration", value: options[0] ? `${options[0].duration} min` : "n/a" },
    { label: "Transit legs", value: focus ? `${transitLegs.length}` : "0" },
    { label: "Interchanges", value: focus ? `${footpaths.length}` : "0" },
    { label: "Current focus", value: focus ? `Option ${Math.max(options.indexOf(focus) + 1, 1)}` : "None" }
  ];
}

function buildServiceNotice(options, plannerMeta) {
  if (plannerMeta?.fallback_service_date && plannerMeta?.service_date) {
    return `The planner is currently routing against timetable service date ${plannerMeta.service_date} because matching service for ${plannerMeta.requested_date || "the requested date"} is not available in the loaded dataset.`;
  }

  const firstOption = options[0];
  if (!firstOption || !firstOption.legs?.length) {
    return "";
  }

  const firstLegDate = new Date(firstOption.legs[0].departure_time);
  if (Number.isNaN(firstLegDate.getTime())) {
    return "";
  }

  const requestedDate = new Date();
  const requestedDay = requestedDate.toISOString().slice(0, 10);
  const serviceDay = firstLegDate.toISOString().slice(0, 10);
  if (requestedDay === serviceDay) {
    return "";
  }

  return `The planner is currently routing against timetable service date ${serviceDay} because matching service for ${requestedDay} is not available in the loaded dataset.`;
}

function legTitle(leg) {
  if (leg.mode === "walking") {
    return isFootpathLeg(leg) ? "Interchange footpath" : "Access walk";
  }
  if (leg.mode === "transfer") {
    return "Transfer buffer";
  }
  return leg.route_name || leg.route_id || "Transit leg";
}

function legModeLabel(leg) {
  if (leg.mode === "walking") {
    return isFootpathLeg(leg) ? "Footpath" : "Walking";
  }
  if (leg.mode === "transfer") {
    return "Transfer";
  }
  return (leg.mode || "transit").toUpperCase();
}

function legTone(leg) {
  if (leg.mode === "transfer") {
    return "transfer";
  }
  if (leg.mode === "walking") {
    return isFootpathLeg(leg) ? "footpath" : "walk";
  }
  return "transit";
}

function isFootpathLeg(leg) {
  return leg.mode === "walking" && leg.from_stop_id && leg.to_stop_id;
}

async function loadPlaces(query, setResults) {
  const payload = await apiRequest(`/v3/locations?query=${encodeURIComponent(query)}&limit=6`);
  setResults((payload.locations || []).map(mapV3LocationToSuggestion));
}

async function resolvePlace(id) {
  const payload = await apiRequest(`/places/resolve?id=${encodeURIComponent(id)}`);
  return payload.place;
}

function mapV3LocationToSuggestion(location) {
  return {
    id: location.id,
    title: location.name,
    subtitle: location.coordinates ? `${location.coordinates.lat.toFixed(5)}, ${location.coordinates.lon.toFixed(5)}` : location.type,
    provider: "v3",
    feature_type: location.type
  };
}

function mapV3ConnectionsToJourneyOptions(connections) {
  return connections.map((connection) => {
    const legs = [];

    for (const section of connection.sections || []) {
      if (section.walk) {
        legs.push({
          mode: "walking",
          route_id: "",
          route_name: "Walk",
          from_stop_id: section.walk.from?.id || "",
          from_stop_name: section.walk.from?.name || "",
          to_stop_id: section.walk.to?.id || "",
          to_stop_name: section.walk.to?.name || "",
          departure_time: section.walk.departure || connection.from?.departure || "",
          arrival_time: section.walk.arrival || connection.to?.arrival || "",
          duration: durationStringToMinutes(section.walk.duration),
          stop_count: 0
        });
        continue;
      }

      if (section.journey) {
        legs.push({
          mode: section.journey.category || "transit",
          route_id: section.journey.id || "",
          route_name: section.journey.name || "",
          from_stop_id: section.journey.from?.id || "",
          from_stop_name: section.journey.from?.name || "",
          to_stop_id: section.journey.to?.id || "",
          to_stop_name: section.journey.to?.name || "",
          departure_time: section.journey.departure || "",
          arrival_time: section.journey.arrival || "",
          duration: durationStringToMinutesFromTimes(section.journey.departure, section.journey.arrival),
          stop_count: 0
        });
      }
    }

    return {
      duration: durationStringToMinutes(connection.duration),
      transfers: connection.transfers,
      walking_time: legs.filter((leg) => leg.mode === "walking").reduce((total, leg) => total + Number(leg.duration || 0), 0),
      departure_time: connection.from?.departure || "",
      arrival_time: connection.to?.arrival || "",
      fare: null,
      legs
    };
  });
}

function durationStringToMinutes(value) {
  if (!value) {
    return 0;
  }

  const parts = value.split(":").map((part) => Number(part));
  if (parts.length !== 3 || parts.some((part) => Number.isNaN(part))) {
    return 0;
  }

  return (parts[0] * 60) + parts[1] + Math.round(parts[2] / 60);
}

function durationStringToMinutesFromTimes(departure, arrival) {
  if (!departure || !arrival) {
    return 0;
  }

  const departureTime = new Date(departure);
  const arrivalTime = new Date(arrival);
  if (Number.isNaN(departureTime.getTime()) || Number.isNaN(arrivalTime.getTime())) {
    return 0;
  }

  return Math.max(0, Math.round((arrivalTime - departureTime) / 60000));
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
