package routing

import (
	"container/heap"
	"context"
	"fmt"
	"math"
	"time"

	"github.com/SrivathsanRam/IsoMap/internal/isochrone/graph"
)

const (
	UnreachableSeconds    int32 = -1
	MaxQueryTravelSeconds       = 7 * 24 * 60 * 60
	maxServiceTimeSeconds       = 7 * 24 * 60 * 60
)

// StopSeed connects an origin to a graph stop. AccessSeconds is normally the
// walking time supplied by the spatial access stage.
type StopSeed struct {
	Stop          uint32
	AccessSeconds int
}

type Query struct {
	Departure        time.Time
	MaxTravelSeconds int
	Origins          []StopSeed
}

type Result struct {
	// StopTravelSeconds is indexed exactly like graph.Graph.Stops. Unreachable
	// stops contain UnreachableSeconds.
	StopTravelSeconds []int32
	ReachedStopCount  int
	SettledNodeCount  int
}

type Router struct {
	graph           *graph.Graph
	location        *time.Location
	stopNodes       []uint32
	serviceLookback int
}

// New validates the graph invariants required by the query algorithm and
// prepares its stop-node lookup. The graph remains immutable and a Router can
// safely serve concurrent queries.
func New(artifact *graph.Artifact) (*Router, error) {
	if artifact == nil {
		return nil, fmt.Errorf("graph artifact is nil")
	}
	if artifact.Manifest.Format != graph.FormatName || artifact.Manifest.SchemaVersion != graph.SchemaVersion {
		return nil, fmt.Errorf("unsupported graph artifact format or schema")
	}
	location, err := time.LoadLocation(artifact.Manifest.Timezone)
	if err != nil {
		return nil, fmt.Errorf("load graph timezone %q: %w", artifact.Manifest.Timezone, err)
	}

	data := &artifact.Graph
	if len(data.FirstOut) != len(data.Nodes)+1 {
		return nil, fmt.Errorf("graph adjacency index has invalid length")
	}
	if len(data.FirstOut) > 0 && data.FirstOut[len(data.FirstOut)-1] != uint32(len(data.Edges)) {
		return nil, fmt.Errorf("graph adjacency index does not cover all edges")
	}
	for index := 1; index < len(data.FirstOut); index++ {
		if data.FirstOut[index] < data.FirstOut[index-1] {
			return nil, fmt.Errorf("graph adjacency index is not monotonic")
		}
	}

	stopNodes := make([]uint32, len(data.Stops))
	for index := range stopNodes {
		stopNodes[index] = graph.NoIndex
	}
	for index, node := range data.Nodes {
		if int(node.Stop) >= len(data.Stops) {
			return nil, fmt.Errorf("node %d refers to unknown stop %d", index, node.Stop)
		}
		switch node.Kind {
		case graph.NodeStop:
			if stopNodes[node.Stop] != graph.NoIndex {
				return nil, fmt.Errorf("stop %d has multiple stop nodes", node.Stop)
			}
			stopNodes[node.Stop] = uint32(index)
		case graph.NodeArrival, graph.NodeDeparture:
			if int(node.Trip) >= len(data.Trips) {
				return nil, fmt.Errorf("node %d refers to unknown trip %d", index, node.Trip)
			}
		default:
			return nil, fmt.Errorf("node %d has unknown kind %d", index, node.Kind)
		}
	}
	for stop, node := range stopNodes {
		if node == graph.NoIndex {
			return nil, fmt.Errorf("stop %d has no stop node", stop)
		}
	}

	for index, trip := range data.Trips {
		if int(trip.Route) >= len(data.Routes) {
			return nil, fmt.Errorf("trip %d refers to unknown route %d", index, trip.Route)
		}
		if int(trip.Service) >= len(data.Services) {
			return nil, fmt.Errorf("trip %d refers to unknown service %d", index, trip.Service)
		}
		for _, frequency := range trip.Frequencies {
			if frequency.HeadwaySeconds <= 0 || frequency.StartSeconds >= frequency.EndSeconds {
				return nil, fmt.Errorf("trip %d has an invalid frequency window", index)
			}
		}
	}

	maximumServiceTime := int64(0)
	for index, edge := range data.Edges {
		if int(edge.To) >= len(data.Nodes) {
			return nil, fmt.Errorf("edge %d points to unknown node %d", index, edge.To)
		}
		if edge.Kind < graph.EdgeBoard || edge.Kind > graph.EdgeTransfer {
			return nil, fmt.Errorf("edge %d has unknown kind %d", index, edge.Kind)
		}
		if edge.Kind != graph.EdgeBoard {
			continue
		}
		if int(edge.Trip) >= len(data.Trips) {
			return nil, fmt.Errorf("boarding edge %d refers to unknown trip %d", index, edge.Trip)
		}
		if edge.DepartureOffsetSeconds < 0 || edge.ScheduledDepartureSeconds < 0 {
			return nil, fmt.Errorf("boarding edge %d has invalid departure time data", index)
		}
		trip := data.Trips[edge.Trip]
		if len(trip.Frequencies) == 0 {
			maximumServiceTime = max64(maximumServiceTime, int64(edge.ScheduledDepartureSeconds))
			continue
		}
		for _, frequency := range trip.Frequencies {
			end := int64(frequency.EndSeconds) + int64(edge.DepartureOffsetSeconds)
			maximumServiceTime = max64(maximumServiceTime, end)
		}
	}
	if maximumServiceTime > maxServiceTimeSeconds {
		return nil, fmt.Errorf("graph contains service times beyond the supported seven-day window")
	}

	return &Router{
		graph:           data,
		location:        location,
		stopNodes:       stopNodes,
		serviceLookback: int(maximumServiceTime / (24 * 60 * 60)),
	}, nil
}

