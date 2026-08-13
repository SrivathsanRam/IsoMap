// Package footpath applies the generated LTA footpath raster at query time.
// It provides both origin-to-stop access and a weighted multi-source walking
// surface after transit, so the same pedestrian cost model is used on both
// sides of a journey.
package footpath

import (
	"container/heap"
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/SrivathsanRam/IsoMap/internal/isochrone/graph"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/routing"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/spatial"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/walking"
)

const (
	ModelName                            = "lta-footpath-raster"
	DefaultMaximumSnapDistanceMeters     = 100.0
	DefaultMaximumTerminalDistanceMeters = 75.0
	DefaultSurfaceCellSizeMeters         = 50.0
	DefaultMaximumSurfaceGridCells       = 750_000
	DefaultMinimumLargestComponentShare  = 0.80
)

// Config controls the small off-network connectors and output surface. The
// main walking cost is always the shortest path through walkable raster cells.
type Config struct {
	SpeedMetersPerSecond          float64
	MaximumSnapDistanceMeters     float64
	MaximumTerminalDistanceMeters float64
	SurfaceCellSizeMeters         float64
	MaximumSurfaceGridCells       int
	MinimumLargestComponentShare  float64
	RadialFallback                spatial.WalkingConfig
}

func DefaultConfig(fallback spatial.WalkingConfig) Config {
	return Config{
		SpeedMetersPerSecond:          fallback.SpeedMetersPerSecond,
		MaximumSnapDistanceMeters:     DefaultMaximumSnapDistanceMeters,
		MaximumTerminalDistanceMeters: DefaultMaximumTerminalDistanceMeters,
		SurfaceCellSizeMeters:         DefaultSurfaceCellSizeMeters,
		MaximumSurfaceGridCells:       DefaultMaximumSurfaceGridCells,
		MinimumLargestComponentShare:  DefaultMinimumLargestComponentShare,
		RadialFallback:                fallback,
	}
}

type Runtime struct {
	network       *walking.Network
	stops         []graph.Stop
	stopLinks     []networkLink
	access        spatial.AccessConfig
	config        Config
	fallbackIndex *spatial.StopIndex
}

type networkLink struct {
	cell           int
	distanceMeters float64
	available      bool
}

type timedPoint struct {
	point         spatial.LocalPoint
	travelSeconds float64
}

type cellSeed struct {
	cell    int
	seconds int32
}

func NewRuntime(
	artifact *walking.Artifact,
	stops []graph.Stop,
	access spatial.AccessConfig,
	config Config,
) (*Runtime, error) {
	if artifact == nil {
		return nil, fmt.Errorf("footpath artifact is nil")
	}
	if artifact.Manifest.Format != walking.FormatName || artifact.Manifest.SchemaVersion != walking.SchemaVersion {
		return nil, fmt.Errorf("unsupported footpath artifact format or schema")
	}
	cellCount := artifact.Network.CellCount()
	if cellCount == 0 || artifact.Manifest.WalkableCellCount == 0 {
		return nil, fmt.Errorf("footpath network is empty")
	}
	if len(artifact.Network.WalkableBits) != (cellCount+63)/64 {
		return nil, fmt.Errorf("footpath network bitset does not match its dimensions")
	}
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	componentShare := float64(artifact.Manifest.LargestComponentCells) / float64(artifact.Manifest.WalkableCellCount)
	if componentShare < config.MinimumLargestComponentShare {
		return nil, fmt.Errorf(
			"footpath network largest component covers %.2f%% of walkable cells; require at least %.2f%%",
			100*componentShare,
			100*config.MinimumLargestComponentShare,
		)
	}
	if access.MaximumDistanceMeters <= 0 || access.MaximumStops <= 0 {
		return nil, fmt.Errorf("footpath access configuration is invalid")
	}

	runtime := &Runtime{
		network:       &artifact.Network,
		stops:         stops,
		stopLinks:     make([]networkLink, len(stops)),
		access:        access,
		config:        config,
		fallbackIndex: spatial.NewStopIndex(stops),
	}
	for index, stop := range stops {
		if stop.LocationType != 0 {
			continue
		}
		link, err := runtime.link(spatial.Point{Lat: stop.Lat, Lon: stop.Lon})
		if err != nil {
			return nil, fmt.Errorf("project stop %q into footpath network: %w", stop.ID, err)
		}
		runtime.stopLinks[index] = link
	}
	return runtime, nil
}

