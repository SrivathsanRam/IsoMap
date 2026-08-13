package spatial

import (
	"fmt"
	"math"
)

// SVY21 is Singapore's projected coordinate system. The parameters below are
// taken from the Footpath_Mar2026 projection definition and describe a
// Transverse Mercator projection on the WGS 84 ellipsoid.
const (
	svy21SemiMajorAxis       = 6378137.0
	svy21InverseFlattening   = 298.257223563
	svy21LatitudeOfOrigin    = 1.366666666666667
	svy21CentralMeridian     = 103.8333333333333
	svy21ScaleFactor         = 1.0
	svy21FalseEasting        = 28001.642
	svy21FalseNorthing       = 38744.572
	svy21MaximumSeriesLatDeg = 89.0
)

var (
	svy21Flattening         = 1.0 / svy21InverseFlattening
	svy21SemiMinorAxis      = svy21SemiMajorAxis * (1.0 - svy21Flattening)
	svy21EccentricitySq     = 2.0*svy21Flattening - svy21Flattening*svy21Flattening
	svy21EccentricityFourth = svy21EccentricitySq * svy21EccentricitySq
	svy21EccentricitySixth  = svy21EccentricityFourth * svy21EccentricitySq
	svy21OriginLatRadians   = degreesToRadians(svy21LatitudeOfOrigin)
	svy21OriginLonRadians   = degreesToRadians(svy21CentralMeridian)
	svy21OriginArc          = svy21MeridionalArc(svy21OriginLatRadians)
)

// SVY21Project converts a WGS 84 latitude/longitude point to SVY21 easting
// (X) and northing (Y), in metres.
func SVY21Project(point Point) (LocalPoint, error) {
	if !validPoint(point) {
		return LocalPoint{}, fmt.Errorf("WGS 84 coordinate is invalid")
	}
	// The truncated Transverse Mercator series is singular at the poles. SVY21
	// is a local Singapore grid, so accepting a polar coordinate would hide a
	// caller error rather than provide a useful result.
	if math.Abs(point.Lat) >= svy21MaximumSeriesLatDeg {
		return LocalPoint{}, fmt.Errorf("WGS 84 latitude %.9f is outside the SVY21 projection domain", point.Lat)
	}

	latitude := degreesToRadians(point.Lat)
	longitude := degreesToRadians(point.Lon)
	sineLatitude := math.Sin(latitude)
	cosineLatitude := math.Cos(latitude)
	tangentLatitude := math.Tan(latitude)
	tangentSquared := tangentLatitude * tangentLatitude
	tangentFourth := tangentSquared * tangentSquared
	tangentSixth := tangentFourth * tangentSquared

	radiusPrimeVertical, radiusMeridional := svy21Radii(sineLatitude)
	psi := radiusPrimeVertical / radiusMeridional
	psiSquared := psi * psi
	psiCubed := psiSquared * psi
	psiFourth := psiSquared * psiSquared
	longitudeDelta := longitude - svy21OriginLonRadians
	longitudeDeltaSquared := longitudeDelta * longitudeDelta
	longitudeDeltaFourth := longitudeDeltaSquared * longitudeDeltaSquared
	longitudeDeltaSixth := longitudeDeltaFourth * longitudeDeltaSquared
	longitudeDeltaEighth := longitudeDeltaFourth * longitudeDeltaFourth
	cosineSquared := cosineLatitude * cosineLatitude
	cosineFourth := cosineSquared * cosineSquared
	cosineSixth := cosineFourth * cosineSquared

	northingTerm2 := longitudeDeltaSquared / 2.0 * radiusPrimeVertical * sineLatitude * cosineLatitude
	northingTerm4 := longitudeDeltaFourth / 24.0 * radiusPrimeVertical * sineLatitude * cosineLatitude * cosineSquared *
		(4.0*psiSquared + psi - tangentSquared)
	northingTerm6 := longitudeDeltaSixth / 720.0 * radiusPrimeVertical * sineLatitude * cosineLatitude * cosineFourth *
		(8.0*psiFourth*(11.0-24.0*tangentSquared) -
			28.0*psiCubed*(1.0-6.0*tangentSquared) +
			psiSquared*(1.0-32.0*tangentSquared) -
			2.0*psi*tangentSquared + tangentFourth)
	northingTerm8 := longitudeDeltaEighth / 40320.0 * radiusPrimeVertical * sineLatitude * cosineLatitude * cosineSixth *
		(1385.0 - 3111.0*tangentSquared + 543.0*tangentFourth - tangentSixth)
	northing := svy21FalseNorthing + svy21ScaleFactor*(svy21MeridionalArc(latitude)-svy21OriginArc+
		northingTerm2+northingTerm4+northingTerm6+northingTerm8)

	eastingTerm2 := longitudeDeltaSquared / 6.0 * cosineSquared * (psi - tangentSquared)
	eastingTerm4 := longitudeDeltaFourth / 120.0 * cosineFourth *
		(4.0*psiCubed*(1.0-6.0*tangentSquared) +
			psiSquared*(1.0+8.0*tangentSquared) -
			2.0*psi*tangentSquared + tangentFourth)
	eastingTerm6 := longitudeDeltaSixth / 5040.0 * cosineSixth *
		(61.0 - 479.0*tangentSquared + 179.0*tangentFourth - tangentSixth)
	easting := svy21FalseEasting + svy21ScaleFactor*radiusPrimeVertical*longitudeDelta*cosineLatitude*
		(1.0+eastingTerm2+eastingTerm4+eastingTerm6)

	if !svy21Finite(easting) || !svy21Finite(northing) {
		return LocalPoint{}, fmt.Errorf("SVY21 projection produced a non-finite coordinate")
	}
	return LocalPoint{X: easting, Y: northing}, nil
}

