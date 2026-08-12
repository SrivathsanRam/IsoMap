package spatial

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/SrivathsanRam/IsoMap/internal/isochrone/graph"
)

func TestProjectionRoundTrip(t *testing.T) {
	reference := Point{Lat: 1.3521, Lon: 103.8198}
	point := Point{Lat: 1.401, Lon: 103.91}
	projection := NewProjection(reference)
	result := projection.Unproject(projection.Project(point))
	if math.Abs(result.Lat-point.Lat) > 1e-10 || math.Abs(result.Lon-point.Lon) > 1e-10 {
		t.Fatalf("round trip = %+v, want %+v", result, point)
	}
}

func TestBuildTravelTimeGridIncludesDirectAndPostTransitWalking(t *testing.T) {
	origin := Point{Lat: 1.35, Lon: 103.82}
	projection := NewProjection(origin)
	distant := projection.Unproject(LocalPoint{X: 2000, Y: 0})
	config := WalkingConfig{
		SpeedMetersPerSecond: 1,
		NetworkFactor:        1,
		CellSizeMeters:       50,
		MaximumGridCells:     10000,
	}
	grid, err := BuildTravelTimeGrid(
		context.Background(),
		origin,
		[]graph.Stop{{ID: "S1", Lat: distant.Lat, Lon: distant.Lon}},
		[]int32{300},
		600,
		config,
	)
	if err != nil {
		t.Fatalf("BuildTravelTimeGrid() error = %v", err)
	}

	originValue := nearestValue(grid, LocalPoint{})
	if originValue > 1 {
		t.Fatalf("origin grid value = %v, want approximately 0", originValue)
	}
	stopValue := nearestValue(grid, projection.Project(distant))
	if math.Abs(stopValue-300) > 40 {
		t.Fatalf("reached stop grid value = %v, want approximately 300", stopValue)
	}
	midpointValue := nearestValue(grid, LocalPoint{X: 1000})
	if midpointValue <= 600 {
		t.Fatalf("gap midpoint grid value = %v, want outside the 600-second threshold", midpointValue)
	}
}

func TestBuildTravelTimeGridEnforcesCellLimit(t *testing.T) {
	config := DefaultWalkingConfig()
	config.CellSizeMeters = 10
	config.MaximumGridCells = 100
	_, err := BuildTravelTimeGrid(
		context.Background(),
		Point{Lat: 1.35, Lon: 103.82},
		nil,
		nil,
		3600,
		config,
	)
	if err == nil || !strings.Contains(err.Error(), "exceeding the configured maximum") {
		t.Fatalf("BuildTravelTimeGrid() error = %v, want cell-limit error", err)
	}
}

func nearestValue(grid *Grid, point LocalPoint) float64 {
	column := int(math.Round((point.X - grid.MinX) / grid.CellSizeMeters))
	row := int(math.Round((point.Y - grid.MinY) / grid.CellSizeMeters))
	return grid.Value(column, row)
}