// Seeds finds transit boarding stops by walking the footpath network from the
// query origin. A radial fallback is used only when the origin cannot be
// snapped to the supplied network coverage.
func (runtime *Runtime) Seeds(
	ctx context.Context,
	origin spatial.Point,
	maxTravelSeconds int,
) ([]routing.StopSeed, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}
	if maxTravelSeconds <= 0 {
		return nil, fmt.Errorf("maximum travel time must be positive")
	}
	originLink, err := runtime.link(origin)
	if err != nil {
		return nil, err
	}
	if !originLink.available {
		return runtime.radialSeeds(origin, maxTravelSeconds)
	}

	accessLimit := int(math.Ceil(runtime.access.MaximumDistanceMeters / runtime.config.SpeedMetersPerSecond))
	if accessLimit > maxTravelSeconds {
		accessLimit = maxTravelSeconds
	}
	originSeconds := walkingSeconds(originLink.distanceMeters, runtime.config.SpeedMetersPerSecond)
	distances, _, err := runtime.shortestPaths(ctx, []cellSeed{{cell: originLink.cell, seconds: int32(originSeconds)}}, accessLimit)
	if err != nil {
		return nil, err
	}

	type candidate struct {
		stop    uint32
		seconds int
	}
	candidates := make([]candidate, 0)
	for stopIndex, link := range runtime.stopLinks {
		if !link.available || distances[link.cell] < 0 {
			continue
		}
		seconds := int(distances[link.cell]) + walkingSeconds(link.distanceMeters, runtime.config.SpeedMetersPerSecond)
		if seconds <= accessLimit {
			candidates = append(candidates, candidate{stop: uint32(stopIndex), seconds: seconds})
		}
	}
	sort.Slice(candidates, func(left, right int) bool {
		if candidates[left].seconds != candidates[right].seconds {
			return candidates[left].seconds < candidates[right].seconds
		}
		return candidates[left].stop < candidates[right].stop
	})
	if len(candidates) > runtime.access.MaximumStops {
		candidates = candidates[:runtime.access.MaximumStops]
	}
	if len(candidates) == 0 {
		return runtime.radialSeeds(origin, maxTravelSeconds)
	}
	seeds := make([]routing.StopSeed, len(candidates))
	for index, candidate := range candidates {
		seeds[index] = routing.StopSeed{Stop: candidate.stop, AccessSeconds: candidate.seconds}
	}
	return seeds, nil
}

