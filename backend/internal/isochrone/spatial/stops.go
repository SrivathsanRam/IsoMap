package spatial

import (
	"fmt"
	"math"
	"sort"

	"github.com/SrivathsanRam/IsoMap/internal/isochrone/graph"
)

const (
	DefaultMaximumAccessDistanceMeters = 1000.0
	DefaultMaximumAccessStops          = 64
)

type AccessConfig struct {
	MaximumDistanceMeters float64
	MaximumStops          int
}

func DefaultAccessConfig() AccessConfig {
	return AccessConfig{
		MaximumDistanceMeters: DefaultMaximumAccessDistanceMeters,
		MaximumStops:          DefaultMaximumAccessStops,
	}
}

type StopAccess struct {
	Stop           uint32
	DistanceMeters float64
}

// StopIndex provides origin access to boarding stops. Singapore's feed is
// small enough that a linear scan is faster to load and plenty fast per query;
// the type leaves room for a tree/grid index without changing callers.
type StopIndex struct {
	stops []indexedStop
}

type indexedStop struct {
	index uint32
	point Point
}

func NewStopIndex(stops []graph.Stop) *StopIndex {
	indexed := make([]indexedStop, 0, len(stops))
	for index, stop := range stops {
		if stop.LocationType != 0 {
			continue
		}
		indexed = append(indexed, indexedStop{
			index: uint32(index),
			point: Point{Lat: stop.Lat, Lon: stop.Lon},
		})
	}
	return &StopIndex{stops: indexed}
}

func (index *StopIndex) Nearby(origin Point, config AccessConfig) ([]StopAccess, error) {
	if index == nil {
		return nil, fmt.Errorf("stop index is nil")
	}
	if !validPoint(origin) {
		return nil, fmt.Errorf("origin coordinate is invalid")
	}
	if config.MaximumDistanceMeters <= 0 || math.IsNaN(config.MaximumDistanceMeters) || math.IsInf(config.MaximumDistanceMeters, 0) {
		return nil, fmt.Errorf("maximum access distance must be a finite positive number")
	}
	if config.MaximumStops <= 0 {
		return nil, fmt.Errorf("maximum access stops must be positive")
	}

	projection := NewProjection(origin)
	candidates := make([]StopAccess, 0)
	for _, stop := range index.stops {
		local := projection.Project(stop.point)
		distance := math.Hypot(local.X, local.Y)
		if distance <= config.MaximumDistanceMeters {
			candidates = append(candidates, StopAccess{Stop: stop.index, DistanceMeters: distance})
		}
	}
	sort.Slice(candidates, func(left, right int) bool {
		if candidates[left].DistanceMeters != candidates[right].DistanceMeters {
			return candidates[left].DistanceMeters < candidates[right].DistanceMeters
		}
		return candidates[left].Stop < candidates[right].Stop
	})
	if len(candidates) > config.MaximumStops {
		candidates = candidates[:config.MaximumStops]
	}
	return candidates, nil
}
