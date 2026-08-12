package contour

import (
	"context"
	"math"
	"testing"

	"github.com/SrivathsanRam/IsoMap/internal/isochrone/graph"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/spatial"
)

func TestMarchingSquaresCreatesDisconnectedMultiPolygon(t *testing.T) {
	origin := spatial.Point{Lat: 1.35, Lon: 103.82}
	projection := spatial.NewProjection(origin)
	distant := projection.Unproject(spatial.LocalPoint{X: 2000})
	grid := buildGrid(t, origin, []graph.Stop{{ID: "S1", Lat: distant.Lat, Lon: distant.Lon}}, []int32{300})

	geometry, err := MarchingSquares(grid, DefaultConfig(grid.CellSizeMeters))
	if err != nil {
		t.Fatalf("MarchingSquares() error = %v", err)
	}
	if geometry.Type != "MultiPolygon" {
		t.Fatalf("geometry type = %q, want MultiPolygon", geometry.Type)
	}
	if got, want := len(geometry.Coordinates), 2; got != want {
		t.Fatalf("polygon count = %d, want %d", got, want)
	}
	assertClosedGeometry(t, geometry)
}

func TestMarchingSquaresClassifiesHole(t *testing.T) {
	origin := spatial.Point{Lat: 1.35, Lon: 103.82}
	projection := spatial.NewProjection(origin)
	stops := make([]graph.Stop, 8)
	times := make([]int32, len(stops))
	for index := range stops {
		angle := 2 * math.Pi * float64(index) / float64(len(stops))
		point := projection.Unproject(spatial.LocalPoint{
			X: 3000 + 800*math.Cos(angle),
			Y: 800 * math.Sin(angle),
		})
		stops[index] = graph.Stop{ID: "ring-stop", Lat: point.Lat, Lon: point.Lon}
		times[index] = 100
	}
	grid := buildGrid(t, origin, stops, times)
	geometry, err := MarchingSquares(grid, DefaultConfig(grid.CellSizeMeters))
	if err != nil {
		t.Fatalf("MarchingSquares() error = %v", err)
	}

	foundHole := false
	for _, polygon := range geometry.Coordinates {
		if len(polygon) > 1 {
			foundHole = true
		}
	}
	if !foundHole {
		t.Fatalf("geometry has no polygon hole: %+v", geometry.Coordinates)
	}
	assertClosedGeometry(t, geometry)
}

func buildGrid(t *testing.T, origin spatial.Point, stops []graph.Stop, times []int32) *spatial.Grid {
	t.Helper()
	grid, err := spatial.BuildTravelTimeGrid(
		context.Background(),
		origin,
		stops,
		times,
		600,
		spatial.WalkingConfig{
			SpeedMetersPerSecond: 1,
			NetworkFactor:        1,
			CellSizeMeters:       50,
			MaximumGridCells:     50000,
		},
	)
	if err != nil {
		t.Fatalf("BuildTravelTimeGrid() error = %v", err)
	}
	return grid
}

func assertClosedGeometry(t *testing.T, geometry MultiPolygon) {
	t.Helper()
	for polygonIndex, polygon := range geometry.Coordinates {
		for ringIndex, ring := range polygon {
			if len(ring) < 4 {
				t.Fatalf("polygon %d ring %d has only %d positions", polygonIndex, ringIndex, len(ring))
			}
			first := ring[0]
			last := ring[len(ring)-1]
			if first[0] != last[0] || first[1] != last[1] {
				t.Fatalf("polygon %d ring %d is not closed", polygonIndex, ringIndex)
			}
		}
	}
}
