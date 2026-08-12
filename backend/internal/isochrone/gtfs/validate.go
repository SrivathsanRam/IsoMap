package gtfs

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const maxReportedValidationIssues = 100

// ValidationError reports semantic and cross-table problems together so a
// feed author can fix more than one issue per build attempt.
type ValidationError struct {
	Issues []string
	Total  int
}

func (e *ValidationError) Error() string {
	message := fmt.Sprintf("GTFS validation failed with %d issue(s)", e.Total)
	if len(e.Issues) == 0 {
		return message
	}
	message += ":\n- " + strings.Join(e.Issues, "\n- ")
	if e.Total > len(e.Issues) {
		message += fmt.Sprintf("\n- ... and %d more", e.Total-len(e.Issues))
	}
	return message
}

type validator struct {
	issues []string
	total  int
}

func (v *validator) add(format string, args ...any) {
	v.total++
	if len(v.issues) < maxReportedValidationIssues {
		v.issues = append(v.issues, fmt.Sprintf(format, args...))
	}
}

func (v *validator) result() error {
	if v.total == 0 {
		return nil
	}
	return &ValidationError{Issues: v.issues, Total: v.total}
}

// Validate checks identifiers, references, time ordering, coordinates,
// calendars, frequency windows, and transfer definitions required by the
// transit graph builder.
func (f *Feed) Validate() error {
	v := &validator{}

	agencyByID := make(map[string]Agency, len(f.Agencies))
	feedTimezone := ""
	if len(f.Agencies) == 0 {
		v.add("agency.txt has no rows")
	}
	for _, agency := range f.Agencies {
		where := source("agency.txt", agency.SourceLine)
		if len(f.Agencies) > 1 && agency.ID == "" {
			v.add("%s: agency_id is required when the feed has multiple agencies", where)
		}
		if previous, exists := agencyByID[agency.ID]; exists {
			v.add("%s: duplicate agency_id %q (first seen on line %d)", where, agency.ID, previous.SourceLine)
		} else {
			agencyByID[agency.ID] = agency
		}
		if agency.Name == "" {
			v.add("%s: agency_name is empty", where)
		}
		if agency.URL == "" {
			v.add("%s: agency_url is empty", where)
		}
		if agency.Timezone == "" {
			v.add("%s: agency_timezone is empty", where)
		} else if _, err := time.LoadLocation(agency.Timezone); err != nil {
			v.add("%s: agency_timezone %q is invalid", where, agency.Timezone)
		}
		if feedTimezone == "" {
			feedTimezone = agency.Timezone
		} else if agency.Timezone != feedTimezone {
			v.add("%s: all agencies must use the same timezone", where)
		}
	}

	stopByID := make(map[string]Stop, len(f.Stops))
	if len(f.Stops) == 0 {
		v.add("stops.txt has no rows")
	}
	for _, stop := range f.Stops {
		where := source("stops.txt", stop.SourceLine)
		if stop.ID == "" {
			v.add("%s: stop_id is empty", where)
		} else if previous, exists := stopByID[stop.ID]; exists {
			v.add("%s: duplicate stop_id %q (first seen on line %d)", where, stop.ID, previous.SourceLine)
		} else {
			stopByID[stop.ID] = stop
		}
		if stop.Name == "" {
			v.add("%s: stop_name is empty", where)
		}
		if math.IsNaN(stop.Lat) || math.IsInf(stop.Lat, 0) || stop.Lat < -90 || stop.Lat > 90 {
			v.add("%s: stop_lat %v is outside [-90, 90]", where, stop.Lat)
		}
		if math.IsNaN(stop.Lon) || math.IsInf(stop.Lon, 0) || stop.Lon < -180 || stop.Lon > 180 {
			v.add("%s: stop_lon %v is outside [-180, 180]", where, stop.Lon)
		}
		if stop.LocationType < 0 || stop.LocationType > 4 {
			v.add("%s: unsupported location_type %d", where, stop.LocationType)
		}
	}
	for _, stop := range f.Stops {
		if stop.ParentStation == "" {
			continue
		}
		where := source("stops.txt", stop.SourceLine)
		parent, exists := stopByID[stop.ParentStation]
		if !exists {
			v.add("%s: parent_station %q does not exist", where, stop.ParentStation)
		} else if parent.ID == stop.ID {
			v.add("%s: stop cannot be its own parent_station", where)
		} else if parent.LocationType != 1 {
			v.add("%s: parent_station %q is not a station", where, stop.ParentStation)
		}
	}

	routeByID := make(map[string]Route, len(f.Routes))
	if len(f.Routes) == 0 {
		v.add("routes.txt has no rows")
	}
	for _, route := range f.Routes {
		where := source("routes.txt", route.SourceLine)
		if route.ID == "" {
			v.add("%s: route_id is empty", where)
		} else if previous, exists := routeByID[route.ID]; exists {
			v.add("%s: duplicate route_id %q (first seen on line %d)", where, route.ID, previous.SourceLine)
		} else {
			routeByID[route.ID] = route
		}
		if route.Type < 0 {
			v.add("%s: route_type must not be negative", where)
		}
		if route.Type > math.MaxInt16 {
			v.add("%s: route_type exceeds the graph format range", where)
		}
		if route.AgencyID != "" {
			if _, exists := agencyByID[route.AgencyID]; !exists {
				v.add("%s: agency_id %q does not exist", where, route.AgencyID)
			}
		} else if len(f.Agencies) != 1 {
			v.add("%s: agency_id is required when the feed does not have exactly one agency", where)
		}
		if route.ShortName == "" && route.LongName == "" {
			v.add("%s: route_short_name and route_long_name cannot both be empty", where)
		}
	}

	serviceByID := make(map[string]Service, len(f.Services))
	if len(f.Services) == 0 {
		v.add("calendar.txt has no rows")
	}
	for _, service := range f.Services {
		where := source("calendar.txt", service.SourceLine)
		if service.ID == "" {
			v.add("%s: service_id is empty", where)
		} else if previous, exists := serviceByID[service.ID]; exists {
			v.add("%s: duplicate service_id %q (first seen on line %d)", where, service.ID, previous.SourceLine)
		} else {
			serviceByID[service.ID] = service
		}
		if !validDate(service.StartDate) {
			v.add("%s: start_date %08d is invalid", where, service.StartDate)
		}
		if !validDate(service.EndDate) {
			v.add("%s: end_date %08d is invalid", where, service.EndDate)
		}
		if service.StartDate > service.EndDate {
			v.add("%s: start_date is after end_date", where)
		}
	}

	tripByID := make(map[string]Trip, len(f.Trips))
	if len(f.Trips) == 0 {
		v.add("trips.txt has no rows")
	}
	for _, trip := range f.Trips {
		where := source("trips.txt", trip.SourceLine)
		if trip.ID == "" {
			v.add("%s: trip_id is empty", where)
		} else if previous, exists := tripByID[trip.ID]; exists {
			v.add("%s: duplicate trip_id %q (first seen on line %d)", where, trip.ID, previous.SourceLine)
		} else {
			tripByID[trip.ID] = trip
		}
		if _, exists := routeByID[trip.RouteID]; !exists {
			v.add("%s: route_id %q does not exist", where, trip.RouteID)
		}
		if _, exists := serviceByID[trip.ServiceID]; !exists {
			v.add("%s: service_id %q does not exist", where, trip.ServiceID)
		}
		if trip.DirectionID != -1 && trip.DirectionID != 0 && trip.DirectionID != 1 {
			v.add("%s: direction_id must be empty, 0, or 1", where)
		}
	}

	stopTimesByTrip := make(map[string][]StopTime, len(f.Trips))
	for _, stopTime := range f.StopTimes {
		where := source("stop_times.txt", stopTime.SourceLine)
		if _, exists := tripByID[stopTime.TripID]; !exists {
			v.add("%s: trip_id %q does not exist", where, stopTime.TripID)
		}
		stop, exists := stopByID[stopTime.StopID]
		if !exists {
			v.add("%s: stop_id %q does not exist", where, stopTime.StopID)
		} else if stop.LocationType != 0 {
			v.add("%s: stop_id %q is not a boarding stop/platform", where, stopTime.StopID)
		}
		if stopTime.StopSequence < 0 {
			v.add("%s: stop_sequence must not be negative", where)
		}
		if stopTime.StopSequence > math.MaxInt32 {
			v.add("%s: stop_sequence exceeds the graph format range", where)
		}
		if stopTime.ArrivalSeconds > math.MaxInt32 || stopTime.DepartureSeconds > math.MaxInt32 {
			v.add("%s: stop time exceeds the graph format range", where)
		}
		if stopTime.DepartureSeconds < stopTime.ArrivalSeconds {
			v.add("%s: departure_time is before arrival_time", where)
		}
		stopTimesByTrip[stopTime.TripID] = append(stopTimesByTrip[stopTime.TripID], stopTime)
	}
	for _, trip := range f.Trips {
		stopTimes := stopTimesByTrip[trip.ID]
		if len(stopTimes) < 2 {
			v.add("%s: trip_id %q has fewer than two stop_times", source("trips.txt", trip.SourceLine), trip.ID)
			continue
		}
		sort.Slice(stopTimes, func(i, j int) bool {
			return stopTimes[i].StopSequence < stopTimes[j].StopSequence
		})
		for index := 1; index < len(stopTimes); index++ {
			previous := stopTimes[index-1]
			current := stopTimes[index]
			where := source("stop_times.txt", current.SourceLine)
			if current.StopSequence == previous.StopSequence {
				v.add("%s: duplicate stop_sequence %d for trip_id %q", where, current.StopSequence, trip.ID)
			}
			if current.ArrivalSeconds < previous.DepartureSeconds {
				v.add("%s: arrival_time goes backwards for trip_id %q", where, trip.ID)
			}
		}
	}

	frequenciesByTrip := make(map[string][]Frequency)
	for _, frequency := range f.Frequencies {
		where := source("frequencies.txt", frequency.SourceLine)
		if _, exists := tripByID[frequency.TripID]; !exists {
			v.add("%s: trip_id %q does not exist", where, frequency.TripID)
		}
		if frequency.StartSeconds >= frequency.EndSeconds {
			v.add("%s: start_time must be before end_time", where)
		}
		if frequency.HeadwaySeconds <= 0 {
			v.add("%s: headway_secs must be positive", where)
		}
		if frequency.StartSeconds > math.MaxInt32 || frequency.EndSeconds > math.MaxInt32 || frequency.HeadwaySeconds > math.MaxInt32 {
			v.add("%s: frequency value exceeds the graph format range", where)
		}
		if frequency.ExactTimes != 0 && frequency.ExactTimes != 1 {
			v.add("%s: exact_times must be 0 or 1", where)
		}
		frequenciesByTrip[frequency.TripID] = append(frequenciesByTrip[frequency.TripID], frequency)
	}
	for tripID, frequencies := range frequenciesByTrip {
		sort.Slice(frequencies, func(i, j int) bool {
			return frequencies[i].StartSeconds < frequencies[j].StartSeconds
		})
		for index := 1; index < len(frequencies); index++ {
			if frequencies[index].StartSeconds < frequencies[index-1].EndSeconds {
				v.add("%s: frequency window overlaps the previous window for trip_id %q", source("frequencies.txt", frequencies[index].SourceLine), tripID)
			}
		}
	}

	for _, transfer := range f.Transfers {
		where := source("transfers.txt", transfer.SourceLine)
		if _, exists := stopByID[transfer.FromStopID]; !exists {
			v.add("%s: from_stop_id %q does not exist", where, transfer.FromStopID)
		}
		if _, exists := stopByID[transfer.ToStopID]; !exists {
			v.add("%s: to_stop_id %q does not exist", where, transfer.ToStopID)
		}
		if transfer.Type < 0 || transfer.Type > 3 {
			v.add("%s: transfer_type must be between 0 and 3", where)
		}
		if transfer.MinTransferSeconds < 0 {
			v.add("%s: min_transfer_time must not be negative", where)
		} else if uint64(transfer.MinTransferSeconds) > math.MaxUint32 {
			v.add("%s: min_transfer_time exceeds the graph format range", where)
		}
		if transfer.Type == 2 && transfer.MinTransferSeconds == 0 {
			v.add("%s: transfer_type 2 requires a positive min_transfer_time", where)
		}
	}

	return v.result()
}

func validDate(value int) bool {
	text := fmt.Sprintf("%08d", value)
	parsed, err := time.Parse("20060102", text)
	return err == nil && parsed.Format("20060102") == text
}

func source(table string, line int) string {
	if line <= 0 {
		return table
	}
	return fmt.Sprintf("%s line %d", table, line)
}
