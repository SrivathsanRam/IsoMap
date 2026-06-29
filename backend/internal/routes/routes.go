package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/SrivathsanRam/IsoMap/internal/handlers/addresses"
	authHandlers "github.com/SrivathsanRam/IsoMap/internal/handlers/auth"
	"github.com/SrivathsanRam/IsoMap/internal/handlers/routing"
	"github.com/SrivathsanRam/IsoMap/internal/handlers/users"
	"github.com/go-chi/chi/v5"
)

var mapboxClient = http.Client{Timeout: 10 * time.Second}

type isochroneRequest struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type point struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type mapboxIsochrone struct {
	Features []struct {
		Geometry struct {
			Coordinates [][][]float64 `json:"coordinates"`
		} `json:"geometry"`
	} `json:"features"`
}

func GetRoutes() func(r chi.Router) {
	return func(r chi.Router) {
		r.Post("/auth/google", authHandlers.HandleGoogleLogin)
		r.Get("/auth/me", authHandlers.HandleMe)
		r.Post("/auth/logout", authHandlers.HandleLogout)
		r.Post("/isochrone", isochrone)
		r.Post("/routing/directions", routing.HandleDirections)
		r.Get("/users", func(w http.ResponseWriter, req *http.Request) {
			response, _ := users.HandleList(w, req)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		})
		r.Post("/users/{userID}/recent-addresses", addresses.HandleCreateRecent)
		r.Get("/users/{userID}/recent-addresses", addresses.HandleListRecent)
		r.Post("/users/{userID}/saved-addresses", addresses.HandleCreateSaved)
		r.Get("/users/{userID}/saved-addresses", addresses.HandleListSaved)
	}
}

func isochrone(w http.ResponseWriter, req *http.Request) {
	var body isochroneRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		write(w, http.StatusBadRequest, "Invalid request")
		return
	}

	points, err := fetchIsochrone(body)
	if err != nil {
		write(w, http.StatusBadGateway, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(points)
}

func fetchIsochrone(body isochroneRequest) ([]point, error) {
	token := os.Getenv("MAPBOX_ACCESS_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("MAPBOX_ACCESS_TOKEN is not set")
	}

	query := url.Values{}
	query.Set("contours_minutes", "15")
	query.Set("polygons", "true")
	query.Set("access_token", token)

	endpoint := fmt.Sprintf(
		"https://api.mapbox.com/isochrone/v1/mapbox/walking/%f,%f?%s",
		body.Lon,
		body.Lat,
		query.Encode(),
	)
	resp, err := mapboxClient.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Mapbox isochrone request failed")
	}

	var data mapboxIsochrone
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if len(data.Features) == 0 || len(data.Features[0].Geometry.Coordinates) == 0 {
		return nil, fmt.Errorf("Mapbox returned no polygon")
	}

	ring := data.Features[0].Geometry.Coordinates[0]
	points := make([]point, 0, len(ring)+1)
	for _, coord := range ring {
		if len(coord) >= 2 {
			points = append(points, point{Lat: coord[1], Lon: coord[0]})
		}
	}
	if len(points) > 0 && points[0] != points[len(points)-1] {
		points = append(points, points[0])
	}
	return points, nil
}

func write(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}