// SVY21Unproject converts an SVY21 easting (X) and northing (Y), in metres, to
// a WGS 84 latitude/longitude point.
func SVY21Unproject(point LocalPoint) (Point, error) {
	if !svy21Finite(point.X) || !svy21Finite(point.Y) {
		return Point{}, fmt.Errorf("SVY21 coordinate is invalid")
	}

	meridionalArc := svy21OriginArc + (point.Y-svy21FalseNorthing)/svy21ScaleFactor
	footpointLatitude := svy21FootpointLatitude(meridionalArc)
	if !svy21Finite(footpointLatitude) || math.Abs(footpointLatitude) >= degreesToRadians(svy21MaximumSeriesLatDeg) {
		return Point{}, fmt.Errorf("SVY21 northing %.3f is outside the projection domain", point.Y)
	}

	sineLatitude := math.Sin(footpointLatitude)
	cosineLatitude := math.Cos(footpointLatitude)
	tangentLatitude := math.Tan(footpointLatitude)
	tangentSquared := tangentLatitude * tangentLatitude
	tangentFourth := tangentSquared * tangentSquared
	tangentSixth := tangentFourth * tangentSquared
	radiusPrimeVertical, radiusMeridional := svy21Radii(sineLatitude)
	psi := radiusPrimeVertical / radiusMeridional
	psiSquared := psi * psi
	psiCubed := psiSquared * psi
	psiFourth := psiSquared * psiSquared

	eastingDelta := point.X - svy21FalseEasting
	x := eastingDelta / (svy21ScaleFactor * radiusPrimeVertical)
	xSquared := x * x
	xCubed := xSquared * x
	xFourth := xSquared * xSquared
	xFifth := xFourth * x
	xSixth := xFourth * xSquared
	xSeventh := xSixth * x
	xEighth := xFourth * xFourth

	latitudeTerm2 := tangentLatitude * radiusPrimeVertical / radiusMeridional * xSquared / 2.0
	latitudeTerm4 := tangentLatitude * radiusPrimeVertical / radiusMeridional * xFourth / 24.0 *
		(-4.0*psiSquared + 9.0*psi*(1.0-tangentSquared) + 12.0*tangentSquared)
	latitudeTerm6 := tangentLatitude * radiusPrimeVertical / radiusMeridional * xSixth / 720.0 *
		(8.0*psiFourth*(11.0-24.0*tangentSquared) -
			12.0*psiCubed*(21.0-71.0*tangentSquared) +
			15.0*psiSquared*(15.0-98.0*tangentSquared+15.0*tangentFourth) +
			180.0*psi*(5.0*tangentSquared-3.0*tangentFourth) + 360.0*tangentFourth)
	latitudeTerm8 := tangentLatitude * radiusPrimeVertical / radiusMeridional * xEighth / 40320.0 *
		(1385.0 + 3633.0*tangentSquared + 4095.0*tangentFourth + 1575.0*tangentSixth)
	latitude := footpointLatitude - latitudeTerm2 + latitudeTerm4 - latitudeTerm6 + latitudeTerm8

	longitudeTerm1 := x
	longitudeTerm3 := xCubed / 6.0 * (psi + 2.0*tangentSquared)
	longitudeTerm5 := xFifth / 120.0 *
		(-4.0*psiCubed*(1.0-6.0*tangentSquared) +
			psiSquared*(9.0-68.0*tangentSquared) +
			72.0*psi*tangentSquared + 24.0*tangentFourth)
	longitudeTerm7 := xSeventh / 5040.0 *
		(61.0 + 662.0*tangentSquared + 1320.0*tangentFourth + 720.0*tangentSixth)
	longitude := svy21OriginLonRadians +
		(longitudeTerm1-longitudeTerm3+longitudeTerm5-longitudeTerm7)/cosineLatitude

	result := Point{Lat: radiansToDegrees(latitude), Lon: radiansToDegrees(longitude)}
	if !validPoint(result) {
		return Point{}, fmt.Errorf("SVY21 inverse projection produced an invalid WGS 84 coordinate")
	}
	return result, nil
}