// Build performs a weighted multi-source walking search. The origin starts at
// zero and each reached transit stop starts at its transit arrival time. Only a
// short, capped straight connector is allowed between a point and the raster.
func (runtime *Runtime) Build(
	ctx context.Context,
	origin spatial.Point,
	stops []graph.Stop,
	stopTravelSeconds []int32,
	maxTravelSeconds int,
) (*spatial.Grid, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}
	if len(stops) != len(runtime.stops) || len(stopTravelSeconds) != len(stops) {
		return nil, fmt.Errorf("stop travel-time count does not match the footpath runtime")
	}
	if maxTravelSeconds <= 0 {
		return nil, fmt.Errorf("maximum travel time must be positive")
	}
	originLink, err := runtime.link(origin)
	if err != nil {
		return nil, err
	}
	if !originLink.available {
		fallback := runtime.config.RadialFallback
		fallback.CellSizeMeters = runtime.config.SurfaceCellSizeMeters
		fallback.MaximumGridCells = runtime.config.MaximumSurfaceGridCells
		return spatial.BuildTravelTimeGrid(ctx, origin, stops, stopTravelSeconds, maxTravelSeconds, fallback)
	}

	projection := spatial.NewProjection(origin)
	sources := []timedPoint{{point: projection.Project(origin)}}
	seeds := []cellSeed{{
		cell:    originLink.cell,
		seconds: int32(walkingSeconds(originLink.distanceMeters, runtime.config.SpeedMetersPerSecond)),
	}}
	for index, travelSeconds := range stopTravelSeconds {
		if travelSeconds < 0 || int(travelSeconds) > maxTravelSeconds {
			continue
		}
		stopPoint := spatial.Point{Lat: stops[index].Lat, Lon: stops[index].Lon}
		sources = append(sources, timedPoint{
			point:         projection.Project(stopPoint),
			travelSeconds: float64(travelSeconds),
		})
		link := runtime.stopLinks[index]
		if !link.available {
			continue
		}
		seedSeconds := int(travelSeconds) + walkingSeconds(link.distanceMeters, runtime.config.SpeedMetersPerSecond)
		if seedSeconds <= maxTravelSeconds {
			seeds = append(seeds, cellSeed{cell: link.cell, seconds: int32(seedSeconds)})
		}
	}

	distances, settled, err := runtime.shortestPaths(ctx, seeds, maxTravelSeconds)
	if err != nil {
		return nil, err
	}
	if len(settled) == 0 {
		return nil, fmt.Errorf("footpath search settled no cells")
	}

	localCells := make([]spatial.LocalPoint, len(settled))
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	terminalRadius := runtime.config.MaximumTerminalDistanceMeters
	for listIndex, cell := range settled {
		easting, northing, ok := runtime.network.CellCenter(cell)
		if !ok {
			return nil, fmt.Errorf("footpath search returned an invalid cell")
		}
		geographic, conversionErr := spatial.SVY21Unproject(spatial.LocalPoint{X: easting, Y: northing})
		if conversionErr != nil {
			return nil, conversionErr
		}
		local := projection.Project(geographic)
		localCells[listIndex] = local
		minX = math.Min(minX, local.X-terminalRadius)
		minY = math.Min(minY, local.Y-terminalRadius)
		maxX = math.Max(maxX, local.X+terminalRadius)
		maxY = math.Max(maxY, local.Y+terminalRadius)
	}
	for _, source := range sources {
		minX = math.Min(minX, source.point.X-terminalRadius)
		minY = math.Min(minY, source.point.Y-terminalRadius)
		maxX = math.Max(maxX, source.point.X+terminalRadius)
		maxY = math.Max(maxY, source.point.Y+terminalRadius)
	}

	cellSize := runtime.config.SurfaceCellSizeMeters
	padding := 2 * cellSize
	minX = math.Floor((minX-padding)/cellSize) * cellSize
	minY = math.Floor((minY-padding)/cellSize) * cellSize
	maxX = math.Ceil((maxX+padding)/cellSize) * cellSize
	maxY = math.Ceil((maxY+padding)/cellSize) * cellSize
	columns := int(math.Round((maxX-minX)/cellSize)) + 1
	rows := int(math.Round((maxY-minY)/cellSize)) + 1
	if columns < 2 || rows < 2 || columns > math.MaxInt/rows {
		return nil, fmt.Errorf("footpath travel-time grid dimensions are invalid")
	}
	cellCount := columns * rows
	if cellCount > runtime.config.MaximumSurfaceGridCells {
		return nil, fmt.Errorf(
			"footpath travel-time grid requires %d cells, exceeding the configured maximum of %d",
			cellCount,
			runtime.config.MaximumSurfaceGridCells,
		)
	}
	secondsPerMeter := 1 / runtime.config.SpeedMetersPerSecond
	outside := float64(maxTravelSeconds) + 2*math.Sqrt2*cellSize*secondsPerMeter
	values := make([]float64, cellCount)
	for index := range values {
		values[index] = outside
	}

	for index, cell := range settled {
		if index%1024 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		splat(values, columns, rows, minX, minY, cellSize, localCells[index], float64(distances[cell]), terminalRadius, secondsPerMeter)
	}
	for _, source := range sources {
		splat(values, columns, rows, minX, minY, cellSize, source.point, source.travelSeconds, terminalRadius, secondsPerMeter)
	}
	return spatial.NewGrid(origin, columns, rows, minX, minY, cellSize, float64(maxTravelSeconds), values)
}

