package graph

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	gtfsdata "github.com/SrivathsanRam/IsoMap/internal/isochrone/gtfs"
)

func TestBuildExpandedGraph(t *testing.T) {
	artifact, err := Build(testFeed(), Source{Filename: "test.zip", SHA256: "abc123"})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if got, want := artifact.Manifest.NodeCount, 6; got != want {
		t.Fatalf("NodeCount = %d, want %d", got, want)
	}
	if got, want := artifact.Manifest.Timezone, "Asia/Singapore"; got != want {
		t.Fatalf("Timezone = %q, want %q", got, want)
	}
	if got, want := artifact.Manifest.EdgeCount, 7; got != want {
		t.Fatalf("EdgeCount = %d, want %d", got, want)
	}
	if got, want := len(artifact.Graph.FirstOut), artifact.Manifest.NodeCount+1; got != want {
		t.Fatalf("FirstOut length = %d, want %d", got, want)
	}
	if got, want := artifact.Graph.FirstOut[len(artifact.Graph.FirstOut)-1], uint32(artifact.Manifest.EdgeCount); got != want {
		t.Fatalf("last FirstOut = %d, want %d", got, want)
	}

	var board *Edge
	start, end := artifact.Graph.FirstOut[0], artifact.Graph.FirstOut[1]
	for index := start; index < end; index++ {
		edge := &artifact.Graph.Edges[index]
		if edge.Kind == EdgeBoard {
			board = edge
			break
		}
	}
	if board == nil {
		t.Fatal("origin stop has no boarding edge")
	}
	if board.Trip != 0 || board.DepartureOffsetSeconds != 0 || board.ScheduledDepartureSeconds != 8*3600 {
		t.Fatalf("unexpected board edge: %+v", *board)
	}
	if got := artifact.Graph.Trips[0].Frequencies[0].HeadwaySeconds; got != 600 {
		t.Fatalf("HeadwaySeconds = %d, want 600", got)
	}
}

func TestArtifactRoundTripIsDeterministic(t *testing.T) {
	artifact, err := Build(testFeed(), Source{Filename: "test.zip", SHA256: "abc123"})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	directory := t.TempDir()
	firstPath := filepath.Join(directory, "first.graph")
	secondPath := filepath.Join(directory, "second.graph")
	if err := WriteArtifact(firstPath, artifact); err != nil {
		t.Fatalf("WriteArtifact(first) error = %v", err)
	}
	if err := WriteArtifact(secondPath, artifact); err != nil {
		t.Fatalf("WriteArtifact(second) error = %v", err)
	}
	first, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("repeated artifact writes produced different bytes")
	}

	loaded, err := LoadArtifact(firstPath)
	if err != nil {
		t.Fatalf("LoadArtifact() error = %v", err)
	}
	if !reflect.DeepEqual(loaded, artifact) {
		t.Fatal("loaded artifact does not equal the written artifact")
	}
}

func testFeed() *gtfsdata.Feed {
	return &gtfsdata.Feed{
		Info: gtfsdata.FeedInfo{Version: "test-v1"},
		Agencies: []gtfsdata.Agency{
			{ID: "A", Name: "Test Transit", URL: "https://example.com", Timezone: "Asia/Singapore", SourceLine: 2},
		},
		Stops: []gtfsdata.Stop{
			{ID: "S1", Name: "First", Lat: 1.30, Lon: 103.80, SourceLine: 2},
			{ID: "S2", Name: "Second", Lat: 1.31, Lon: 103.81, SourceLine: 3},
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
		},
		Frequencies: []gtfsdata.Frequency{
			{TripID: "T1", StartSeconds: 8 * 3600, EndSeconds: 10 * 3600, HeadwaySeconds: 600, SourceLine: 2},
		},
		Transfers: []gtfsdata.Transfer{
			{FromStopID: "S2", ToStopID: "S1", Type: 2, MinTransferSeconds: 60, SourceLine: 2},
		},
	}
}
