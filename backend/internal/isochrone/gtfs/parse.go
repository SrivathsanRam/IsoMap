package gtfs

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
)

// ParseArchive reads the GTFS tables needed by the graph builder directly
// from a ZIP archive. It validates CSV shape and scalar field syntax; callers
// should then call Feed.Validate for cross-table and semantic validation.
func ParseArchive(archivePath string) (*Feed, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, fmt.Errorf("open GTFS archive: %w", err)
	}
	defer reader.Close()

	files := make(map[string]*zip.File, len(reader.File))
	for _, file := range reader.File {
		name := strings.ToLower(path.Base(file.Name))
		if _, exists := files[name]; exists {
			return nil, fmt.Errorf("GTFS archive contains duplicate %s", name)
		}
		files[name] = file
	}

	feed := &Feed{}
	if err := parseFeedInfo(files, feed); err != nil {
		return nil, err
	}
	if err := parseAgencies(files, feed); err != nil {
		return nil, err
	}
	if err := parseStops(files, feed); err != nil {
		return nil, err
	}
	if err := parseRoutes(files, feed); err != nil {
		return nil, err
	}
	if err := parseCalendar(files, feed); err != nil {
		return nil, err
	}
	if err := parseTrips(files, feed); err != nil {
		return nil, err
	}
	if err := parseStopTimes(files, feed); err != nil {
		return nil, err
	}
	if err := parseFrequencies(files, feed); err != nil {
		return nil, err
	}
	if err := parseTransfers(files, feed); err != nil {
		return nil, err
	}

	return feed, nil
}

type record struct {
	values []string
	index  map[string]int
}

func (r record) value(field string) string {
	index, ok := r.index[field]
	if !ok || index >= len(r.values) {
		return ""
	}
	return strings.TrimSpace(r.values[index])
}

func readTable(
	files map[string]*zip.File,
	name string,
	required bool,
	requiredHeaders []string,
	visit func(record, int) error,
) error {
	file, ok := files[strings.ToLower(name)]
	if !ok {
		if required {
			return fmt.Errorf("GTFS archive is missing required file %s", name)
		}
		return nil
	}

	stream, err := file.Open()
	if err != nil {
		return fmt.Errorf("open %s: %w", name, err)
	}
	defer stream.Close()

	reader := csv.NewReader(stream)
	reader.ReuseRecord = false
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("read %s header: %w", name, err)
	}

	index := make(map[string]int, len(header))
	for column, rawName := range header {
		field := strings.TrimPrefix(strings.TrimSpace(rawName), "\ufeff")
		if field == "" {
			return fmt.Errorf("%s has an empty header at column %d", name, column+1)
		}
		if _, duplicate := index[field]; duplicate {
			return fmt.Errorf("%s has duplicate header %q", name, field)
		}
		index[field] = column
	}
	for _, field := range requiredHeaders {
		if _, ok := index[field]; !ok {
			return fmt.Errorf("%s is missing required column %q", name, field)
		}
	}

	reader.FieldsPerRecord = len(header)
	for line := 2; ; line++ {
		values, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read %s line %d: %w", name, line, err)
		}
		if err := visit(record{values: values, index: index}, line); err != nil {
			return fmt.Errorf("%s line %d: %w", name, line, err)
		}
	}
	return nil
}

func parseFeedInfo(files map[string]*zip.File, feed *Feed) error {
	rows := 0
	return readTable(files, "feed_info.txt", false, nil, func(row record, _ int) error {
		rows++
		if rows > 1 {
			return fmt.Errorf("multiple feed_info rows are not supported")
		}
		feed.Info = FeedInfo{
			PublisherName: row.value("feed_publisher_name"),
			PublisherURL:  row.value("feed_publisher_url"),
			Language:      row.value("feed_lang"),
			Version:       row.value("feed_version"),
		}
		return nil
	})
}

func parseAgencies(files map[string]*zip.File, feed *Feed) error {
	headers := []string{"agency_name", "agency_url", "agency_timezone"}
	return readTable(files, "agency.txt", true, headers, func(row record, line int) error {
		feed.Agencies = append(feed.Agencies, Agency{
			ID:         row.value("agency_id"),
			Name:       row.value("agency_name"),
			URL:        row.value("agency_url"),
			Timezone:   row.value("agency_timezone"),
			SourceLine: line,
		})
		return nil
	})
}

func parseStops(files map[string]*zip.File, feed *Feed) error {
	headers := []string{"stop_id", "stop_name", "stop_lat", "stop_lon"}
	return readTable(files, "stops.txt", true, headers, func(row record, line int) error {
		lat, err := parseFloat(row, "stop_lat")
		if err != nil {
			return err
		}
		lon, err := parseFloat(row, "stop_lon")
		if err != nil {
			return err
		}
		locationType, err := parseOptionalInt(row, "location_type", 0)
		if err != nil {
			return err
		}
		feed.Stops = append(feed.Stops, Stop{
			ID:            row.value("stop_id"),
			Name:          row.value("stop_name"),
			Lat:           lat,
			Lon:           lon,
			LocationType:  locationType,
			ParentStation: row.value("parent_station"),
			SourceLine:    line,
		})
		return nil
	})
}

func parseRoutes(files map[string]*zip.File, feed *Feed) error {
	headers := []string{"route_id", "route_type"}
	return readTable(files, "routes.txt", true, headers, func(row record, line int) error {
		routeType, err := parseInt(row, "route_type")
		if err != nil {
			return err
		}
		feed.Routes = append(feed.Routes, Route{
			ID:         row.value("route_id"),
			AgencyID:   row.value("agency_id"),
			ShortName:  row.value("route_short_name"),
			LongName:   row.value("route_long_name"),
			Type:       routeType,
			SourceLine: line,
		})
		return nil
	})
}

