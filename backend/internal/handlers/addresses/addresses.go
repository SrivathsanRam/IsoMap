package addresses

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/SrivathsanRam/IsoMap/internal/api"
	addressStore "github.com/SrivathsanRam/IsoMap/internal/dataaccess/addresses"
	"github.com/SrivathsanRam/IsoMap/internal/database"
	"github.com/SrivathsanRam/IsoMap/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type addressRequest struct {
	FormattedAddress string  `json:"formatted_address"`
	PlaceID          string  `json:"place_id"`
	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
	QueryText        string  `json:"query_text"`
	Nickname         string  `json:"nickname"`
}

func HandleCreateRecent(w http.ResponseWriter, r *http.Request) {
	db, userID, ok := requestContext(w, r)
	if !ok {
		return
	}

	var body addressRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if body.FormattedAddress == "" {
		writeError(w, http.StatusBadRequest, "formatted_address is required")
		return
	}

	address, err := addressStore.FindOrCreate(db, models.Address{
		FormattedAddress: body.FormattedAddress,
		PlaceID:          body.PlaceID,
		Latitude:         body.Latitude,
		Longitude:        body.Longitude,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create address")
		return
	}

	if err := addressStore.AddSearch(db, userID, address.ID, body.QueryText); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save recent address")
		return
	}

	writeData(w, http.StatusCreated, address, "Successfully saved recent address")
}

func HandleListRecent(w http.ResponseWriter, r *http.Request) {
	db, userID, ok := requestContext(w, r)
	if !ok {
		return
	}

	limit := 10
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 50 {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and 50")
			return
		}
		limit = parsed
	}

	searches, err := addressStore.ListRecent(db, userID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list recent addresses")
		return
	}

	writeData(w, http.StatusOK, searches, "Successfully listed recent addresses")
}

func HandleCreateSaved(w http.ResponseWriter, r *http.Request) {
	db, userID, ok := requestContext(w, r)
	if !ok {
		return
	}

	var body addressRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if body.FormattedAddress == "" {
		writeError(w, http.StatusBadRequest, "formatted_address is required")
		return
	}

	address, err := addressStore.FindOrCreate(db, models.Address{
		FormattedAddress: body.FormattedAddress,
		PlaceID:          body.PlaceID,
		Latitude:         body.Latitude,
		Longitude:        body.Longitude,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create address")
		return
	}

	if err := addressStore.SaveAddress(db, userID, address.ID, body.Nickname); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save address")
		return
	}

	writeData(w, http.StatusCreated, address, "Successfully saved address")
}

func HandleListSaved(w http.ResponseWriter, r *http.Request) {
	db, userID, ok := requestContext(w, r)
	if !ok {
		return
	}

	saved, err := addressStore.ListSaved(db, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list saved addresses")
		return
	}

	writeData(w, http.StatusOK, saved, "Successfully listed saved addresses")
}

func requestContext(w http.ResponseWriter, r *http.Request) (*database.Database, uuid.UUID, bool) {
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user ID")
		return nil, uuid.Nil, false
	}

	db, err := database.GetDB()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to connect to database")
		return nil, uuid.Nil, false
	}

	return db, userID, true
}

func writeData(w http.ResponseWriter, status int, data any, message string) {
	payload, err := json.Marshal(data)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to encode response")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(api.Response{
		Payload: api.Payload{
			Data: payload,
		},
		Messages: []string{message},
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(api.Response{
		Messages:  []string{message},
		ErrorCode: status,
	})
}