func svy21Radii(sineLatitude float64) (primeVertical float64, meridional float64) {
	denominator := 1.0 - svy21EccentricitySq*sineLatitude*sineLatitude
	primeVertical = svy21SemiMajorAxis / math.Sqrt(denominator)
	meridional = svy21SemiMajorAxis * (1.0 - svy21EccentricitySq) / math.Pow(denominator, 1.5)
	return primeVertical, meridional
}

func svy21MeridionalArc(latitude float64) float64 {
	a0 := 1.0 - svy21EccentricitySq/4.0 - 3.0*svy21EccentricityFourth/64.0 - 5.0*svy21EccentricitySixth/256.0
	a2 := 3.0 / 8.0 * (svy21EccentricitySq + svy21EccentricityFourth/4.0 + 15.0*svy21EccentricitySixth/128.0)
	a4 := 15.0 / 256.0 * (svy21EccentricityFourth + 3.0*svy21EccentricitySixth/4.0)
	a6 := 35.0 * svy21EccentricitySixth / 3072.0
	return svy21SemiMajorAxis * (a0*latitude - a2*math.Sin(2.0*latitude) + a4*math.Sin(4.0*latitude) - a6*math.Sin(6.0*latitude))
}

func svy21FootpointLatitude(meridionalArc float64) float64 {
	n := (svy21SemiMajorAxis - svy21SemiMinorAxis) / (svy21SemiMajorAxis + svy21SemiMinorAxis)
	nSquared := n * n
	nCubed := nSquared * n
	nFourth := nSquared * nSquared
	g := svy21SemiMajorAxis * (1.0 - n) * (1.0 - nSquared) *
		(1.0 + 9.0*nSquared/4.0 + 225.0*nFourth/64.0) * math.Pi / 180.0
	sigma := meridionalArc * math.Pi / (180.0 * g)
	return sigma +
		(3.0*n/2.0-27.0*nCubed/32.0)*math.Sin(2.0*sigma) +
		(21.0*nSquared/16.0-55.0*nFourth/32.0)*math.Sin(4.0*sigma) +
		151.0*nCubed/96.0*math.Sin(6.0*sigma) +
		1097.0*nFourth/512.0*math.Sin(8.0*sigma)
}

func svy21Finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