// Search runs bounded, time-dependent Dijkstra. Frequency-based boarding uses
// an expected wait of headway_secs / 2; the exact_times field is deliberately
// ignored by this approximation.
func (r *Router) Search(ctx context.Context, query Query) (*Result, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}
	if query.Departure.IsZero() {
		return nil, fmt.Errorf("departure time is required")
	}
	if query.MaxTravelSeconds <= 0 || query.MaxTravelSeconds > MaxQueryTravelSeconds {
		return nil, fmt.Errorf("max travel time must be between 1 and %d seconds", MaxQueryTravelSeconds)
	}
	if len(query.Origins) == 0 {
		return nil, fmt.Errorf("at least one origin stop is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	departureUnix := query.Departure.Unix()
	if departureUnix > math.MaxInt64-int64(query.MaxTravelSeconds) {
		return nil, fmt.Errorf("departure time is outside the supported range")
	}
	deadlineUnix := departureUnix + int64(query.MaxTravelSeconds)
	distance := make([]int64, len(r.graph.Nodes))
	for index := range distance {
		distance[index] = math.MaxInt64
	}
	queue := priorityQueue{}
	heap.Init(&queue)
	for _, origin := range query.Origins {
		if int(origin.Stop) >= len(r.stopNodes) {
			return nil, fmt.Errorf("origin refers to unknown stop %d", origin.Stop)
		}
		if origin.AccessSeconds < 0 {
			return nil, fmt.Errorf("origin access time must not be negative")
		}
		if origin.AccessSeconds > query.MaxTravelSeconds {
			continue
		}
		node := r.stopNodes[origin.Stop]
		arrival := departureUnix + int64(origin.AccessSeconds)
		if arrival < distance[node] {
			distance[node] = arrival
			heap.Push(&queue, queueItem{node: node, arrival: arrival})
		}
	}

	result := &Result{StopTravelSeconds: make([]int32, len(r.graph.Stops))}
	for index := range result.StopTravelSeconds {
		result.StopTravelSeconds[index] = UnreachableSeconds
	}

	for queue.Len() > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		current := heap.Pop(&queue).(queueItem)
		if current.arrival != distance[current.node] {
			continue
		}
		if current.arrival > deadlineUnix {
			break
		}
		result.SettledNodeCount++

		node := r.graph.Nodes[current.node]
		if node.Kind == graph.NodeStop && result.StopTravelSeconds[node.Stop] == UnreachableSeconds {
			result.StopTravelSeconds[node.Stop] = int32(current.arrival - departureUnix)
			result.ReachedStopCount++
		}

		start := r.graph.FirstOut[current.node]
		end := r.graph.FirstOut[current.node+1]
		for edgeIndex := start; edgeIndex < end; edgeIndex++ {
			edge := r.graph.Edges[edgeIndex]
			var arrival int64
			if edge.Kind == graph.EdgeBoard {
				var available bool
				arrival, available = r.nextBoarding(edge, current.arrival, deadlineUnix)
				if !available {
					continue
				}
			} else {
				duration := int64(edge.DurationSeconds)
				if duration > deadlineUnix-current.arrival {
					continue
				}
				arrival = current.arrival + duration
			}
			if arrival > deadlineUnix || arrival >= distance[edge.To] {
				continue
			}
			distance[edge.To] = arrival
			heap.Push(&queue, queueItem{node: edge.To, arrival: arrival})
		}
	}

	return result, nil
}

