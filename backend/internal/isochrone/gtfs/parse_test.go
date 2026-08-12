package gtfs

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseArchiveAndValidate(t *testing.T) {
	archive := writeTestArchive(t, validTables())
	feed, err := ParseArchive(archive)
	if err != nil {
		t.Fatalf("ParseArchive() error = %v", err)
	}
	if err := feed.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if len(feed.Stops) != 2 || len(feed.Trips) != 1 || len(feed.StopTimes) != 2 {
		t.Fatalf("unexpected parsed counts: stops=%d trips=%d stop_times=%d", len(feed.Stops), len(feed.Trips), len(feed.StopTimes))
	}
	if got, want := feed.StopTimes[1].ArrivalSeconds, 25*3600+5*60; got != want {
		t.Fatalf("time after midnight = %d, want %d", got, want)
	}
}

func TestParseArchiveRequiresCoreTables(t *testing.T) {
	tables := validTables()
	delete(tables, "stop_times.txt")
	_, err := ParseArchive(writeTestArchive(t, tables))
	if err == nil || !strings.Contains(err.Error(), "missing required file stop_times.txt") {
		t.Fatalf("ParseArchive() error = %v, want missing stop_times.txt", err)
	}
}

func TestValidateReportsMultipleSemanticErrors(t *testing.T) {
	feed, err := ParseArchive(writeTestArchive(t, validTables()))
	if err != nil {
		t.Fatalf("ParseArchive() error = %v", err)
	}
	feed.StopTimes[1].StopID = "missing"
	feed.Frequencies[0].HeadwaySeconds = 0

	err = feed.Validate()
	var validationError *ValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("Validate() error = %v, want ValidationError", err)
	}
	if validationError.Total < 2 {
		t.Fatalf("ValidationError.Total = %d, want at least 2", validationError.Total)
	}
	if !strings.Contains(err.Error(), "stop_id \"missing\"") || !strings.Contains(err.Error(), "headway_secs") {
		t.Fatalf("Validate() error does not contain both problems: %v", err)
	}
}

func validTables() map[string]string {
	return map[string]string{
		"feed_info.txt":   "feed_publisher_name,feed_publisher_url,feed_lang,feed_version\nTest,https://example.com,en,test-v1\n",
		"agency.txt":      "agency_id,agency_name,agency_url,agency_timezone\nA,Test Transit,https://example.com,Asia/Singapore\n",
		"stops.txt":       "stop_id,stop_name,stop_lat,stop_lon,location_type,parent_station\nS1,First,1.30,103.80,0,\nS2,Second,1.31,103.81,0,\n",
		"routes.txt":      "route_id,agency_id,route_short_name,route_long_name,route_type\nR1,A,R1,Test Route,3\n",
		"calendar.txt":    "service_id,monday,tuesday,wednesday,thursday,friday,saturday,sunday,start_date,end_date\nDAILY,1,1,1,1,1,1,1,20260101,20301231\n",
		"trips.txt":       "route_id,service_id,trip_id,direction_id\nR1,DAILY,T1,0\n",
		"stop_times.txt":  "trip_id,arrival_time,departure_time,stop_id,stop_sequence\nT1,24:55:00,24:55:00,S1,1\nT1,25:05:00,25:05:00,S2,2\n",
		"frequencies.txt": "trip_id,start_time,end_time,headway_secs,exact_times\nT1,24:00:00,26:00:00,600,0\n",
		"transfers.txt":   "from_stop_id,to_stop_id,transfer_type,min_transfer_time\nS2,S1,2,60\n",
	}
}

func writeTestArchive(t *testing.T, tables map[string]string) string {
	t.Helper()
	archivePath := filepath.Join(t.TempDir(), "feed.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for name, contents := range tables {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return archivePath
}
