package gtfs

// Feed is the subset of a static GTFS feed needed to build the transit graph.
// SourceLine fields are retained only to make validation errors actionable.
type Feed struct {
	Info        FeedInfo
	Agencies    []Agency
	Stops       []Stop
	Routes      []Route
	Services    []Service
	Trips       []Trip
	StopTimes   []StopTime
	Frequencies []Frequency
	Transfers   []Transfer
}

type Agency struct {
	ID         string
	Name       string
	URL        string
	Timezone   string
	SourceLine int
}

type FeedInfo struct {
	PublisherName string
	PublisherURL  string
	Language      string
	Version       string
}

type Stop struct {
	ID            string
	Name          string
	Lat           float64
	Lon           float64
	LocationType  int
	ParentStation string
	SourceLine    int
}

type Route struct {
	ID         string
	AgencyID   string
	ShortName  string
	LongName   string
	Type       int
	SourceLine int
}

type Service struct {
	ID         string
	Days       [7]bool
	StartDate  int
	EndDate    int
	SourceLine int
}

type Trip struct {
	ID          string
	RouteID     string
	ServiceID   string
	DirectionID int
	SourceLine  int
}

type StopTime struct {
	TripID           string
	StopID           string
	ArrivalSeconds   int
	DepartureSeconds int
	StopSequence     int
	SourceLine       int
}

type Frequency struct {
	TripID         string
	StartSeconds   int
	EndSeconds     int
	HeadwaySeconds int
	ExactTimes     int
	SourceLine     int
}

type Transfer struct {
	FromStopID         string
	ToStopID           string
	Type               int
	MinTransferSeconds int
	SourceLine         int
}
