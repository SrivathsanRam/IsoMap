package graph

const (
	FormatName    = "isomap-gtfs-graph"
	SchemaVersion = 1
	NoIndex       = ^uint32(0)
)

type NodeKind uint8

const (
	NodeStop NodeKind = iota
	NodeArrival
	NodeDeparture
)

type EdgeKind uint8

const (
	EdgeBoard EdgeKind = iota
	EdgeRide
	EdgeDwell
	EdgeAlight
	EdgeTransfer
)

// Source identifies the exact raw feed used to create an artifact.
type Source struct {
	Filename string
	SHA256   string
}

type Manifest struct {
	Format        string `json:"format"`
	SchemaVersion int    `json:"schema_version"`
	FeedVersion   string `json:"feed_version,omitempty"`
	SourceFile    string `json:"source_file"`
	SourceSHA256  string `json:"source_sha256"`
	StopCount     int    `json:"stop_count"`
	RouteCount    int    `json:"route_count"`
	ServiceCount  int    `json:"service_count"`
	TripCount     int    `json:"trip_count"`
	NodeCount     int    `json:"node_count"`
	EdgeCount     int    `json:"edge_count"`
	Bounds        Bounds `json:"bounds"`
}

type Bounds struct {
	MinLat float64 `json:"min_lat"`
	MinLon float64 `json:"min_lon"`
	MaxLat float64 `json:"max_lat"`
	MaxLon float64 `json:"max_lon"`
}

type Artifact struct {
	Manifest Manifest
	Graph    Graph
}

// Graph uses a compressed sparse row adjacency layout. Edges originating at
// node n are Edges[FirstOut[n]:FirstOut[n+1]].
type Graph struct {
	Stops    []Stop
	Routes   []Route
	Services []Service
	Trips    []TripSchedule
	Nodes    []Node
	FirstOut []uint32
	Edges    []Edge
}

type Stop struct {
	ID            string
	Name          string
	Lat           float64
	Lon           float64
	LocationType  uint8
	ParentStation uint32
}

type Route struct {
	ID        string
	ShortName string
	LongName  string
	Type      int16
}

// Days uses bits Monday through Sunday in positions zero through six.
type Service struct {
	ID        string
	Days      uint8
	StartDate int32
	EndDate   int32
}

type TripSchedule struct {
	ID                        string
	Route                     uint32
	Service                   uint32
	Direction                 int8
	ReferenceDepartureSeconds int32
	Frequencies               []FrequencyWindow
}

type FrequencyWindow struct {
	StartSeconds   int32
	EndSeconds     int32
	HeadwaySeconds int32
	ExactTimes     bool
}

type Node struct {
	Kind         NodeKind
	Stop         uint32
	Trip         uint32
	StopSequence int32
}

// Board edges are time dependent. For a scheduled trip,
// ScheduledDepartureSeconds is the departure time. For a frequency trip,
// DepartureOffsetSeconds shifts each frequency window from the trip's first
// stop to this boarding stop.
type Edge struct {
	To                        uint32
	Kind                      EdgeKind
	DurationSeconds           uint32
	Trip                      uint32
	DepartureOffsetSeconds    int32
	ScheduledDepartureSeconds int32
}
