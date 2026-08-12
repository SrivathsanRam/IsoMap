package spatial

import (
	"context"
	"fmt"
	"math"

	"github.com/SrivathsanRam/IsoMap/internal/isochrone/graph"
)

const (
	earthRadiusMeters       = 6371008.8
	DefaultWalkingSpeedMPS  = 1.3
	DefaultNetworkFactor    = 1.2
	DefaultCellSizeMeters   = 150.0
	DefaultMaximumGridCells = 300000
)

type Point struct {
	Lat float64
	Lon float64
}

// WalkingConfig controls the local walking approximation used to construct a
// travel-time surface. NetworkFactor accounts for paths being longer than the
// straight-line distance. Set it to 1 for pure crow-flight walking.
type WalkingConfig struct {
	SpeedMetersPerSecond float64
	NetworkFactor        float64
	CellSizeMeters       float64
	MaximumGridCells     int
}

func DefaultWalkingConfig() WalkingConfig {
	return WalkingConfig{
		SpeedMetersPerSecond: DefaultWalkingSpeedMPS,
		NetworkFactor:        DefaultNetworkFactor,
		CellSizeMeters:       DefaultCellSizeMeters,
		MaximumGridCells:     DefaultMaximumGridCells,
	}
}

// Grid stores travel times at regularly spaced vertices, ordered from west to
// east and south to north. Values contains seconds from the original query
// departure, not seconds from a stop.
type Grid struct {
	Columns          int
	Rows             int
	MinX             float64
	MinY             float64
	CellSizeMeters   float64
	ThresholdSeconds float64
	Values           []float64
	projection       Projection
}

func (g *Grid) Value(column, row int) float64 {
	return g.Values[row*g.Columns+column]
}

func (g *Grid) LocalPoint(column, row int) LocalPoint {
	return LocalPoint{
		X: g.MinX + float64(column)*g.CellSizeMeters,
		Y: g.MinY + float64(row)*g.CellSizeMeters,
	}
}

func (g *Grid) GeographicPoint(local LocalPoint) Point {
	return g.projection.Unproject(local)
}

type timedSource struct {
	point         LocalPoint
	travelSeconds float64
}

