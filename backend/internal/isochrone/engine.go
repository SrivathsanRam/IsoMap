package isochrone

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/SrivathsanRam/IsoMap/internal/isochrone/contour"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/graph"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/routing"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/spatial"
)

type Config struct {
	Walking spatial.WalkingConfig
	Access  spatial.AccessConfig
	Contour contour.Config
}

func DefaultConfig() Config {
	walking := spatial.DefaultWalkingConfig()
	return Config{
		Walking: walking,
		Access:  spatial.DefaultAccessConfig(),
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

// Engine composes origin walking access, transit routing, walking egress,
// travel-time rasterization, and marching squares. Callers can provide
// OriginStops explicitly for advanced use; otherwise nearby stops are seeded
// automatically from the query origin.
type Engine struct {
	graph   *graph.Graph
	router  *routing.Router
	surface SurfaceBuilder
	access  OriginAccessProvider
	contour contour.Config
}

type OriginAccessProvider interface {
	Seeds(origin spatial.Point, maxTravelSeconds int) ([]routing.StopSeed, error)
}

func New(artifact *graph.Artifact, config Config) (*Engine, error) {
	if artifact == nil {
		return nil, fmt.Errorf("graph artifact is nil")
	}
	access := radialOriginAccess{
		index:   spatial.NewStopIndex(artifact.Graph.Stops),
		access:  config.Access,
		walking: config.Walking,
	}
	return newEngine(artifact, radialWalkingSurface{config: config.Walking}, access, config.Contour)
}

func NewWithSurface(artifact *graph.Artifact, surface SurfaceBuilder, contourConfig contour.Config) (*Engine, error) {
	return newEngine(artifact, surface, nil, contourConfig)
}

func newEngine(
	artifact *graph.Artifact,
	surface SurfaceBuilder,
	access OriginAccessProvider,
	contourConfig contour.Config,
) (*Engine, error) {
	if surface == nil {
		return nil, fmt.Errorf("walking surface builder is nil")
	}
	router, err := routing.New(artifact)
	if err != nil {
		return nil, err
	}
	return &Engine{graph: &artifact.Graph, router: router, surface: surface, access: access, contour: contourConfig}, nil
}

func (e *Engine) Generate(ctx context.Context, query Query) (*Result, error) {
	if e == nil || e.router == nil || e.graph == nil {
		return nil, fmt.Errorf("isochrone engine is not initialized")
	}
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}
	if query.Departure.IsZero() {
		return nil, fmt.Errorf("departure time is required")
	}
	if query.MaxTravelSeconds <= 0 || query.MaxTravelSeconds > routing.MaxQueryTravelSeconds {
		return nil, fmt.Errorf("max travel time must be between 1 and %d seconds", routing.MaxQueryTravelSeconds)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	originStops := query.OriginStops
	if len(originStops) == 0 && e.access != nil {
		var err error
		originStops, err = e.access.Seeds(query.Origin, query.MaxTravelSeconds)
		if err != nil {
			return nil, fmt.Errorf("find transit access stops: %w", err)
		}
	}
	routingResult := emptyRoutingResult(len(e.graph.Stops))
	if len(originStops) > 0 {
		var err error
		routingResult, err = e.router.Search(ctx, routing.Query{
			Departure:        query.Departure,
			MaxTravelSeconds: query.MaxTravelSeconds,
			Origins:          originStops,
		})
		if err != nil {
			return nil, fmt.Errorf("route transit graph: %w", err)
		}
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

func emptyRoutingResult(stopCount int) *routing.Result {
	result := &routing.Result{StopTravelSeconds: make([]int32, stopCount)}
	for index := range result.StopTravelSeconds {
		result.StopTravelSeconds[index] = routing.UnreachableSeconds
	}
	return result
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

type radialOriginAccess struct {
	index   *spatial.StopIndex
	access  spatial.AccessConfig
	walking spatial.WalkingConfig
}

func (a radialOriginAccess) Seeds(origin spatial.Point, maxTravelSeconds int) ([]routing.StopSeed, error) {
	candidates, err := a.index.Nearby(origin, a.access)
	if err != nil {
		return nil, err
	}
	if a.walking.SpeedMetersPerSecond <= 0 || a.walking.NetworkFactor < 1 {
		return nil, fmt.Errorf("walking configuration is invalid")
	}
	seeds := make([]routing.StopSeed, 0, len(candidates))
	for _, candidate := range candidates {
		seconds := int(math.Ceil(candidate.DistanceMeters * a.walking.NetworkFactor / a.walking.SpeedMetersPerSecond))
		if seconds <= maxTravelSeconds {
			seeds = append(seeds, routing.StopSeed{Stop: candidate.Stop, AccessSeconds: seconds})
		}
	}
	return seeds, nil
}
