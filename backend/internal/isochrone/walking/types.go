// Package walking builds and loads the offline pedestrian raster used by the
// isochrone engine. Coordinates in this package are SVY21 metres (EPSG:3414).
package walking

import (
	"math"
)

const (
	FormatName    = "isomap-footpath-raster"
	SchemaVersion = 1

	// Fifteen metres is the finest tested resolution at which the supplied LTA
	// linework forms a useful dominant raster component (91.49% of occupied
	// cells). Finer cells expose endpoint gaps in the source geometry.
	DefaultCellSizeMeters   = 15.0
	DefaultMaximumGridCells = int64(250_000_000)
	DefaultMaximumShapeSize = int64(512 << 20)
)

// BuildConfig controls conversion of the source shapefile into a regular
// raster. Zero limits are replaced with their defaults so callers may set only
// CellSizeMeters when comparing resolutions.
type BuildConfig struct {
	CellSizeMeters       float64
	MaximumGridCells     int64
	MaximumShapefileSize int64
}

func DefaultBuildConfig() BuildConfig {
	return BuildConfig{
		CellSizeMeters:       DefaultCellSizeMeters,
		MaximumGridCells:     DefaultMaximumGridCells,
		MaximumShapefileSize: DefaultMaximumShapeSize,
	}
}

// Bounds describes the source geometry extent in projected SVY21 metres.
type Bounds struct {
	MinEasting  float64 `json:"min_easting"`
	MinNorthing float64 `json:"min_northing"`
	MaxEasting  float64 `json:"max_easting"`
	MaxNorthing float64 `json:"max_northing"`
}

type Manifest struct {
	Format                  string  `json:"format"`
	SchemaVersion           int     `json:"schema_version"`
	SourceFile              string  `json:"source_file"`
	SourceSHA256            string  `json:"source_sha256"`
	Projection              string  `json:"projection"`
	CellSizeMeters          float64 `json:"cell_size_meters"`
	Columns                 int     `json:"columns"`
	Rows                    int     `json:"rows"`
	ShapeRecordCount        int     `json:"shape_record_count"`
	PolylineRecordCount     int     `json:"polyline_record_count"`
	PartCount               int     `json:"part_count"`
	PointCount              int     `json:"point_count"`
	SegmentCount            int     `json:"segment_count"`
	RasterizedSampleCount   int64   `json:"rasterized_sample_count"`
	WalkableCellCount       int     `json:"walkable_cell_count"`
	ConnectedComponentCount int     `json:"connected_component_count"`
	LargestComponentCells   int     `json:"largest_component_cells"`
	Bounds                  Bounds  `json:"bounds"`
}

type Artifact struct {
	Manifest Manifest
	Network  Network
}

// Network is an implicit bidirectional 8-neighbour graph. A set bit means the
// corresponding cell is a walkable vertex; callers derive orthogonal and
// diagonal edges from adjacent set bits.
type Network struct {
	MinEasting     float64
	MinNorthing    float64
	CellSizeMeters float64
	Columns        int
	Rows           int
	WalkableBits   []uint64
}

func (network *Network) CellCount() int {
	if network == nil || network.Columns <= 0 || network.Rows <= 0 || network.Columns > maxInt()/network.Rows {
		return 0
	}
	return network.Columns * network.Rows
}

// Index maps a row and column to the packed raster index.
func (network *Network) Index(row, column int) (int, bool) {
	if network == nil || row < 0 || row >= network.Rows || column < 0 || column >= network.Columns {
		return 0, false
	}
	return row*network.Columns + column, true
}

func (network *Network) RowColumn(index int) (row, column int, ok bool) {
	if network == nil || index < 0 || index >= network.CellCount() {
		return 0, 0, false
	}
	return index / network.Columns, index % network.Columns, true
}

// CellCenter returns the SVY21 coordinate at the center of an indexed cell.
func (network *Network) CellCenter(index int) (easting, northing float64, ok bool) {
	row, column, ok := network.RowColumn(index)
	if !ok {
		return 0, 0, false
	}
	return network.MinEasting + (float64(column)+0.5)*network.CellSizeMeters,
		network.MinNorthing + (float64(row)+0.5)*network.CellSizeMeters,
		true
}

func (network *Network) CellCenterAt(row, column int) (easting, northing float64, ok bool) {
	index, ok := network.Index(row, column)
	if !ok {
		return 0, 0, false
	}
	return network.CellCenter(index)
}

// CellAt returns the cell containing a projected coordinate.
func (network *Network) CellAt(easting, northing float64) (int, bool) {
	if network == nil || !finite(easting) || !finite(northing) || network.CellSizeMeters <= 0 {
		return 0, false
	}
	maxEasting := network.MinEasting + float64(network.Columns)*network.CellSizeMeters
	maxNorthing := network.MinNorthing + float64(network.Rows)*network.CellSizeMeters
	if easting < network.MinEasting || easting >= maxEasting || northing < network.MinNorthing || northing >= maxNorthing {
		return 0, false
	}
	column := int(math.Floor((easting - network.MinEasting) / network.CellSizeMeters))
	row := int(math.Floor((northing - network.MinNorthing) / network.CellSizeMeters))
	return network.Index(row, column)
}

func (network *Network) IsWalkable(index int) bool {
	if network == nil || index < 0 || index >= network.CellCount() {
		return false
	}
	word := index >> 6
	return word < len(network.WalkableBits) && network.WalkableBits[word]&(uint64(1)<<uint(index&63)) != 0
}

// Walkable is an alias for IsWalkable.
func (network *Network) Walkable(index int) bool {
	return network.IsWalkable(index)
}

// Convenience methods keep query code terse when it owns the whole artifact.
func (artifact *Artifact) Walkable(index int) bool {
	return artifact != nil && artifact.Network.IsWalkable(index)
}

func (artifact *Artifact) Index(row, column int) (int, bool) {
	if artifact == nil {
		return 0, false
	}
	return artifact.Network.Index(row, column)
}

func (artifact *Artifact) CellCenter(row, column int) (easting, northing float64, ok bool) {
	if artifact == nil {
		return 0, 0, false
	}
	return artifact.Network.CellCenterAt(row, column)
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
