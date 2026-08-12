package routing

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/SrivathsanRam/IsoMap/internal/isochrone/graph"
	gtfsdata "github.com/SrivathsanRam/IsoMap/internal/isochrone/gtfs"
)

func TestSingaporeArtifactSmoke(t *testing.T) {
	artifactPath := os.Getenv("ISOMAP_GTFS_GRAPH")
	if artifactPath == "" {
		t.Skip("set ISOMAP_GTFS_GRAPH to run the full-feed smoke test")
	}
	artifact, err := graph.LoadArtifact(artifactPath)
	if err != nil {
		t.Fatalf("graph.LoadArtifact() error = %v", err)
	}
	router, err := New(artifact)
	if err != nil {
		t.Fatalf("New() error = %v", err)
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

	result, err := router.Search(context.Background(), Query{
		Departure:        singaporeTime(t, 2026, time.August, 13, 8, 0),
		MaxTravelSeconds: 90 * 60,
		Origins:          []StopSeed{{Stop: stop}},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if result.ReachedStopCount < 100 {
		t.Fatalf("ReachedStopCount = %d, want at least 100", result.ReachedStopCount)
	}
	t.Logf("reached %d stops after settling %d nodes", result.ReachedStopCount, result.SettledNodeCount)
}

func TestSearchUsesExpectedHalfHeadwayWait(t *testing.T) {
	router := testRouter(t, dailyService(), []int{8 * 3600, 8*3600 + 600, 8*3600 + 1200}, []gtfsdata.Frequency{
		{TripID: "T1", StartSeconds: 8 * 3600, EndSeconds: 10 * 3600, HeadwaySeconds: 600, SourceLine: 2},
	})

	result, err := router.Search(context.Background(), Query{
		Departure:        singaporeTime(t, 2026, time.January, 5, 8, 1),
		MaxTravelSeconds: 30 * 60,
		Origins:          []StopSeed{{Stop: 0}},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if got, want := result.StopTravelSeconds[0], int32(0); got != want {
		t.Fatalf("origin travel time = %d, want %d", got, want)
	}
	if got, want := result.StopTravelSeconds[1], int32(15*60); got != want {
		t.Fatalf("second stop travel time = %d, want %d", got, want)
	}
	if got, want := result.StopTravelSeconds[2], int32(25*60); got != want {
		t.Fatalf("third stop travel time = %d, want %d", got, want)
	}
}

func TestSearchShiftsFrequencyWindowAtDownstreamStop(t *testing.T) {
	router := testRouter(t, dailyService(), []int{8 * 3600, 8*3600 + 600, 8*3600 + 1200}, []gtfsdata.Frequency{
		{TripID: "T1", StartSeconds: 8 * 3600, EndSeconds: 10 * 3600, HeadwaySeconds: 600, SourceLine: 2},
	})

	result, err := router.Search(context.Background(), Query{
		Departure:        singaporeTime(t, 2026, time.January, 5, 8, 11),
		MaxTravelSeconds: 20 * 60,
		Origins:          []StopSeed{{Stop: 1}},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if got, want := result.StopTravelSeconds[2], int32(15*60); got != want {
		t.Fatalf("downstream travel time = %d, want %d", got, want)
	}
}

func TestSearchHonorsTravelTimeBound(t *testing.T) {
	router := testRouter(t, dailyService(), []int{8 * 3600, 8*3600 + 600, 8*3600 + 1200}, []gtfsdata.Frequency{
		{TripID: "T1", StartSeconds: 8 * 3600, EndSeconds: 10 * 3600, HeadwaySeconds: 600, SourceLine: 2},
	})

	result, err := router.Search(context.Background(), Query{
		Departure:        singaporeTime(t, 2026, time.January, 5, 8, 11),
		MaxTravelSeconds: 14*60 + 59,
		Origins:          []StopSeed{{Stop: 1}},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if got := result.StopTravelSeconds[2]; got != UnreachableSeconds {
		t.Fatalf("bounded destination travel time = %d, want unreachable", got)
	}
}

func TestSearchHonorsServiceCalendar(t *testing.T) {
	mondayOnly := [7]bool{true, false, false, false, false, false, false}
	router := testRouter(t, mondayOnly, []int{8 * 3600, 8*3600 + 600, 8*3600 + 1200}, []gtfsdata.Frequency{
		{TripID: "T1", StartSeconds: 8 * 3600, EndSeconds: 10 * 3600, HeadwaySeconds: 600, SourceLine: 2},
	})

	result, err := router.Search(context.Background(), Query{
		Departure:        singaporeTime(t, 2026, time.January, 6, 8, 1), // Tuesday
		MaxTravelSeconds: 30 * 60,
		Origins:          []StopSeed{{Stop: 0}},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if got := result.StopTravelSeconds[1]; got != UnreachableSeconds {
		t.Fatalf("inactive-service destination = %d, want unreachable", got)
	}
}

func TestSearchFindsPreviousServiceDayAfterMidnight(t *testing.T) {
	mondayOnly := [7]bool{true, false, false, false, false, false, false}
	router := testRouter(t, mondayOnly, []int{24 * 3600, 24*3600 + 600, 24*3600 + 1200}, []gtfsdata.Frequency{
		{TripID: "T1", StartSeconds: 24 * 3600, EndSeconds: 26 * 3600, HeadwaySeconds: 600, SourceLine: 2},
	})

	result, err := router.Search(context.Background(), Query{
		Departure:        singaporeTime(t, 2026, time.January, 6, 0, 15), // Tuesday, Monday service day
		MaxTravelSeconds: 30 * 60,
		Origins:          []StopSeed{{Stop: 0}},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if got, want := result.StopTravelSeconds[1], int32(15*60); got != want {
		t.Fatalf("after-midnight travel time = %d, want %d", got, want)
	}
}

func TestSearchSupportsScheduledTrips(t *testing.T) {
	router := testRouter(t, dailyService(), []int{8 * 3600, 8*3600 + 600, 8*3600 + 1200}, nil)
	result, err := router.Search(context.Background(), Query{
		Departure:        singaporeTime(t, 2026, time.January, 5, 7, 55),
		MaxTravelSeconds: 20 * 60,
		Origins:          []StopSeed{{Stop: 0}},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if got, want := result.StopTravelSeconds[1], int32(15*60); got != want {
		t.Fatalf("scheduled travel time = %d, want %d", got, want)
	}
}

func TestSearchReturnsCancelledContext(t *testing.T) {
	router := testRouter(t, dailyService(), []int{8 * 3600, 8*3600 + 600, 8*3600 + 1200}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := router.Search(ctx, Query{
		Departure:        singaporeTime(t, 2026, time.January, 5, 7, 55),
		MaxTravelSeconds: 20 * 60,
		Origins:          []StopSeed{{Stop: 0}},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Search() error = %v, want context.Canceled", err)
	}
}

func testRouter(t *testing.T, days [7]bool, departures []int, frequencies []gtfsdata.Frequency) *Router {
	t.Helper()
	stops := []gtfsdata.Stop{
		{ID: "S1", Name: "First", Lat: 1.30, Lon: 103.80, SourceLine: 2},
		{ID: "S2", Name: "Second", Lat: 1.31, Lon: 103.81, SourceLine: 3},
		{ID: "S3", Name: "Third", Lat: 1.32, Lon: 103.82, SourceLine: 4},
	}
	stopTimes := make([]gtfsdata.StopTime, len(stops))
	for index := range stops {
		stopTimes[index] = gtfsdata.StopTime{
			TripID:           "T1",
			StopID:           stops[index].ID,
			ArrivalSeconds:   departures[index],
			DepartureSeconds: departures[index],
			StopSequence:     index + 1,
			SourceLine:       index + 2,
		}
	}
	feed := &gtfsdata.Feed{
		Info: gtfsdata.FeedInfo{Version: "routing-test"},
		Agencies: []gtfsdata.Agency{
			{ID: "A", Name: "Test Transit", URL: "https://example.com", Timezone: "Asia/Singapore", SourceLine: 2},
		},
		Stops: stops,
		Routes: []gtfsdata.Route{
			{ID: "R1", AgencyID: "A", ShortName: "R1", LongName: "Test Route", Type: 3, SourceLine: 2},
		},
		Services: []gtfsdata.Service{
			{ID: "SERVICE", Days: days, StartDate: 20260101, EndDate: 20301231, SourceLine: 2},
		},
		Trips: []gtfsdata.Trip{
			{ID: "T1", RouteID: "R1", ServiceID: "SERVICE", DirectionID: 0, SourceLine: 2},
		},
		StopTimes:   stopTimes,
		Frequencies: frequencies,
	}
	artifact, err := graph.Build(feed, graph.Source{Filename: "test.zip", SHA256: "test"})
	if err != nil {
		t.Fatalf("graph.Build() error = %v", err)
	}
	router, err := New(artifact)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return router
}

func dailyService() [7]bool {
	return [7]bool{true, true, true, true, true, true, true}
}

func singaporeTime(t *testing.T, year int, month time.Month, day, hour, minute int) time.Time {
	t.Helper()
	location, err := time.LoadLocation("Asia/Singapore")
	if err != nil {
		t.Fatal(err)
	}
	return time.Date(year, month, day, hour, minute, 0, 0, location)
}
