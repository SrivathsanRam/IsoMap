package footpath_test

import (
	"context"
	"math"
	"os"
	"testing"
	"time"

	engine "github.com/SrivathsanRam/IsoMap/internal/isochrone"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/contour"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/footpath"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/graph"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/spatial"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/walking"
)

func TestSingaporeFootpathTransitSmoke(t *testing.T) {
	transitPath := os.Getenv("ISOMAP_GTFS_GRAPH")
	walkingPath := os.Getenv("ISOMAP_FOOTPATH_GRAPH")
	if transitPath == "" || walkingPath == "" {
		t.Skip("set ISOMAP_GTFS_GRAPH and ISOMAP_FOOTPATH_GRAPH to run the full walking-network smoke test")
	}
	transitArtifact, err := graph.LoadArtifact(transitPath)
	if err != nil {
		t.Fatalf("load transit artifact: %v", err)
	}
	walkingArtifact, err := walking.LoadArtifact(walkingPath)
	if err != nil {
		t.Fatalf("load walking artifact: %v", err)
	}
	baseConfig := engine.DefaultConfig()
	walkingConfig := footpath.DefaultConfig(baseConfig.Walking)
	runtime, err := footpath.NewRuntime(walkingArtifact, transitArtifact.Graph.Stops, baseConfig.Access, walkingConfig)
	if err != nil {
		t.Fatalf("initialize footpath runtime: %v", err)
	}
	isochrone, err := engine.NewWithWalkingProviders(
		transitArtifact,
		runtime,
		runtime,
		contour.DefaultConfig(walkingConfig.SurfaceCellSizeMeters),
		footpath.ModelName,
	)
	if err != nil {
		t.Fatalf("initialize isochrone engine: %v", err)
	}
	location, err := time.LoadLocation("Asia/Singapore")
	if err != nil {
		t.Fatal(err)
	}
	result, err := isochrone.Generate(context.Background(), engine.Query{
		Origin:           spatial.Point{Lat: 1.381758, Lon: 103.844801},
		Departure:        time.Date(2026, time.August, 13, 8, 0, 0, 0, location),
		MaxTravelSeconds: 45 * 60,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !result.TransitUsed || result.ReachedStopCount < 100 {
		t.Fatalf("unexpected transit result: used=%t reached=%d", result.TransitUsed, result.ReachedStopCount)
	}
	if result.WalkingModel != footpath.ModelName || len(result.Geometry.Coordinates) == 0 {
		t.Fatalf("unexpected walking result: model=%q polygons=%d", result.WalkingModel, len(result.Geometry.Coordinates))
	}
	t.Logf(
		"reached %d stops, settled %d transit nodes, rasterized %dx%d, generated %d polygons",
		result.ReachedStopCount,
		result.SettledNodeCount,
		result.GridColumns,
		result.GridRows,
		len(result.Geometry.Coordinates),
	)
}

func TestSingaporeFootpathWalkingSurfaceIsNotRadial(t *testing.T) {
	walkingPath := os.Getenv("ISOMAP_FOOTPATH_GRAPH")
	if walkingPath == "" {
		t.Skip("set ISOMAP_FOOTPATH_GRAPH to run the full walking-network smoke test")
	}
	walkingArtifact, err := walking.LoadArtifact(walkingPath)
	if err != nil {
		t.Fatalf("load walking artifact: %v", err)
	}
	baseConfig := engine.DefaultConfig()
	walkingConfig := footpath.DefaultConfig(baseConfig.Walking)
	runtime, err := footpath.NewRuntime(walkingArtifact, nil, baseConfig.Access, walkingConfig)
	if err != nil {
		t.Fatalf("initialize footpath runtime: %v", err)
	}
	origin := spatial.Point{Lat: 1.381758, Lon: 103.844801}
	unreachable := []int32{}
	grid, err := runtime.Build(context.Background(), origin, nil, unreachable, 15*60)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	geometry, err := contour.MarchingSquares(grid, contour.DefaultConfig(walkingConfig.SurfaceCellSizeMeters))
	if err != nil {
		t.Fatalf("MarchingSquares() error = %v", err)
	}
	minimumRadius, maximumRadius := math.Inf(1), 0.0
	projection := spatial.NewProjection(origin)
	for _, polygon := range geometry.Coordinates {
		for _, ring := range polygon {
			for _, coordinate := range ring {
				local := projection.Project(spatial.Point{Lon: coordinate[0], Lat: coordinate[1]})
				radius := math.Hypot(local.X, local.Y)
				minimumRadius = math.Min(minimumRadius, radius)
				maximumRadius = math.Max(maximumRadius, radius)
			}
		}
	}
	if len(geometry.Coordinates) == 0 || !math.IsInf(minimumRadius, 0) && maximumRadius/minimumRadius < 1.25 {
		t.Fatalf(
			"walking contour still looks radial: polygons=%d min radius=%.1fm max radius=%.1fm",
			len(geometry.Coordinates),
			minimumRadius,
			maximumRadius,
		)
	}
	t.Logf(
		"walking-only contour has %d polygons and boundary radii %.1f..%.1fm",
		len(geometry.Coordinates),
		minimumRadius,
		maximumRadius,
	)
}