// BuildTravelTimeGrid converts reached transit stops into a continuous local
// travel-time surface. It also includes direct walking from the origin. A stop
// reached at time t contributes a walking radius based on budget-t.
func BuildTravelTimeGrid(
	ctx context.Context,
	origin Point,
	stops []graph.Stop,
	stopTravelSeconds []int32,
	maxTravelSeconds int,
	config WalkingConfig,
) (*Grid, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !validPoint(origin) {
		return nil, fmt.Errorf("origin coordinate is invalid")
	}
	if len(stops) != len(stopTravelSeconds) {
		return nil, fmt.Errorf("stop travel-time count %d does not match stop count %d", len(stopTravelSeconds), len(stops))
	}
	if maxTravelSeconds <= 0 {
		return nil, fmt.Errorf("max travel time must be positive")
	}
	if err := validateWalkingConfig(config); err != nil {
		return nil, err
	}

	projection := NewProjection(origin)
	sources := make([]timedSource, 0, len(stops)+1)
	sources = append(sources, timedSource{point: projection.Project(origin)})
	for index, travelSeconds := range stopTravelSeconds {
		if travelSeconds < 0 || int(travelSeconds) > maxTravelSeconds {
			continue
		}
		stopPoint := Point{Lat: stops[index].Lat, Lon: stops[index].Lon}
		if !validPoint(stopPoint) {
			return nil, fmt.Errorf("stop %q has an invalid coordinate", stops[index].ID)
		}
		sources = append(sources, timedSource{
			point:         projection.Project(stopPoint),
			travelSeconds: float64(travelSeconds),
		})
	}

	walkingMetersPerSecond := config.SpeedMetersPerSecond / config.NetworkFactor
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, source := range sources {
		remaining := float64(maxTravelSeconds) - source.travelSeconds
		if remaining < 0 {
			continue
		}
		radius := remaining * walkingMetersPerSecond
		minX = math.Min(minX, source.point.X-radius)
		minY = math.Min(minY, source.point.Y-radius)
		maxX = math.Max(maxX, source.point.X+radius)
		maxY = math.Max(maxY, source.point.Y+radius)
	}

	// Two guaranteed-outside cells make every contour close inside the raster.
	padding := 2 * config.CellSizeMeters
	minX = math.Floor((minX-padding)/config.CellSizeMeters) * config.CellSizeMeters
	minY = math.Floor((minY-padding)/config.CellSizeMeters) * config.CellSizeMeters
	maxX = math.Ceil((maxX+padding)/config.CellSizeMeters) * config.CellSizeMeters
	maxY = math.Ceil((maxY+padding)/config.CellSizeMeters) * config.CellSizeMeters
	columns := int(math.Round((maxX-minX)/config.CellSizeMeters)) + 1
	rows := int(math.Round((maxY-minY)/config.CellSizeMeters)) + 1
	if columns < 2 || rows < 2 {
		return nil, fmt.Errorf("travel-time grid dimensions are invalid")
	}
	cellCount := int64(columns) * int64(rows)
	if cellCount > int64(config.MaximumGridCells) {
		return nil, fmt.Errorf(
			"travel-time grid requires %d cells, exceeding the configured maximum of %d; increase cell size",
			cellCount,
			config.MaximumGridCells,
		)
	}

	secondsPerMeter := config.NetworkFactor / config.SpeedMetersPerSecond
	outsideValue := float64(maxTravelSeconds) + 2*math.Sqrt2*config.CellSizeMeters*secondsPerMeter
	grid := &Grid{
		Columns:          columns,
		Rows:             rows,
		MinX:             minX,
		MinY:             minY,
		CellSizeMeters:   config.CellSizeMeters,
		ThresholdSeconds: float64(maxTravelSeconds),
		Values:           make([]float64, int(cellCount)),
		projection:       projection,
	}
	for index := range grid.Values {
		grid.Values[index] = outsideValue
	}

	// Evaluate one cell beyond each source's reachable radius. These outside
	// samples let marching squares interpolate the threshold accurately.
	margin := math.Sqrt2 * config.CellSizeMeters
	for sourceIndex, source := range sources {
		if sourceIndex%32 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		remaining := float64(maxTravelSeconds) - source.travelSeconds
		if remaining < 0 {
			continue
		}
		radius := remaining*walkingMetersPerSecond + margin
		minimumColumn := clampInt(int(math.Floor((source.point.X-radius-minX)/config.CellSizeMeters)), 0, columns-1)
		maximumColumn := clampInt(int(math.Ceil((source.point.X+radius-minX)/config.CellSizeMeters)), 0, columns-1)
		minimumRow := clampInt(int(math.Floor((source.point.Y-radius-minY)/config.CellSizeMeters)), 0, rows-1)
		maximumRow := clampInt(int(math.Ceil((source.point.Y+radius-minY)/config.CellSizeMeters)), 0, rows-1)

		for row := minimumRow; row <= maximumRow; row++ {
			y := minY + float64(row)*config.CellSizeMeters
			for column := minimumColumn; column <= maximumColumn; column++ {
				x := minX + float64(column)*config.CellSizeMeters
				distance := math.Hypot(x-source.point.X, y-source.point.Y)
				candidate := source.travelSeconds + distance*secondsPerMeter
				valueIndex := row*columns + column
				if candidate < grid.Values[valueIndex] {
					grid.Values[valueIndex] = candidate
				}
			}
		}
	}

	return grid, nil
}

func validateWalkingConfig(config WalkingConfig) error {
	if config.SpeedMetersPerSecond <= 0 || math.IsNaN(config.SpeedMetersPerSecond) || math.IsInf(config.SpeedMetersPerSecond, 0) {
		return fmt.Errorf("walking speed must be a finite positive number")
	}
	if config.NetworkFactor < 1 || math.IsNaN(config.NetworkFactor) || math.IsInf(config.NetworkFactor, 0) {
		return fmt.Errorf("walking network factor must be finite and at least 1")
	}
	if config.CellSizeMeters <= 0 || math.IsNaN(config.CellSizeMeters) || math.IsInf(config.CellSizeMeters, 0) {
		return fmt.Errorf("grid cell size must be a finite positive number")
	}
	if config.MaximumGridCells <= 0 {
		return fmt.Errorf("maximum grid cells must be positive")
	}
	return nil
}

func validPoint(point Point) bool {
	return !math.IsNaN(point.Lat) && !math.IsNaN(point.Lon) &&
		!math.IsInf(point.Lat, 0) && !math.IsInf(point.Lon, 0) &&
		point.Lat >= -90 && point.Lat <= 90 && point.Lon >= -180 && point.Lon <= 180
}

func clampInt(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}