func (runtime *Runtime) radialSeeds(origin spatial.Point, maxTravelSeconds int) ([]routing.StopSeed, error) {
	candidates, err := runtime.fallbackIndex.Nearby(origin, runtime.access)
	if err != nil {
		return nil, err
	}
	walkingConfig := runtime.config.RadialFallback
	seeds := make([]routing.StopSeed, 0, len(candidates))
	for _, candidate := range candidates {
		seconds := int(math.Ceil(candidate.DistanceMeters * walkingConfig.NetworkFactor / walkingConfig.SpeedMetersPerSecond))
		if seconds <= maxTravelSeconds {
			seeds = append(seeds, routing.StopSeed{Stop: candidate.Stop, AccessSeconds: seconds})
		}
	}
	return seeds, nil
}

func (runtime *Runtime) link(point spatial.Point) (networkLink, error) {
	projected, err := spatial.SVY21Project(point)
	if err != nil {
		return networkLink{}, err
	}
	cellSize := runtime.network.CellSizeMeters
	radius := int(math.Ceil(runtime.config.MaximumSnapDistanceMeters/cellSize)) + 1
	centerColumn := int(math.Floor((projected.X - runtime.network.MinEasting) / cellSize))
	centerRow := int(math.Floor((projected.Y - runtime.network.MinNorthing) / cellSize))
	best := networkLink{distanceMeters: math.Inf(1)}
	for row := maxInt(0, centerRow-radius); row <= minInt(runtime.network.Rows-1, centerRow+radius); row++ {
		for column := maxInt(0, centerColumn-radius); column <= minInt(runtime.network.Columns-1, centerColumn+radius); column++ {
			index, _ := runtime.network.Index(row, column)
			if !runtime.network.IsWalkable(index) {
				continue
			}
			easting, northing, _ := runtime.network.CellCenter(index)
			distance := math.Hypot(easting-projected.X, northing-projected.Y)
			if distance <= runtime.config.MaximumSnapDistanceMeters &&
				(distance < best.distanceMeters || (distance == best.distanceMeters && index < best.cell)) {
				best = networkLink{cell: index, distanceMeters: distance, available: true}
			}
		}
	}
	return best, nil
}

func (runtime *Runtime) shortestPaths(
	ctx context.Context,
	seeds []cellSeed,
	maxSeconds int,
) ([]int32, []int, error) {
	distances := make([]int32, runtime.network.CellCount())
	for index := range distances {
		distances[index] = -1
	}
	queue := cellQueue{}
	heap.Init(&queue)
	for _, seed := range seeds {
		if seed.cell < 0 || !runtime.network.IsWalkable(seed.cell) || seed.seconds < 0 || int(seed.seconds) > maxSeconds {
			continue
		}
		if distances[seed.cell] < 0 || seed.seconds < distances[seed.cell] {
			distances[seed.cell] = seed.seconds
			heap.Push(&queue, cellQueueItem{cell: seed.cell, seconds: seed.seconds})
		}
	}
	orthogonal := int32(walkingSeconds(runtime.network.CellSizeMeters, runtime.config.SpeedMetersPerSecond))
	diagonal := int32(walkingSeconds(runtime.network.CellSizeMeters*math.Sqrt2, runtime.config.SpeedMetersPerSecond))
	settled := make([]int, 0)
	for queue.Len() > 0 {
		if len(settled)%1024 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, nil, err
			}
		}
		current := heap.Pop(&queue).(cellQueueItem)
		if distances[current.cell] != current.seconds {
			continue
		}
		settled = append(settled, current.cell)
		row, column, _ := runtime.network.RowColumn(current.cell)
		for rowOffset := -1; rowOffset <= 1; rowOffset++ {
			for columnOffset := -1; columnOffset <= 1; columnOffset++ {
				if rowOffset == 0 && columnOffset == 0 {
					continue
				}
				neighbor, ok := runtime.network.Index(row+rowOffset, column+columnOffset)
				if !ok || !runtime.network.IsWalkable(neighbor) {
					continue
				}
				cost := orthogonal
				if rowOffset != 0 && columnOffset != 0 {
					cost = diagonal
				}
				candidate := current.seconds + cost
				if int(candidate) > maxSeconds || (distances[neighbor] >= 0 && candidate >= distances[neighbor]) {
					continue
				}
				distances[neighbor] = candidate
				heap.Push(&queue, cellQueueItem{cell: neighbor, seconds: candidate})
			}
		}
	}
	return distances, settled, nil
}

