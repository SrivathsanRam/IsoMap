package outings

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/SrivathsanRam/IsoMap/internal/auth"
	"github.com/SrivathsanRam/IsoMap/internal/database"
	"github.com/SrivathsanRam/IsoMap/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type createOutingRequest struct {
	Title string `json:"title"`
}

type joinOutingRequest struct {
	DisplayName string `json:"display_name"`
}

type updateMemberRequest struct {
	LocationName     string  `json:"location_name"`
	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
	MaxTravelMinutes int     `json:"max_travel_minutes"`
}

func HandleCreate(w http.ResponseWriter, r *http.Request) {
	var body createOutingRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		write(w, http.StatusBadRequest, "Invalid request")
		return
	}

	title := strings.TrimSpace(body.Title)
	if title == "" {
		title = "Group outing"
	}

	db, err := database.GetDB()
	if err != nil {
		write(w, http.StatusInternalServerError, "Failed to connect to database")
		return
	}

	var userID *uuid.UUID
	if currentUserID, err := auth.CurrentUserID(r); err == nil {
		userID = &currentUserID
	}

	outing := models.Outing{
		JoinToken:     newJoinToken(),
		Title:         title,
		CreatedByUser: userID,
	}
	if err := db.DB.Create(&outing).Error; err != nil {
		write(w, http.StatusInternalServerError, "Failed to create outing")
		return
	}

	writeJSON(w, http.StatusCreated, outing)
}

func HandleGet(w http.ResponseWriter, r *http.Request) {
	outing, ok := getOuting(w, r)
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, outing)
}

func HandleJoin(w http.ResponseWriter, r *http.Request) {
	outing, ok := getOuting(w, r)
	if !ok {
		return
	}

	var body joinOutingRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		write(w, http.StatusBadRequest, "Invalid request")
		return
	}

	displayName := strings.TrimSpace(body.DisplayName)
	if displayName == "" {
		write(w, http.StatusBadRequest, "display_name is required")
		return
	}

	db, err := database.GetDB()
	if err != nil {
		write(w, http.StatusInternalServerError, "Failed to connect to database")
		return
	}

	var userID *uuid.UUID
	if currentUserID, err := auth.CurrentUserID(r); err == nil {
		userID = &currentUserID
	}

	member := models.OutingMember{
		OutingID:         outing.ID,
		UserID:           userID,
		DisplayName:      displayName,
		MaxTravelMinutes: 15,
	}
	if err := db.DB.Create(&member).Error; err != nil {
		write(w, http.StatusInternalServerError, "Failed to join outing")
		return
	}

	writeJSON(w, http.StatusCreated, member)
}

func HandleUpdateMember(w http.ResponseWriter, r *http.Request) {
	outing, ok := getOuting(w, r)
	if !ok {
		return
	}

	memberID, err := uuid.Parse(chi.URLParam(r, "memberID"))
	if err != nil {
		write(w, http.StatusBadRequest, "Invalid member ID")
		return
	}

	var body updateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		write(w, http.StatusBadRequest, "Invalid request")
		return
	}
	if body.Latitude == 0 || body.Longitude == 0 {
		write(w, http.StatusBadRequest, "latitude and longitude are required")
		return
	}
	if body.MaxTravelMinutes < 5 {
		body.MaxTravelMinutes = 5
	}
	if body.MaxTravelMinutes > 90 {
		body.MaxTravelMinutes = 90
	}

	db, err := database.GetDB()
	if err != nil {
		write(w, http.StatusInternalServerError, "Failed to connect to database")
		return
	}

	var member models.OutingMember
	if err := db.DB.First(&member, "id = ? AND outing_id = ?", memberID, outing.ID).Error; err != nil {
		write(w, http.StatusNotFound, "Member not found")
		return
	}

	member.LocationName = strings.TrimSpace(body.LocationName)
	member.Latitude = &body.Latitude
	member.Longitude = &body.Longitude
	member.MaxTravelMinutes = body.MaxTravelMinutes

	if err := db.DB.Save(&member).Error; err != nil {
		write(w, http.StatusInternalServerError, "Failed to update member")
		return
	}

	writeJSON(w, http.StatusOK, member)
}

func getOuting(w http.ResponseWriter, r *http.Request) (*models.Outing, bool) {
	token := strings.TrimSpace(chi.URLParam(r, "token"))
	if token == "" {
		write(w, http.StatusBadRequest, "Missing outing token")
		return nil, false
	}

	db, err := database.GetDB()
	if err != nil {
		write(w, http.StatusInternalServerError, "Failed to connect to database")
		return nil, false
	}

	var outing models.Outing
	err = db.DB.
		Preload("Members", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("created_at ASC")
		}).
		First(&outing, "join_token = ?", token).Error
	if err != nil {
		write(w, http.StatusNotFound, "Outing not found")
		return nil, false
	}

	return &outing, true
}

func newJoinToken() string {
	bytes := make([]byte, 18)
	if _, err := rand.Read(bytes); err != nil {
		return uuid.NewString()
	}
	return strings.TrimRight(base64.URLEncoding.EncodeToString(bytes), "=")
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
