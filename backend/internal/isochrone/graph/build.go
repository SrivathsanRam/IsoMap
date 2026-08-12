package graph

import (
	"fmt"
	"math"
	"sort"

	gtfsdata "github.com/SrivathsanRam/IsoMap/internal/isochrone/gtfs"
)

type rawEdge struct {
	from uint32
	edge Edge
}

// Build converts a validated GTFS feed into an expanded graph with separate
// arrival and departure nodes. That split preserves dwell time while allowing
// boarding edges to remain time dependent.
func Build(feed *gtfsdata.Feed, source Source) (*Artifact, error) {
	if feed == nil {
		return nil, fmt.Errorf("GTFS feed is nil")
	}
	if err := feed.Validate(); err != nil {
		return nil, err
	}

	estimatedNodes := uint64(len(feed.Stops)) + 2*uint64(len(feed.StopTimes))
	if estimatedNodes > math.MaxUint32 {
		return nil, fmt.Errorf("graph has too many nodes: %d", estimatedNodes)
	}

	result := Graph{
		Agencies: make([]Agency, 0, len(feed.Agencies)),
		Stops:    make([]Stop, 0, len(feed.Stops)),
		Routes:   make([]Route, 0, len(feed.Routes)),
		Services: make([]Service, 0, len(feed.Services)),
		Trips:    make([]TripSchedule, 0, len(feed.Trips)),
		Nodes:    make([]Node, 0, estimatedNodes),
	}

	agencyIndex := make(map[string]uint32, len(feed.Agencies))
	for index, agency := range feed.Agencies {
		agencyIndex[agency.ID] = uint32(index)
		result.Agencies = append(result.Agencies, Agency{
			ID:       agency.ID,
			Name:     agency.Name,
			URL:      agency.URL,
			Timezone: agency.Timezone,
		})
	}

	stopIndex := make(map[string]uint32, len(feed.Stops))
	for index, stop := range feed.Stops {
		stopIndex[stop.ID] = uint32(index)
	}
	for _, stop := range feed.Stops {
		parent := NoIndex
		if stop.ParentStation != "" {
			parent = stopIndex[stop.ParentStation]
		}
		result.Stops = append(result.Stops, Stop{
			ID:            stop.ID,
			Name:          stop.Name,
			Lat:           stop.Lat,
			Lon:           stop.Lon,
			LocationType:  uint8(stop.LocationType),
			ParentStation: parent,
		})
		result.Nodes = append(result.Nodes, Node{
			Kind: NodeStop,
			Stop: stopIndex[stop.ID],
			Trip: NoIndex,
		})
	}

	routeIndex := make(map[string]uint32, len(feed.Routes))
	for index, route := range feed.Routes {
		if route.Type > math.MaxInt16 {
			return nil, fmt.Errorf("route %q has route_type outside the graph format range", route.ID)
		}
		routeIndex[route.ID] = uint32(index)
		agency := uint32(0)
		if route.AgencyID != "" {
			agency = agencyIndex[route.AgencyID]
		}
		result.Routes = append(result.Routes, Route{
			ID:        route.ID,
			Agency:    agency,
			ShortName: route.ShortName,
			LongName:  route.LongName,
			Type:      int16(route.Type),
		})
	}

	serviceIndex := make(map[string]uint32, len(feed.Services))
	for index, service := range feed.Services {
		serviceIndex[service.ID] = uint32(index)
		result.Services = append(result.Services, Service{
			ID:        service.ID,
			Days:      encodeDays(service.Days),
			StartDate: int32(service.StartDate),
			EndDate:   int32(service.EndDate),
		})
	}

	stopTimesByTrip := make(map[string][]gtfsdata.StopTime, len(feed.Trips))
	for _, stopTime := range feed.StopTimes {
		stopTimesByTrip[stopTime.TripID] = append(stopTimesByTrip[stopTime.TripID], stopTime)
	}
	frequenciesByTrip := make(map[string][]gtfsdata.Frequency, len(feed.Trips))
	for _, frequency := range feed.Frequencies {
		frequenciesByTrip[frequency.TripID] = append(frequenciesByTrip[frequency.TripID], frequency)
	}

	rawEdges := make([]rawEdge, 0, len(feed.StopTimes)*4+len(feed.Transfers))
	for tripIndex, trip := range feed.Trips {
		stopTimes := append([]gtfsdata.StopTime(nil), stopTimesByTrip[trip.ID]...)
		sort.Slice(stopTimes, func(i, j int) bool {
			return stopTimes[i].StopSequence < stopTimes[j].StopSequence
		})
		frequencies := append([]gtfsdata.Frequency(nil), frequenciesByTrip[trip.ID]...)
		sort.Slice(frequencies, func(i, j int) bool {
			return frequencies[i].StartSeconds < frequencies[j].StartSeconds
		})

		referenceDeparture := stopTimes[0].DepartureSeconds
		schedule := TripSchedule{
			ID:                        trip.ID,
			Route:                     routeIndex[trip.RouteID],
			Service:                   serviceIndex[trip.ServiceID],
			Direction:                 int8(trip.DirectionID),
			ReferenceDepartureSeconds: int32(referenceDeparture),
			Frequencies:               make([]FrequencyWindow, 0, len(frequencies)),
		}
		for _, frequency := range frequencies {
			schedule.Frequencies = append(schedule.Frequencies, FrequencyWindow{
				StartSeconds:   int32(frequency.StartSeconds),
				EndSeconds:     int32(frequency.EndSeconds),
				HeadwaySeconds: int32(frequency.HeadwaySeconds),
				ExactTimes:     frequency.ExactTimes == 1,
			})
		}
		result.Trips = append(result.Trips, schedule)

		arrivalNodes := make([]uint32, len(stopTimes))
		departureNodes := make([]uint32, len(stopTimes))
		for index, stopTime := range stopTimes {
			physicalStop := stopIndex[stopTime.StopID]
			arrivalNode := uint32(len(result.Nodes))
			result.Nodes = append(result.Nodes, Node{
				Kind:         NodeArrival,
				Stop:         physicalStop,
				Trip:         uint32(tripIndex),
				StopSequence: int32(stopTime.StopSequence),
			})
			departureNode := uint32(len(result.Nodes))
			result.Nodes = append(result.Nodes, Node{
				Kind:         NodeDeparture,
				Stop:         physicalStop,
				Trip:         uint32(tripIndex),
				StopSequence: int32(stopTime.StopSequence),
			})
			arrivalNodes[index] = arrivalNode
			departureNodes[index] = departureNode

			rawEdges = append(rawEdges,
				staticEdge(arrivalNode, physicalStop, EdgeAlight, 0),
				staticEdge(arrivalNode, departureNode, EdgeDwell, stopTime.DepartureSeconds-stopTime.ArrivalSeconds),
			)
			if index < len(stopTimes)-1 {
				rawEdges = append(rawEdges, rawEdge{
					from: physicalStop,
					edge: Edge{
						To:                        departureNode,
						Kind:                      EdgeBoard,
						Trip:                      uint32(tripIndex),
						DepartureOffsetSeconds:    int32(stopTime.DepartureSeconds - referenceDeparture),
						ScheduledDepartureSeconds: int32(stopTime.DepartureSeconds),
					},
				})
			}
		}
		for index := 0; index < len(stopTimes)-1; index++ {
			duration := stopTimes[index+1].ArrivalSeconds - stopTimes[index].DepartureSeconds
			rawEdges = append(rawEdges, staticEdge(departureNodes[index], arrivalNodes[index+1], EdgeRide, duration))
		}
	}

	for _, transfer := range feed.Transfers {
		if transfer.Type == 3 {
			continue
		}
		rawEdges = append(rawEdges, staticEdge(
			stopIndex[transfer.FromStopID],
			stopIndex[transfer.ToStopID],
			EdgeTransfer,
			transfer.MinTransferSeconds,
		))
	}

	sort.SliceStable(rawEdges, func(i, j int) bool {
		if rawEdges[i].from != rawEdges[j].from {
			return rawEdges[i].from < rawEdges[j].from
		}
		if rawEdges[i].edge.Kind != rawEdges[j].edge.Kind {
			return rawEdges[i].edge.Kind < rawEdges[j].edge.Kind
		}
		if rawEdges[i].edge.To != rawEdges[j].edge.To {
			return rawEdges[i].edge.To < rawEdges[j].edge.To
		}
		return rawEdges[i].edge.Trip < rawEdges[j].edge.Trip
	})
	if uint64(len(rawEdges)) > math.MaxUint32 {
		return nil, fmt.Errorf("graph has too many edges: %d", len(rawEdges))
	}

	result.FirstOut = make([]uint32, len(result.Nodes)+1)
	result.Edges = make([]Edge, len(rawEdges))
	for index, item := range rawEdges {
		result.FirstOut[item.from+1]++
		result.Edges[index] = item.edge
	}
	for index := 1; index < len(result.FirstOut); index++ {
		result.FirstOut[index] += result.FirstOut[index-1]
	}

	artifact := &Artifact{
		Manifest: Manifest{
			Format:        FormatName,
			SchemaVersion: SchemaVersion,
			FeedVersion:   feed.Info.Version,
			Timezone:      result.Agencies[0].Timezone,
			SourceFile:    source.Filename,
			SourceSHA256:  source.SHA256,
			AgencyCount:   len(result.Agencies),
			StopCount:     len(result.Stops),
			RouteCount:    len(result.Routes),
			ServiceCount:  len(result.Services),
			TripCount:     len(result.Trips),
			NodeCount:     len(result.Nodes),
			EdgeCount:     len(result.Edges),
			Bounds:        bounds(result.Stops),
		},
		Graph: result,
	}
	return artifact, nil
}

func staticEdge(from, to uint32, kind EdgeKind, duration int) rawEdge {
	return rawEdge{
		from: from,
		edge: Edge{
			To:                        to,
			Kind:                      kind,
			DurationSeconds:           uint32(duration),
			Trip:                      NoIndex,
			DepartureOffsetSeconds:    -1,
			ScheduledDepartureSeconds: -1,
		},
	}
}

func encodeDays(days [7]bool) uint8 {
	var encoded uint8
	for index, active := range days {
		if active {
			encoded |= 1 << index
		}
	}
	return encoded
}

func bounds(stops []Stop) Bounds {
	result := Bounds{
		MinLat: math.Inf(1),
		MinLon: math.Inf(1),
		MaxLat: math.Inf(-1),
		MaxLon: math.Inf(-1),
	}
	for _, stop := range stops {
		result.MinLat = math.Min(result.MinLat, stop.Lat)
		result.MinLon = math.Min(result.MinLon, stop.Lon)
		result.MaxLat = math.Max(result.MaxLat, stop.Lat)
		result.MaxLon = math.Max(result.MaxLon, stop.Lon)
	}
	return result
}