func splat(
	values []float64,
	columns int,
	rows int,
	minX float64,
	minY float64,
	cellSize float64,
	source spatial.LocalPoint,
	baseSeconds float64,
	radius float64,
	secondsPerMeter float64,
) {
	minimumColumn := clampInt(int(math.Floor((source.X-radius-minX)/cellSize)), 0, columns-1)
	maximumColumn := clampInt(int(math.Ceil((source.X+radius-minX)/cellSize)), 0, columns-1)
	minimumRow := clampInt(int(math.Floor((source.Y-radius-minY)/cellSize)), 0, rows-1)
	maximumRow := clampInt(int(math.Ceil((source.Y+radius-minY)/cellSize)), 0, rows-1)
	for row := minimumRow; row <= maximumRow; row++ {
		y := minY + float64(row)*cellSize
		for column := minimumColumn; column <= maximumColumn; column++ {
			x := minX + float64(column)*cellSize
			distance := math.Hypot(x-source.X, y-source.Y)
			if distance > radius {
				continue
			}
			candidate := baseSeconds + distance*secondsPerMeter
			index := row*columns + column
			if candidate < values[index] {
				values[index] = candidate
			}
		}
	}
}

func validateConfig(config Config) error {
	if !finitePositive(config.SpeedMetersPerSecond) {
		return fmt.Errorf("footpath walking speed must be a finite positive number")
	}
	if !finitePositive(config.MaximumSnapDistanceMeters) {
		return fmt.Errorf("footpath maximum snap distance must be a finite positive number")
	}
	if !finitePositive(config.MaximumTerminalDistanceMeters) {
		return fmt.Errorf("footpath maximum terminal distance must be a finite positive number")
	}
	if !finitePositive(config.SurfaceCellSizeMeters) {
		return fmt.Errorf("footpath surface cell size must be a finite positive number")
	}
	if config.MaximumSurfaceGridCells <= 0 {
		return fmt.Errorf("footpath maximum surface grid cells must be positive")
	}
	if config.MinimumLargestComponentShare <= 0 || config.MinimumLargestComponentShare > 1 ||
		math.IsNaN(config.MinimumLargestComponentShare) || math.IsInf(config.MinimumLargestComponentShare, 0) {
		return fmt.Errorf("footpath minimum largest-component share must be between zero and one")
	}
	if config.RadialFallback.SpeedMetersPerSecond <= 0 || config.RadialFallback.NetworkFactor < 1 {
		return fmt.Errorf("radial walking fallback configuration is invalid")
	}
	return nil
}

func walkingSeconds(distanceMeters, speedMetersPerSecond float64) int {
	// Projection round trips can leave sub-millimetre noise at an exact cell
	// center; it must not turn a zero-length connector into a whole second.
	if distanceMeters < 0.001 {
		return 0
	}
	return int(math.Ceil(distanceMeters / speedMetersPerSecond))
}

func finitePositive(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func clampInt(value, minimum, maximum int) int {
	return maxInt(minimum, minInt(value, maximum))
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

type cellQueueItem struct {
	cell    int
	seconds int32
}

type cellQueue []cellQueueItem

func (queue cellQueue) Len() int { return len(queue) }
func (queue cellQueue) Less(left, right int) bool {
	if queue[left].seconds != queue[right].seconds {
		return queue[left].seconds < queue[right].seconds
	}
	return queue[left].cell < queue[right].cell
}
func (queue cellQueue) Swap(left, right int) { queue[left], queue[right] = queue[right], queue[left] }
func (queue *cellQueue) Push(value any)      { *queue = append(*queue, value.(cellQueueItem)) }
func (queue *cellQueue) Pop() any {
	old := *queue
	last := len(old) - 1
	item := old[last]
	*queue = old[:last]
	return item
}
