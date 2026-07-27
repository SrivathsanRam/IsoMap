package presets

import (
	"encoding/json"
	"math"
	"net/http"
	"strings"

	"github.com/SrivathsanRam/IsoMap/internal/database"
	"github.com/SrivathsanRam/IsoMap/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type createPresetRequest struct {
	Name      string                  `json:"name"`
	Locations []createLocationRequest `json:"locations"`
}

type createLocationRequest struct {
	Name      string  `json:"name"`
	Address   string  `json:"address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type voteRequest struct {
	Vote         string `json:"vote"`
	PreviousVote string `json:"previous_vote"`
}

func HandleList(w http.ResponseWriter, r *http.Request) {
	db, err := database.GetDB()
	if err != nil {
		write(w, http.StatusInternalServerError, "Failed to connect to database")
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	presetsQuery := db.DB.
		Preload("Locations", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("created_at ASC")
		})

	if query != "" {
		like := "%" + query + "%"
		presetsQuery = presetsQuery.Where(
			`name ILIKE ? OR EXISTS (
				SELECT 1
				FROM community_preset_locations
				WHERE community_preset_locations.preset_id = community_presets.id
				AND (community_preset_locations.name ILIKE ? OR community_preset_locations.address ILIKE ?)
			)`,
			like,
			like,
			like,
		)
	}

	switch strings.ToLower(strings.TrimSpace(r.URL.Query().Get("sort"))) {
	case "top":
		presetsQuery = presetsQuery.Order("(upvotes - downvotes) DESC, upvotes DESC, created_at DESC")
	case "trending":
		presetsQuery = presetsQuery.Order("(upvotes + downvotes) DESC, (upvotes - downvotes) DESC, created_at DESC")
	default:
		presetsQuery = presetsQuery.Order("created_at DESC")
	}

	var presets []models.CommunityPreset
	if err := presetsQuery.Find(&presets).Error; err != nil {
		write(w, http.StatusInternalServerError, "Failed to list presets")
		return
	}

	for index := range presets {
		presets[index].Source = "community"
	}

	writeJSON(w, http.StatusOK, presets)
}

func HandleCreate(w http.ResponseWriter, r *http.Request) {
	var body createPresetRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		write(w, http.StatusBadRequest, "Invalid request")
		return
	}

	name := strings.TrimSpace(body.Name)
	if name == "" {
		write(w, http.StatusBadRequest, "name is required")
		return
	}
	if len(body.Locations) == 0 {
		write(w, http.StatusBadRequest, "at least one location is required")
		return
	}

	locations := make([]models.CommunityPresetLocation, 0, len(body.Locations))
	for _, rawLocation := range body.Locations {
		location := models.CommunityPresetLocation{
			Name:      strings.TrimSpace(rawLocation.Name),
			Address:   strings.TrimSpace(rawLocation.Address),
			Latitude:  rawLocation.Latitude,
			Longitude: rawLocation.Longitude,
		}
		if location.Name == "" {
			location.Name = location.Address
		}
		if location.Name == "" {
			write(w, http.StatusBadRequest, "location name is required")
			return
		}
		if !validCoordinate(location.Latitude, location.Longitude) {
			write(w, http.StatusBadRequest, "location coordinates are invalid")
			return
		}
		locations = append(locations, location)
	}

	preset := models.CommunityPreset{
		Name:      name,
		Source:    "community",
		Locations: locations,
	}

	db, err := database.GetDB()
	if err != nil {
		write(w, http.StatusInternalServerError, "Failed to connect to database")
		return
	}
	if err := db.DB.Create(&preset).Error; err != nil {
		write(w, http.StatusInternalServerError, "Failed to create preset")
		return
	}

	writeJSON(w, http.StatusCreated, preset)
}

func HandleVote(w http.ResponseWriter, r *http.Request) {
	presetID, err := uuid.Parse(chi.URLParam(r, "presetID"))
	if err != nil {
		write(w, http.StatusBadRequest, "Invalid preset ID")
		return
	}

	var body voteRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		write(w, http.StatusBadRequest, "Invalid request")
		return
	}

	previousVote := normalizeVote(body.PreviousVote)
	nextVote := normalizeVote(body.Vote)
	upvoteDelta, downvoteDelta := voteDeltas(previousVote, nextVote)

	db, err := database.GetDB()
	if err != nil {
		write(w, http.StatusInternalServerError, "Failed to connect to database")
		return
	}

	err = db.DB.Model(&models.CommunityPreset{}).
		Where("id = ?", presetID).
		Updates(map[string]any{
			"upvotes":   gorm.Expr("GREATEST(upvotes + ?, 0)", upvoteDelta),
			"downvotes": gorm.Expr("GREATEST(downvotes + ?, 0)", downvoteDelta),
		}).Error
	if err != nil {
		write(w, http.StatusInternalServerError, "Failed to update vote")
		return
	}

	var preset models.CommunityPreset
	if err := db.DB.Preload("Locations").First(&preset, "id = ?", presetID).Error; err != nil {
		write(w, http.StatusNotFound, "Preset not found")
		return
	}
	preset.Source = "community"

	writeJSON(w, http.StatusOK, preset)
}

func validCoordinate(latitude float64, longitude float64) bool {
	return !math.IsNaN(latitude) &&
		!math.IsInf(latitude, 0) &&
		!math.IsNaN(longitude) &&
		!math.IsInf(longitude, 0) &&
		latitude >= -90 &&
		latitude <= 90 &&
		longitude >= -180 &&
		longitude <= 180
}

func normalizeVote(vote string) string {
	switch strings.ToLower(strings.TrimSpace(vote)) {
	case "up", "down":
		return strings.ToLower(strings.TrimSpace(vote))
	default:
		return ""
	}
}

func voteDeltas(previousVote string, nextVote string) (int, int) {
	upvoteDelta := 0
	downvoteDelta := 0
	if previousVote == "up" {
		upvoteDelta--
	}
	if previousVote == "down" {
		downvoteDelta--
	}
	if nextVote == "up" {
		upvoteDelta++
	}
	if nextVote == "down" {
		downvoteDelta++
	}
	return upvoteDelta, downvoteDelta
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func write(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}