func parseCalendar(files map[string]*zip.File, feed *Feed) error {
	headers := []string{
		"service_id", "monday", "tuesday", "wednesday", "thursday",
		"friday", "saturday", "sunday", "start_date", "end_date",
	}
	return readTable(files, "calendar.txt", true, headers, func(row record, line int) error {
		dayFields := [...]string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
		var days [7]bool
		for index, field := range dayFields {
			value, err := parseInt(row, field)
			if err != nil {
				return err
			}
			if value != 0 && value != 1 {
				return fmt.Errorf("%s must be 0 or 1", field)
			}
			days[index] = value == 1
		}
		startDate, err := parseInt(row, "start_date")
		if err != nil {
			return err
		}
		endDate, err := parseInt(row, "end_date")
		if err != nil {
			return err
		}
		feed.Services = append(feed.Services, Service{
			ID:         row.value("service_id"),
			Days:       days,
			StartDate:  startDate,
			EndDate:    endDate,
			SourceLine: line,
		})
		return nil
	})
}

func parseTrips(files map[string]*zip.File, feed *Feed) error {
	headers := []string{"route_id", "service_id", "trip_id"}
	return readTable(files, "trips.txt", true, headers, func(row record, line int) error {
		directionID, err := parseOptionalInt(row, "direction_id", -1)
		if err != nil {
			return err
		}
		feed.Trips = append(feed.Trips, Trip{
			ID:          row.value("trip_id"),
			RouteID:     row.value("route_id"),
			ServiceID:   row.value("service_id"),
			DirectionID: directionID,
			SourceLine:  line,
		})
		return nil
	})
}

func parseStopTimes(files map[string]*zip.File, feed *Feed) error {
	headers := []string{"trip_id", "arrival_time", "departure_time", "stop_id", "stop_sequence"}
	return readTable(files, "stop_times.txt", true, headers, func(row record, line int) error {
		arrival, err := parseGTFSTime(row.value("arrival_time"))
		if err != nil {
			return fmt.Errorf("arrival_time: %w", err)
		}
		departure, err := parseGTFSTime(row.value("departure_time"))
		if err != nil {
			return fmt.Errorf("departure_time: %w", err)
		}
		sequence, err := parseInt(row, "stop_sequence")
		if err != nil {
			return err
		}
		feed.StopTimes = append(feed.StopTimes, StopTime{
			TripID:           row.value("trip_id"),
			StopID:           row.value("stop_id"),
			ArrivalSeconds:   arrival,
			DepartureSeconds: departure,
			StopSequence:     sequence,
			SourceLine:       line,
		})
		return nil
	})
}

func parseFrequencies(files map[string]*zip.File, feed *Feed) error {
	headers := []string{"trip_id", "start_time", "end_time", "headway_secs"}
	return readTable(files, "frequencies.txt", false, headers, func(row record, line int) error {
		start, err := parseGTFSTime(row.value("start_time"))
		if err != nil {
			return fmt.Errorf("start_time: %w", err)
		}
		end, err := parseGTFSTime(row.value("end_time"))
		if err != nil {
			return fmt.Errorf("end_time: %w", err)
		}
		headway, err := parseInt(row, "headway_secs")
		if err != nil {
			return err
		}
		exactTimes, err := parseOptionalInt(row, "exact_times", 0)
		if err != nil {
			return err
		}
		feed.Frequencies = append(feed.Frequencies, Frequency{
			TripID:         row.value("trip_id"),
			StartSeconds:   start,
			EndSeconds:     end,
			HeadwaySeconds: headway,
			ExactTimes:     exactTimes,
			SourceLine:     line,
		})
		return nil
	})
}

func parseTransfers(files map[string]*zip.File, feed *Feed) error {
	headers := []string{"from_stop_id", "to_stop_id"}
	return readTable(files, "transfers.txt", false, headers, func(row record, line int) error {
		transferType, err := parseOptionalInt(row, "transfer_type", 0)
		if err != nil {
			return err
		}
		minimum, err := parseOptionalInt(row, "min_transfer_time", 0)
		if err != nil {
			return err
		}
		feed.Transfers = append(feed.Transfers, Transfer{
			FromStopID:         row.value("from_stop_id"),
			ToStopID:           row.value("to_stop_id"),
			Type:               transferType,
			MinTransferSeconds: minimum,
			SourceLine:         line,
		})
		return nil
	})
}

func parseInt(row record, field string) (int, error) {
	value := row.value(field)
	if value == "" {
		return 0, fmt.Errorf("%s is required", field)
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s %q is not an integer", field, value)
	}
	return parsed, nil
}

func parseOptionalInt(row record, field string, fallback int) (int, error) {
	if row.value(field) == "" {
		return fallback, nil
	}
	return parseInt(row, field)
}

func parseFloat(row record, field string) (float64, error) {
	value := row.value(field)
	if value == "" {
		return 0, fmt.Errorf("%s is required", field)
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("%s %q is not a number", field, value)
	}
	return parsed, nil
}

func parseGTFSTime(value string) (int, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("%q is not HH:MM:SS", value)
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 {
		return 0, fmt.Errorf("%q has an invalid hour", value)
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return 0, fmt.Errorf("%q has an invalid minute", value)
	}
	second, err := strconv.Atoi(parts[2])
	if err != nil || second < 0 || second > 59 {
		return 0, fmt.Errorf("%q has an invalid second", value)
	}
	return hour*3600 + minute*60 + second, nil
}
