package spatial

import (
	"fmt"
	"math"
)

// NewGrid constructs a contour-ready travel-time grid whose local coordinates
// are measured from origin. It is used by walking-network implementations that
// compute their costs outside this package while keeping contour projection in
// one place.
func NewGrid(
	origin Point,
	columns int,
	rows int,
	minX float64,
	minY float64,
	cellSizeMeters float64,
	thresholdSeconds float64,
	values []float64,
) (*Grid, error) {
	if !validPoint(origin) {
		return nil, fmt.Errorf("origin coordinate is invalid")
	}
	if columns < 2 || rows < 2 {
		return nil, fmt.Errorf("travel-time grid dimensions are invalid")
	}
	if columns > math.MaxInt/rows || len(values) != columns*rows {
		return nil, fmt.Errorf("travel-time grid values do not match its dimensions")
	}
	if !finiteGridValue(minX) || !finiteGridValue(minY) {
		return nil, fmt.Errorf("travel-time grid origin is invalid")
	}
	if cellSizeMeters <= 0 || !finiteGridValue(cellSizeMeters) {
		return nil, fmt.Errorf("travel-time grid cell size is invalid")
	}
	if thresholdSeconds <= 0 || !finiteGridValue(thresholdSeconds) {
		return nil, fmt.Errorf("travel-time grid threshold is invalid")
	}
	return &Grid{
		Columns:          columns,
		Rows:             rows,
		MinX:             minX,
		MinY:             minY,
		CellSizeMeters:   cellSizeMeters,
		ThresholdSeconds: thresholdSeconds,
		Values:           values,
		projection:       NewProjection(origin),
	}, nil
}

func finiteGridValue(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
