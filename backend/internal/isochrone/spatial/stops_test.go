package spatial

import (
	"testing"

	"github.com/SrivathsanRam/IsoMap/internal/isochrone/graph"
)

func TestStopIndexReturnsNearestBoardingStops(t *testing.T) {
	origin := Point{Lat: 1.35, Lon: 103.82}
	projection := NewProjection(origin)
	near := projection.Unproject(LocalPoint{X: 100})
	far := projection.Unproject(LocalPoint{X: 400})
	station := projection.Unproject(LocalPoint{X: 50})
	index := NewStopIndex([]graph.Stop{
		{ID: "far", Lat: far.Lat, Lon: far.Lon},
		{ID: "near", Lat: near.Lat, Lon: near.Lon},
		{ID: "station", Lat: station.Lat, Lon: station.Lon, LocationType: 1},
	})

	result, err := index.Nearby(origin, AccessConfig{MaximumDistanceMeters: 500, MaximumStops: 1})
	if err != nil {
		t.Fatalf("Nearby() error = %v", err)
	}
	if len(result) != 1 || result[0].Stop != 1 {
		t.Fatalf("Nearby() = %+v, want boarding stop index 1", result)
	}
}
