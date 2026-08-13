package routing

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBuildRouteQueryUsesSingaporeTimeForPublicTransport(t *testing.T) {
	query := buildRouteQuery(
		point{Lat: 1.300001, Lon: 103.800001},
		point{Lat: 1.350001, Lon: 103.850001},
		"transit",
		time.Date(2026, time.August, 13, 0, 15, 30, 0, time.UTC),
	)

	if got := query.Get("routeType"); got != "pt" {
		t.Fatalf("routeType = %q, want pt", got)
	}
	if got := query.Get("date"); got != "08-13-2026" {
		t.Fatalf("date = %q, want 08-13-2026", got)
	}
	if got := query.Get("time"); got != "08:15:30" {
		t.Fatalf("time = %q, want 08:15:30", got)
	}
}

func TestResolveDepartureTimeUsesSelectedSingaporeClockTime(t *testing.T) {
	departure, err := resolveDepartureTime(
		"18:35",
		time.Date(2026, time.August, 12, 20, 15, 30, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := departure.Format(time.RFC3339); got != "2026-08-13T18:35:00+08:00" {
		t.Fatalf("departure = %q, want Singapore-local date and selected time", got)
	}
}

func TestBuildRoutesDropsWalkOnlyPublicTransportAlternative(t *testing.T) {
	var response oneMapRouteResponse
	err := json.Unmarshal([]byte(`{
		"plan":{"itineraries":[
			{"duration":600,"walkDistance":700,"legs":[
				{"mode":"WALK","duration":600,"distance":700,"from":{"name":"A","lat":1.3,"lon":103.8},"to":{"name":"B","lat":1.31,"lon":103.81}}
			]},
			{"duration":900,"walkDistance":180,"legs":[
				{"mode":"WALK","duration":120,"distance":100,"from":{"name":"A","lat":1.3,"lon":103.8},"to":{"name":"Stop 1","lat":1.301,"lon":103.801}},
				{"mode":"BUS","duration":700,"distance":5000,"routeShortName":"14","from":{"name":"Stop 1","lat":1.301,"lon":103.801},"to":{"name":"Stop 2","lat":1.34,"lon":103.84}},
				{"mode":"WALK","duration":80,"distance":80,"from":{"name":"Stop 2","lat":1.34,"lon":103.84},"to":{"name":"B","lat":1.341,"lon":103.841}}
			]}
		]}}`), &response)
	if err != nil {
		t.Fatal(err)
	}

	routes, err := buildRoutes(response, point{Lat: 1.3, Lon: 103.8}, point{Lat: 1.341, Lon: 103.841}, "transit")
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 1 {
		t.Fatalf("got %d routes, want only the transit itinerary", len(routes))
	}
	if got := routes[0].DistanceMeters; got != 5180 {
		t.Fatalf("distance = %.0f, want 5180 without double-counting walk distance", got)
	}
	if len(routes[0].Segments) != 3 || routes[0].Segments[1].Mode != "BUS" {
		t.Fatalf("segments = %#v, want WALK/BUS/WALK", routes[0].Segments)
	}
}

func TestBuildRoutesRejectsWalkOnlyPublicTransportResponse(t *testing.T) {
	var response oneMapRouteResponse
	if err := json.Unmarshal([]byte(`{"plan":{"itineraries":[{"duration":60,"legs":[{"mode":"WALK","duration":60,"distance":50}]}]}}`), &response); err != nil {
		t.Fatal(err)
	}

	if _, err := buildRoutes(response, point{Lat: 1.3, Lon: 103.8}, point{Lat: 1.31, Lon: 103.81}, "transit"); err == nil {
		t.Fatal("expected a walk-only public transport response to be rejected")
	}
}
