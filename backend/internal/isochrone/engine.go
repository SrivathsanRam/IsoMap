package isochrone

import (
	"context"
	"fmt"
	"time"

	"github.com/SrivathsanRam/IsoMap/internal/isochrone/contour"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/graph"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/routing"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/spatial"
)

type Config struct {
	Walking spatial.WalkingConfig
	Contour contour.Config
}

func DefaultConfig() Config {
	walking := spatial.DefaultWalkingConfig()
	return Config{
		Walking: walking,
		Contour: contour.DefaultConfig(walking.CellSizeMeters),
	}
}

type Query struct {
	Origin           spatial.Point
	Departure        time.Time
	MaxTravelSeconds int
	OriginStops      []routing.StopSeed
}

type Result struct {
	Geometry         contour.MultiPolygon
	ReachedStopCount int
	SettledNodeCount int
	GridColumns      int
	GridRows         int
}

// SurfaceBuilder is the replaceable walking-cost boundary. The built-in
// implementation is deterministic and radial; an offline OSM walker or a
// batched routing provider can instead produce the same travel-time grid.
type SurfaceBuilder interface {
	Build(
		ctx context.Context,
		origin spatial.Point,
		stops []graph.Stop,
		stopTravelSeconds []int32,
		maxTravelSeconds int,
	) (*spatial.Grid, error)
}

// Engine composes transit routing, walking egress, travel-time rasterization,
// and marching squares. OriginStops are deliberately supplied by the caller
// until the separate nearest-stop access index is implemented.
type Engine struct {
	graph   *graph.Graph
	router  *routing.Router
	surface SurfaceBuilder
	contour contour.Config
}

func New(artifact *graph.Artifact, config Config) (*Engine, error) {
	return NewWithSurface(artifact, radialWalkingSurface{config: config.Walking}, config.Contour)
}

func NewWithSurface(artifact *graph.Artifact, surface SurfaceBuilder, contourConfig contour.Config) (*Engine, error) {
	if surface == nil {
		return nil, fmt.Errorf("walking surface builder is nil")
	}
	router, err := routing.New(artifact)
	if err != nil {
		return nil, err
	}
	return &Engine{graph: &artifact.Graph, router: router, surface: surface, contour: contourConfig}, nil
}

func (e *Engine) Generate(ctx context.Context, query Query) (*Result, error) {
	if e == nil || e.router == nil || e.graph == nil {
		return nil, fmt.Errorf("isochrone engine is not initialized")
	}
	routingResult, err := e.router.Search(ctx, routing.Query{
		Departure:        query.Departure,
		MaxTravelSeconds: query.MaxTravelSeconds,
		Origins:          query.OriginStops,
	})
	if err != nil {
		return nil, fmt.Errorf("route transit graph: %w", err)
	}
	grid, err := e.surface.Build(
		ctx,
		query.Origin,
		e.graph.Stops,
		routingResult.StopTravelSeconds,
		query.MaxTravelSeconds,
	)
	if err != nil {
		return nil, fmt.Errorf("build travel-time surface: %w", err)
	}
	geometry, err := contour.MarchingSquares(grid, e.contour)
	if err != nil {
		return nil, fmt.Errorf("extract isochrone contour: %w", err)
	}
	return &Result{
		Geometry:         geometry,
		ReachedStopCount: routingResult.ReachedStopCount,
		SettledNodeCount: routingResult.SettledNodeCount,
		GridColumns:      grid.Columns,
		GridRows:         grid.Rows,
	}, nil
}

type radialWalkingSurface struct {
	config spatial.WalkingConfig
}

func (s radialWalkingSurface) Build(
	ctx context.Context,
	origin spatial.Point,
	stops []graph.Stop,
	stopTravelSeconds []int32,
	maxTravelSeconds int,
) (*spatial.Grid, error) {
	return spatial.BuildTravelTimeGrid(ctx, origin, stops, stopTravelSeconds, maxTravelSeconds, s.config)
}