func (r *Router) nextBoarding(edge graph.Edge, readyUnix, deadlineUnix int64) (int64, bool) {
	trip := r.graph.Trips[edge.Trip]
	service := r.graph.Services[trip.Service]
	readyLocal := time.Unix(readyUnix, 0).In(r.location)
	localMidnight := time.Date(readyLocal.Year(), readyLocal.Month(), readyLocal.Day(), 0, 0, 0, 0, r.location)
	remainingDays := int((deadlineUnix-readyUnix)/(24*60*60)) + 1
	best := int64(math.MaxInt64)

	for dayOffset := -r.serviceLookback; dayOffset <= remainingDays; dayOffset++ {
		serviceDay := localMidnight.AddDate(0, 0, dayOffset)
		if !serviceRuns(service, serviceDay) {
			continue
		}
		if len(trip.Frequencies) == 0 {
			candidate := serviceTime(serviceDay, int64(edge.ScheduledDepartureSeconds), r.location).Unix()
			if candidate >= readyUnix && candidate <= deadlineUnix && candidate < best {
				best = candidate
			}
			continue
		}

		for _, frequency := range trip.Frequencies {
			windowStart := serviceTime(
				serviceDay,
				int64(frequency.StartSeconds)+int64(edge.DepartureOffsetSeconds),
				r.location,
			).Unix()
			windowEnd := serviceTime(
				serviceDay,
				int64(frequency.EndSeconds)+int64(edge.DepartureOffsetSeconds),
				r.location,
			).Unix()
			ready := max64(readyUnix, windowStart)
			candidate := ready + int64(frequency.HeadwaySeconds)/2
			if candidate < windowEnd && candidate <= deadlineUnix && candidate < best {
				best = candidate
			}
		}
	}

	return best, best != math.MaxInt64
}

func serviceRuns(service graph.Service, serviceDay time.Time) bool {
	date := int32(serviceDay.Year()*10000 + int(serviceDay.Month())*100 + serviceDay.Day())
	if date < service.StartDate || date > service.EndDate {
		return false
	}
	mondayBasedWeekday := (int(serviceDay.Weekday()) + 6) % 7
	return service.Days&(1<<mondayBasedWeekday) != 0
}

func serviceTime(serviceDay time.Time, seconds int64, location *time.Location) time.Time {
	days := seconds / (24 * 60 * 60)
	remainder := seconds % (24 * 60 * 60)
	hour := remainder / (60 * 60)
	minute := (remainder % (60 * 60)) / 60
	second := remainder % 60
	return time.Date(
		serviceDay.Year(),
		serviceDay.Month(),
		serviceDay.Day()+int(days),
		int(hour),
		int(minute),
		int(second),
		0,
		location,
	)
}

type queueItem struct {
	node    uint32
	arrival int64
}

type priorityQueue []queueItem

func (q priorityQueue) Len() int { return len(q) }

func (q priorityQueue) Less(i, j int) bool {
	if q[i].arrival != q[j].arrival {
		return q[i].arrival < q[j].arrival
	}
	return q[i].node < q[j].node
}

func (q priorityQueue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }

func (q *priorityQueue) Push(value any) {
	*q = append(*q, value.(queueItem))
}

func (q *priorityQueue) Pop() any {
	old := *q
	last := len(old) - 1
	item := old[last]
	*q = old[:last]
	return item
}

func max64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}
