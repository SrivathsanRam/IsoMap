package spatial

import "testing"

func TestNewGridSetsProjectionAndValidatesShape(t *testing.T) {
	origin := Point{Lat: 1.35, Lon: 103.82}
	grid, err := NewGrid(origin, 2, 2, -10, -20, 10, 300, []float64{0, 1, 2, 3})
	if err != nil {
		t.Fatalf("NewGrid() error = %v", err)
	}
	if got := grid.GeographicPoint(LocalPoint{}); got != origin {
		t.Fatalf("grid origin = %+v, want %+v", got, origin)
	}
	if _, err := NewGrid(origin, 2, 2, 0, 0, 10, 300, []float64{0}); err == nil {
		t.Fatal("NewGrid accepted a mismatched values slice")
	}
}
