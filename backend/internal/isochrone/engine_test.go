package isochrone

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/SrivathsanRam/IsoMap/internal/isochrone/graph"
	gtfsdata "github.com/SrivathsanRam/IsoMap/internal/isochrone/gtfs"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/routing"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/spatial"
)

func TestEngineGeneratesWalkingAwareMultiPolygon(t *testing.T) {
	artifact := engineTestArtifact(t)
	config := DefaultConfig()
	config.Walking.SpeedMetersPerSecond = 1
	config.Walking.NetworkFactor = 1
	config.Walking.CellSizeMeters = 50
	config.Walking.MaximumGridCells = 10000
	config.Contour.MinimumRingAreaSquareMeters = 2500
	engine, err := New(artifact, config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	location, err := time.LoadLocation("Asia/Singapore")
	if err != nil {
		t.Fatal(err)
	}
	result, err := engine.Generate(context.Background(), Query{
		Origin:           spatial.Point{Lat: artifact.Graph.Stops[0].Lat, Lon: artifact.Graph.Stops[0].Lon},
		Departure:        time.Date(2026, time.January, 5, 8, 1, 0, 0, location),
		MaxTravelSeconds: 30 * 60,
		OriginStops:      []routing.StopSeed{{Stop: 0}},
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.Geometry.Type != "MultiPolygon" || len(result.Geometry.Coordinates) == 0 {
		t.Fatalf("Generate() geometry = %+v, want non-empty MultiPolygon", result.Geometry)
	}
	if result.ReachedStopCount != 3 {
		t.Fatalf("ReachedStopCount = %d, want 3", result.ReachedStopCount)
	}
	if result.GridColumns < 2 || result.GridRows < 2 {
		t.Fatalf("invalid grid dimensions: %dx%d", result.GridColumns, result.GridRows)
	}
}

func TestSingaporeEngineSmoke(t *testing.T) {
	artifactPath := os.Getenv("ISOMAP_GTFS_GRAPH")
	if artifactPath == "" {
		t.Skip("set ISOMAP_GTFS_GRAPH to run the full-feed smoke test")
	}
	artifact, err := graph.LoadArtifact(artifactPath)
	if err != nil {
		t.Fatalf("graph.LoadArtifact() error = %v", err)
	}
	stop := graph.NoIndex
	for index, candidate := range artifact.Graph.Stops {
		if candidate.ID == "NS15" {
			stop = uint32(index)
			break
		}
	}
	if stop == graph.NoIndex {
		t.Fatal("Singapore artifact does not contain stop NS15")
	}
	engine, err := New(artifact, DefaultConfig())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	location, err := time.LoadLocation("Asia/Singapore")
	if err != nil {
		t.Fatal(err)
	}
	result, err := engine.Generate(context.Background(), Query{
		Origin: spatial.Point{
			Lat: artifact.Graph.Stops[stop].Lat,
			Lon: artifact.Graph.Stops[stop].Lon,
		},
		Departure:        time.Date(2026, time.August, 13, 8, 0, 0, 0, location),
		MaxTravelSeconds: 90 * 60,
		OriginStops:      []routing.StopSeed{{Stop: stop}},
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.ReachedStopCount < 100 || len(result.Geometry.Coordinates) == 0 {
		t.Fatalf("unexpected smoke result: reached=%d polygons=%d", result.ReachedStopCount, len(result.Geometry.Coordinates))
	}
	t.Logf(
		"reached %d stops, settled %d nodes, rasterized %dx%d vertices, generated %d polygons",
		result.ReachedStopCount,
		result.SettledNodeCount,
		result.GridColumns,
		result.GridRows,
		len(result.Geometry.Coordinates),
	)
}

func engineTestArtifact(t *testing.T) *graph.Artifact {
	t.Helper()
	feed := &gtfsdata.Feed{
		Info: gtfsdata.FeedInfo{Version: "engine-test"},
		Agencies: []gtfsdata.Agency{
			{ID: "A", Name: "Test Transit", URL: "https://example.com", Timezone: "Asia/Singapore", SourceLine: 2},
		},
		Stops: []gtfsdata.Stop{
			{ID: "S1", Name: "First", Lat: 1.35, Lon: 103.82, SourceLine: 2},
			{ID: "S2", Name: "Second", Lat: 1.355, Lon: 103.825, SourceLine: 3},
			{ID: "S3", Name: "Third", Lat: 1.36, Lon: 103.83, SourceLine: 4},
		},
		Routes: []gtfsdata.Route{
			{ID: "R1", AgencyID: "A", ShortName: "R1", LongName: "Test Route", Type: 3, SourceLine: 2},
		},
		Services: []gtfsdata.Service{
			{ID: "DAILY", Days: [7]bool{true, true, true, true, true, true, true}, StartDate: 20260101, EndDate: 20301231, SourceLine: 2},
		},
		Trips: []gtfsdata.Trip{
			{ID: "T1", RouteID: "R1", ServiceID: "DAILY", DirectionID: 0, SourceLine: 2},
		},
		StopTimes: []gtfsdata.StopTime{
			{TripID: "T1", StopID: "S1", ArrivalSeconds: 8 * 3600, DepartureSeconds: 8 * 3600, StopSequence: 1, SourceLine: 2},
			{TripID: "T1", StopID: "S2", ArrivalSeconds: 8*3600 + 600, DepartureSeconds: 8*3600 + 600, StopSequence: 2, SourceLine: 3},
			{TripID: "T1", StopID: "S3", ArrivalSeconds: 8*3600 + 1200, DepartureSeconds: 8*3600 + 1200, StopSequence: 3, SourceLine: 4},
		},
		Frequencies: []gtfsdata.Frequency{
			{TripID: "T1", StartSeconds: 8 * 3600, EndSeconds: 10 * 3600, HeadwaySeconds: 600, SourceLine: 2},
		},
	}
	artifact, err := graph.Build(feed, graph.Source{Filename: "test.zip", SHA256: "test"})
	if err != nil {
		t.Fatalf("graph.Build() error = %v", err)
	}
	return artifact
}
