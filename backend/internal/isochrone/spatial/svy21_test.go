package spatial

import (
	"math"
	"testing"
)

func TestSVY21FalseOrigin(t *testing.T) {
	origin := Point{Lat: svy21LatitudeOfOrigin, Lon: svy21CentralMeridian}

	projected, err := SVY21Project(origin)
	if err != nil {
		t.Fatalf("project false origin: %v", err)
	}
	if difference := math.Abs(projected.X - svy21FalseEasting); difference > 1e-9 {
		t.Fatalf("false easting = %.12f, want %.12f (difference %.12g)", projected.X, svy21FalseEasting, difference)
	}
	if difference := math.Abs(projected.Y - svy21FalseNorthing); difference > 1e-9 {
		t.Fatalf("false northing = %.12f, want %.12f (difference %.12g)", projected.Y, svy21FalseNorthing, difference)
	}

	unprojected, err := SVY21Unproject(LocalPoint{X: svy21FalseEasting, Y: svy21FalseNorthing})
	if err != nil {
		t.Fatalf("unproject false origin: %v", err)
	}
	assertGeographicClose(t, unprojected, origin, 1e-11)
}

func TestSVY21RoundTripAcrossSingapore(t *testing.T) {
	points := []Point{
		{Lat: 1.290270, Lon: 103.851959}, // central Singapore
		{Lat: 1.352100, Lon: 103.819800},
		{Lat: 1.253364, Lon: 103.621046}, // Footpath dataset southwest bound
		{Lat: 1.469928, Lon: 104.021305}, // Footpath dataset northeast bound
	}

	for _, original := range points {
		projected, err := SVY21Project(original)
		if err != nil {
			t.Fatalf("project %+v: %v", original, err)
		}
		unprojected, err := SVY21Unproject(projected)
		if err != nil {
			t.Fatalf("unproject %+v as %+v: %v", original, projected, err)
		}
		assertGeographicClose(t, unprojected, original, 1e-9)
	}
}

func TestSVY21RejectsInvalidCoordinates(t *testing.T) {
	if _, err := SVY21Project(Point{Lat: math.NaN(), Lon: 103.8}); err == nil {
		t.Fatal("SVY21Project accepted a NaN latitude")
	}
	if _, err := SVY21Project(Point{Lat: 90, Lon: 103.8}); err == nil {
		t.Fatal("SVY21Project accepted a polar latitude")
	}
	if _, err := SVY21Unproject(LocalPoint{X: math.Inf(1), Y: svy21FalseNorthing}); err == nil {
		t.Fatal("SVY21Unproject accepted an infinite easting")
	}
}

func assertGeographicClose(t *testing.T, got, want Point, toleranceDegrees float64) {
	t.Helper()
	if latitudeDifference := math.Abs(got.Lat - want.Lat); latitudeDifference > toleranceDegrees {
		t.Errorf("latitude = %.12f, want %.12f (difference %.12g)", got.Lat, want.Lat, latitudeDifference)
	}
	if longitudeDifference := math.Abs(got.Lon - want.Lon); longitudeDifference > toleranceDegrees {
		t.Errorf("longitude = %.12f, want %.12f (difference %.12g)", got.Lon, want.Lon, longitudeDifference)
	}
}
