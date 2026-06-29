package routing

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var oneMapClient = http.Client{Timeout: 15 * time.Second}

type point struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type directionsRequest struct {
	Start point  `json:"start"`
	End   point  `json:"end"`
	Mode  string `json:"mode"`
}

type directionsResponse struct {
	Routes []routeOption `json:"routes"`
}

type routeOption struct {
	Summary         string      `json:"summary"`
	DurationSeconds int         `json:"duration_seconds"`
	DistanceMeters  float64     `json:"distance_meters"`
	Geometry        []point     `json:"geometry"`
	Steps           []routeStep `json:"steps"`
}

type routeStep struct {
	Mode            string  `json:"mode"`
	Instruction     string  `json:"instruction"`
	DurationSeconds int     `json:"duration_seconds"`
	DistanceMeters  float64 `json:"distance_meters"`
}

type oneMapRouteResponse struct {
	Plan struct {
		Itineraries []struct {
			Duration     int     `json:"duration"`
			WalkDistance float64 `json:"walkDistance"`
			Legs         []struct {
				Mode        string  `json:"mode"`
				Duration    int     `json:"duration"`
				Distance    float64 `json:"distance"`
				Route       string  `json:"route"`
				RouteID     string  `json:"routeId"`
				RouteShort  string  `json:"routeShortName"`
				From        place   `json:"from"`
				To          place   `json:"to"`
				LegGeometry struct {
					Points string `json:"points"`
				} `json:"legGeometry"`
			} `json:"legs"`
		} `json:"itineraries"`
	} `json:"plan"`
	Error string `json:"error"`
}

type place struct {
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
}

func HandleDirections(w http.ResponseWriter, r *http.Request) {
	var body directionsRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		write(w, http.StatusBadRequest, "Invalid request")
		return
	}
	if !validPoint(body.Start) || !validPoint(body.End) {
		write(w, http.StatusBadRequest, "start and end coordinates are required")
		return
	}

	mode := strings.ToLower(strings.TrimSpace(body.Mode))
	if mode == "" {
		mode = "transit"
	}

	routes, err := fetchRoutes(r, body.Start, body.End, mode)
	if err != nil {
		write(w, http.StatusBadGateway, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(directionsResponse{Routes: routes})
}

func fetchRoutes(r *http.Request, start point, end point, mode string) ([]routeOption, error) {
	query := url.Values{}
	query.Set("start", formatPoint(start))
	query.Set("end", formatPoint(end))
	query.Set("routeType", oneMapRouteType(mode))

	if oneMapRouteType(mode) == "pt" {
		now := time.Now()
		query.Set("date", now.Format("01-02-2006"))
		query.Set("time", now.Format("15:04:05"))
		query.Set("mode", "TRANSIT")
		query.Set("maxWalkDistance", "1000")
		query.Set("numItineraries", "3")
	}

	endpoint := "https://www.onemap.gov.sg/api/public/routingsvc/route?" + query.Encode()
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if token := os.Getenv("ONEMAP_API_KEY"); token != "" {
		req.Header.Set("Authorization", token)
	}

	resp, err := oneMapClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OneMap routing request failed")
	}

	var data oneMapRouteResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if data.Error != "" {
		return nil, errors.New(data.Error)
	}
	if len(data.Plan.Itineraries) == 0 {
		return nil, fmt.Errorf("OneMap returned no routes")
	}

	routes := make([]routeOption, 0, len(data.Plan.Itineraries))
	for _, itinerary := range data.Plan.Itineraries {
		route := routeOption{
			DurationSeconds: itinerary.Duration,
			DistanceMeters:  itinerary.WalkDistance,
			Geometry:        []point{},
			Steps:           []routeStep{},
		}

		for _, leg := range itinerary.Legs {
			if leg.Distance > 0 {
				route.DistanceMeters += leg.Distance
			}
			route.Steps = append(route.Steps, routeStep{
				Mode:            leg.Mode,
				Instruction:     describeLeg(leg.Mode, leg.RouteShort, leg.Route, leg.From.Name, leg.To.Name),
				DurationSeconds: leg.Duration,
				DistanceMeters:  leg.Distance,
			})

			points := decodePolyline(leg.LegGeometry.Points)
			if len(points) == 0 && leg.From.Lat != 0 && leg.To.Lat != 0 {
				points = []point{
					{Lat: leg.From.Lat, Lon: leg.From.Lon},
					{Lat: leg.To.Lat, Lon: leg.To.Lon},
				}
			}
			route.Geometry = appendGeometry(route.Geometry, points)
		}

		if len(route.Geometry) == 0 {
			route.Geometry = []point{start, end}
		}
		route.Summary = summarize(route.Steps)
		routes = append(routes, route)
	}

	return routes, nil
}

func validPoint(p point) bool {
	return p.Lat != 0 && p.Lon != 0
}

func formatPoint(p point) string {
	return strconv.FormatFloat(p.Lat, 'f', 6, 64) + "," + strconv.FormatFloat(p.Lon, 'f', 6, 64)
}

func oneMapRouteType(mode string) string {
	switch mode {
	case "walk", "drive", "cycle":
		return mode
	default:
		return "pt"
	}
}

func describeLeg(mode string, routeShort string, route string, from string, to string) string {
	label := mode
	if routeShort != "" {
		label += " " + routeShort
	} else if route != "" {
		label += " " + route
	}
	if from != "" && to != "" {
		return fmt.Sprintf("%s from %s to %s", strings.Title(strings.ToLower(label)), from, to)
	}
	return strings.Title(strings.ToLower(label))
}

func summarize(steps []routeStep) string {
	modes := []string{}
	for _, step := range steps {
		mode := strings.TrimSpace(step.Mode)
		if mode == "" {
			continue
		}
		if len(modes) == 0 || modes[len(modes)-1] != mode {
			modes = append(modes, mode)
		}
	}
	if len(modes) == 0 {
		return "Route"
	}
	return strings.Join(modes, " -> ")
}

func appendGeometry(existing []point, next []point) []point {
	if len(next) == 0 {
		return existing
	}
	if len(existing) == 0 {
		return append(existing, next...)
	}
	last := existing[len(existing)-1]
	first := next[0]
	if math.Abs(last.Lat-first.Lat) < 0.000001 && math.Abs(last.Lon-first.Lon) < 0.000001 {
		return append(existing, next[1:]...)
	}
	return append(existing, next...)
}

func decodePolyline(encoded string) []point {
	if encoded == "" {
		return nil
	}

	points := []point{}
	index := 0
	lat := 0
	lon := 0

	for index < len(encoded) {
		dLat, nextIndex, ok := decodePolylineValue(encoded, index)
		if !ok {
			return nil
		}
		index = nextIndex

		dLon, nextIndex, ok := decodePolylineValue(encoded, index)
		if !ok {
			return nil
		}
		index = nextIndex

		lat += dLat
		lon += dLon
		points = append(points, point{
			Lat: float64(lat) / 100000,
			Lon: float64(lon) / 100000,
		})
	}

	return points
}

func decodePolylineValue(encoded string, index int) (int, int, bool) {
	result := 0
	shift := 0

	for index < len(encoded) {
		b := int(encoded[index]) - 63
		index++
		result |= (b & 0x1f) << shift
		shift += 5
		if b < 0x20 {
			if result&1 != 0 {
				return ^(result >> 1), index, true
			}
			return result >> 1, index, true
		}
	}

	return 0, index, false
}

func write(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}
