package isochrone

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	engine "github.com/SrivathsanRam/IsoMap/internal/isochrone"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/contour"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/graph"
)

type fakeGenerator struct {
	query engine.Query
	calls int
}

func (g *fakeGenerator) Generate(_ context.Context, query engine.Query) (*engine.Result, error) {
	g.query = query
	g.calls++
	return &engine.Result{
		Geometry: contour.MultiPolygon{
			Type: "MultiPolygon",
			Coordinates: [][][][]float64{
				{{{103.8, 1.3}, {103.81, 1.3}, {103.81, 1.31}, {103.8, 1.3}}},
			},
		},
		ReachedStopCount: 12,
		SettledNodeCount: 34,
		GridColumns:      20,
		GridRows:         10,
	}, nil
}

func TestHandlerReturnsGeoJSONFeature(t *testing.T) {
	generator := &fakeGenerator{}
	handler, err := New(generator, graph.Manifest{Timezone: "Asia/Singapore", FeedVersion: "test-v1"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	handler.clock = func() time.Time {
		return time.Date(2026, time.August, 13, 9, 30, 0, 0, time.UTC)
	}
	request := httptest.NewRequest(http.MethodPost, "/isochrone", strings.NewReader(`{"lat":1.35,"lon":103.82,"minutes":30}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if got, want := response.Header().Get("Content-Type"), "application/geo+json"; got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}
	if generator.calls != 1 || generator.query.MaxTravelSeconds != 30*60 {
		t.Fatalf("generator call = %d, query = %+v", generator.calls, generator.query)
	}

	var body feature
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Type != "Feature" || body.Geometry.Type != "MultiPolygon" || body.Properties.FeedVersion != "test-v1" {
		t.Fatalf("unexpected response: %+v", body)
	}
	if body.Properties.ReachedStopCount != 12 || body.Properties.Minutes != 30 {
		t.Fatalf("unexpected properties: %+v", body.Properties)
	}
}

func TestHandlerRejectsInvalidTravelTime(t *testing.T) {
	generator := &fakeGenerator{}
	handler, err := New(generator, graph.Manifest{Timezone: "Asia/Singapore"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/isochrone", strings.NewReader(`{"lat":1.35,"lon":103.82,"minutes":91}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
	if generator.calls != 0 {
		t.Fatalf("generator called %d times for invalid input", generator.calls)
	}
}

func TestHandlerUsesExplicitDepartureTime(t *testing.T) {
	generator := &fakeGenerator{}
	handler, err := New(generator, graph.Manifest{Timezone: "Asia/Singapore"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/isochrone",
		strings.NewReader(`{"lat":1.35,"lon":103.82,"minutes":30,"departure_time":"2026-08-13T08:00:00+08:00"}`),
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if got := generator.query.Departure.Format(time.RFC3339); got != "2026-08-13T08:00:00+08:00" {
		t.Fatalf("departure = %s", got)
	}
}

func TestSingaporeEndpointSmoke(t *testing.T) {
	artifactPath := os.Getenv("ISOMAP_GTFS_GRAPH")
	if artifactPath == "" {
		t.Skip("set ISOMAP_GTFS_GRAPH to run the full endpoint smoke test")
	}
	artifact, err := graph.LoadArtifact(artifactPath)
	if err != nil {
		t.Fatalf("graph.LoadArtifact() error = %v", err)
	}
	isochroneEngine, err := engine.New(artifact, engine.DefaultConfig())
	if err != nil {
		t.Fatalf("engine.New() error = %v", err)
	}
	handler, err := New(isochroneEngine, artifact.Manifest)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/isochrone",
		strings.NewReader(`{"lat":1.381758,"lon":103.844801,"minutes":45,"departure_time":"2026-08-13T08:00:00+08:00"}`),
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body feature
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Geometry.Type != "MultiPolygon" || len(body.Geometry.Coordinates) == 0 {
		t.Fatalf("endpoint returned empty geometry")
	}
	if body.Properties.ReachedStopCount < 100 {
		t.Fatalf("endpoint reached only %d stops", body.Properties.ReachedStopCount)
	}
	t.Logf(
		"endpoint reached %d stops and returned %d polygon(s)",
		body.Properties.ReachedStopCount,
		len(body.Geometry.Coordinates),
	)
}
