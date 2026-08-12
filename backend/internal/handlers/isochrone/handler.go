package isochrone

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	engine "github.com/SrivathsanRam/IsoMap/internal/isochrone"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/contour"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/graph"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/spatial"
)

const maximumRequestBytes = 1 << 20

type Generator interface {
	Generate(ctx context.Context, query engine.Query) (*engine.Result, error)
}

type Handler struct {
	generator   Generator
	manifest    graph.Manifest
	location    *time.Location
	clock       func() time.Time
}

type request struct {
	Lat           float64 `json:"lat"`
	Lon           float64 `json:"lon"`
	Minutes       int     `json:"minutes"`
	DepartureTime string  `json:"departure_time,omitempty"`
}

type feature struct {
	Type       string             `json:"type"`
	Geometry   contour.MultiPolygon `json:"geometry"`
	Properties properties         `json:"properties"`
}

type properties struct {
	Minutes          int    `json:"minutes"`
	DepartureTime    string `json:"departure_time"`
	FeedVersion      string `json:"feed_version,omitempty"`
	ReachedStopCount int    `json:"reached_stop_count"`
	SettledNodeCount int    `json:"settled_node_count"`
	GridColumns      int    `json:"grid_columns"`
	GridRows         int    `json:"grid_rows"`
	WalkingModel     string `json:"walking_model"`
}

func New(generator Generator, manifest graph.Manifest) (*Handler, error) {
	if generator == nil {
		return nil, fmt.Errorf("isochrone engine is nil")
	}
	location, err := time.LoadLocation(manifest.Timezone)
	if err != nil {
		return nil, fmt.Errorf("load isochrone timezone %q: %w", manifest.Timezone, err)
	}
	return &Handler{
		generator: generator,
		manifest: manifest,
		location: location,
		clock:    time.Now,
	}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maximumRequestBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var body request
	if err := decoder.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	if decoder.Decode(&struct{}{}) == nil {
		writeError(w, http.StatusBadRequest, "request must contain one JSON object")
		return
	}
	if math.IsNaN(body.Lat) || math.IsInf(body.Lat, 0) || body.Lat < -90 || body.Lat > 90 ||
		math.IsNaN(body.Lon) || math.IsInf(body.Lon, 0) || body.Lon < -180 || body.Lon > 180 {
		writeError(w, http.StatusBadRequest, "lat and lon must be valid coordinates")
		return
	}
	if body.Minutes < 5 || body.Minutes > 90 {
		writeError(w, http.StatusBadRequest, "minutes must be between 5 and 90")
		return
	}

	departure, err := h.departureTime(body.DepartureTime)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.generator.Generate(r.Context(), engine.Query{
		Origin:           spatial.Point{Lat: body.Lat, Lon: body.Lon},
		Departure:        departure,
		MaxTravelSeconds: body.Minutes * 60,
	})
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		writeError(w, http.StatusInternalServerError, "could not generate isochrone")
		return
	}

	response := feature{
		Type:     "Feature",
		Geometry: result.Geometry,
		Properties: properties{
			Minutes:          body.Minutes,
			DepartureTime:    departure.In(h.location).Format(time.RFC3339),
			FeedVersion:      h.manifest.FeedVersion,
			ReachedStopCount: result.ReachedStopCount,
			SettledNodeCount: result.SettledNodeCount,
			GridColumns:      result.GridColumns,
			GridRows:         result.GridRows,
			WalkingModel:     "radial-distance-factor",
		},
	}
	w.Header().Set("Content-Type", "application/geo+json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}

func (h *Handler) departureTime(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return h.clock().In(h.location), nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("departure_time must be RFC3339")
	}
	return parsed.In(h.location), nil
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": message})
}
