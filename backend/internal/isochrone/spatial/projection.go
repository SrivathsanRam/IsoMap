package spatial

import "math"

type LocalPoint struct {
	X float64
	Y float64
}

// Projection is a local equirectangular projection centered on the query
// origin. Singapore's small geographic extent keeps its distortion negligible
// relative to the raster resolution.
type Projection struct {
	referenceLatRadians float64
	referenceLonRadians float64
	longitudeScale      float64
}

func NewProjection(reference Point) Projection {
	latitude := degreesToRadians(reference.Lat)
	return Projection{
		referenceLatRadians: latitude,
		referenceLonRadians: degreesToRadians(reference.Lon),
		longitudeScale:      math.Cos(latitude),
	}
}

func (p Projection) Project(point Point) LocalPoint {
	return LocalPoint{
		X: earthRadiusMeters * (degreesToRadians(point.Lon)-p.referenceLonRadians) * p.longitudeScale,
		Y: earthRadiusMeters * (degreesToRadians(point.Lat)-p.referenceLatRadians),
	}
}

func (p Projection) Unproject(point LocalPoint) Point {
	return Point{
		Lat: radiansToDegrees(point.Y/earthRadiusMeters + p.referenceLatRadians),
		Lon: radiansToDegrees(point.X/(earthRadiusMeters*p.longitudeScale) + p.referenceLonRadians),
	}
}

func degreesToRadians(value float64) float64 { return value * math.Pi / 180 }

func radiansToDegrees(value float64) float64 { return value * 180 / math.Pi }
