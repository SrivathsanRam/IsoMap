package footpath

import (
	"context"
	"math"
	"testing"

	"github.com/SrivathsanRam/IsoMap/internal/isochrone/graph"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/spatial"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/walking"
)

func TestSeedsUseFootpathDistanceAndIgnoreUnsnappedStops(t *testing.T) {
	artifact := testWalkingArtifact(t, 7, 7, [][2]int{{3, 1}, {3, 2}, {3, 3}, {3, 4}, {3, 5}})
	origin := cellPoint(t, &artifact.Network, 3, 1)
	stops := []graph.Stop{
		{ID: "on-network", Lat: cellPoint(t, &artifact.Network, 3, 5).Lat, Lon: cellPoint(t, &artifact.Network, 3, 5).Lon},
		{ID: "off-network", Lat: cellPoint(t, &artifact.Network, 5, 1).Lat, Lon: cellPoint(t, &artifact.Network, 5, 1).Lon},
	}
	runtime := testRuntime(t, artifact, stops)

	seeds, err := runtime.Seeds(context.Background(), origin, 5*60)
	if err != nil {
		t.Fatalf("Seeds() error = %v", err)
	}
	if len(seeds) != 1 || seeds[0].Stop != 0 {
		t.Fatalf("Seeds() = %+v, want only the on-network stop", seeds)
	}
	if seeds[0].AccessSeconds < 79 || seeds[0].AccessSeconds > 81 {
		t.Fatalf("network access = %d seconds, want about 80", seeds[0].AccessSeconds)
	}
}

func TestBuildProducesPathConstrainedSurface(t *testing.T) {
	artifact := testWalkingArtifact(t, 9, 9, [][2]int{
		{4, 1}, {4, 2}, {4, 3}, {4, 4}, {4, 5}, {4, 6}, {4, 7},
	})
	origin := cellPoint(t, &artifact.Network, 4, 1)
	runtime := testRuntime(t, artifact, nil)

	grid, err := runtime.Build(context.Background(), origin, nil, nil, 90)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if valueNear(grid, spatial.LocalPoint{X: 80, Y: 0}) > grid.ThresholdSeconds {
		t.Fatal("point along the footpath was not reachable")
	}
	if valueNear(grid, spatial.LocalPoint{X: 0, Y: 60}) <= grid.ThresholdSeconds {
		t.Fatal("surface expanded radially beyond the bounded terminal connector")
	}
}

func TestBuildSeedsPostTransitWalkingAtArrivalCost(t *testing.T) {
	artifact := testWalkingArtifact(t, 12, 12, [][2]int{
		{2, 1}, {2, 2},
		{8, 7}, {8, 8}, {8, 9}, {8, 10},
	})
	origin := cellPoint(t, &artifact.Network, 2, 1)
	stopPoint := cellPoint(t, &artifact.Network, 8, 7)
	stops := []graph.Stop{{ID: "transit-arrival", Lat: stopPoint.Lat, Lon: stopPoint.Lon}}
	runtime := testRuntime(t, artifact, stops)

	grid, err := runtime.Build(context.Background(), origin, stops, []int32{30}, 100)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	projection := spatial.NewProjection(origin)
	arrivalLocal := projection.Project(stopPoint)
	if valueNear(grid, arrivalLocal) > 35 {
		t.Fatalf("transit arrival source cost = %.1f, want about 30 seconds", valueNear(grid, arrivalLocal))
	}
	east := spatial.LocalPoint{X: arrivalLocal.X + 60, Y: arrivalLocal.Y}
	if valueNear(grid, east) > grid.ThresholdSeconds {
		t.Fatal("post-transit walking did not expand along the second footpath component")
	}
}

func TestNewRuntimeRejectsFragmentedWalkingArtifact(t *testing.T) {
	artifact := testWalkingArtifact(t, 3, 3, [][2]int{{0, 0}, {2, 2}})
	artifact.Manifest.LargestComponentCells = 1
	fallback := spatial.DefaultWalkingConfig()
	if _, err := NewRuntime(
		artifact,
		nil,
		spatial.DefaultAccessConfig(),
		DefaultConfig(fallback),
	); err == nil {
		t.Fatal("NewRuntime accepted an artifact whose dominant component is below the quality gate")
	}
}

func testRuntime(t *testing.T, artifact *walking.Artifact, stops []graph.Stop) *Runtime {
	t.Helper()
	fallback := spatial.DefaultWalkingConfig()
	fallback.SpeedMetersPerSecond = 1
	config := DefaultConfig(fallback)
	config.SpeedMetersPerSecond = 1
	config.MaximumSnapDistanceMeters = 5
	config.MaximumTerminalDistanceMeters = 25
	config.SurfaceCellSizeMeters = 10
	config.MaximumSurfaceGridCells = 10000
	runtime, err := NewRuntime(artifact, stops, spatial.AccessConfig{
		MaximumDistanceMeters: 200,
		MaximumStops:          10,
	}, config)
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	return runtime
}

func testWalkingArtifact(t *testing.T, rows, columns int, cells [][2]int) *walking.Artifact {
	t.Helper()
	network := walking.Network{
		MinEasting:     28000,
		MinNorthing:    38700,
		CellSizeMeters: 20,
		Columns:        columns,
		Rows:           rows,
		WalkableBits:   make([]uint64, (rows*columns+63)/64),
	}
	for _, cell := range cells {
		index, ok := network.Index(cell[0], cell[1])
		if !ok {
			t.Fatalf("test cell %+v is outside network", cell)
		}
		network.WalkableBits[index>>6] |= uint64(1) << uint(index&63)
	}
	return &walking.Artifact{
		Manifest: walking.Manifest{
			Format:                walking.FormatName,
			SchemaVersion:         walking.SchemaVersion,
			WalkableCellCount:     len(cells),
			LargestComponentCells: len(cells),
		},
		Network: network,
	}
}

func cellPoint(t *testing.T, network *walking.Network, row, column int) spatial.Point {
	t.Helper()
	easting, northing, ok := network.CellCenterAt(row, column)
	if !ok {
		t.Fatalf("cell (%d,%d) is invalid", row, column)
	}
	point, err := spatial.SVY21Unproject(spatial.LocalPoint{X: easting, Y: northing})
	if err != nil {
		t.Fatalf("unproject cell: %v", err)
	}
	return point
}

func valueNear(grid *spatial.Grid, point spatial.LocalPoint) float64 {
	column := int(math.Round((point.X - grid.MinX) / grid.CellSizeMeters))
	row := int(math.Round((point.Y - grid.MinY) / grid.CellSizeMeters))
	if column < 0 || column >= grid.Columns || row < 0 || row >= grid.Rows {
		return math.Inf(1)
	}
	return grid.Value(column, row)
}
