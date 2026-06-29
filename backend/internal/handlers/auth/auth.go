package auth

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/SrivathsanRam/IsoMap/internal/api"
	sessionauth "github.com/SrivathsanRam/IsoMap/internal/auth"
	users "github.com/SrivathsanRam/IsoMap/internal/dataaccess/users"
	"github.com/SrivathsanRam/IsoMap/internal/database"
	"google.golang.org/api/idtoken"
)

type googleLoginRequest struct {
	Credential string `json:"credential"`
}

func HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	var body googleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if body.Credential == "" {
		writeError(w, http.StatusBadRequest, "credential is required")
		return
	}

	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		writeError(w, http.StatusInternalServerError, "GOOGLE_CLIENT_ID is not set")
		return
	}

	payload, err := idtoken.Validate(r.Context(), body.Credential, clientID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid Google credential")
		return
	}

	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	pictureURL, _ := payload.Claims["picture"].(string)
	emailVerified, _ := payload.Claims["email_verified"].(bool)

	if payload.Subject == "" || email == "" || !emailVerified {
		writeError(w, http.StatusUnauthorized, "Google account email is not verified")
		return
	}
	if name == "" {
		name = email
	}

	db, err := database.GetDB()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to connect to database")
		return
	}

	user, err := users.FindOrCreateGoogle(db, users.GoogleProfile{
		Subject:    payload.Subject,
		Email:      email,
		Name:       name,
		PictureURL: pictureURL,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save user")
		return
	}

	if err := sessionauth.SetUserID(w, r, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	writeData(w, http.StatusOK, user, "Successfully signed in")
}

func HandleMe(w http.ResponseWriter, r *http.Request) {
	userID, err := sessionauth.CurrentUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Not signed in")
		return
	}

	db, err := database.GetDB()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to connect to database")
		return
	}

	user, err := users.FindByID(db, userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Not signed in")
		return
	}

	writeData(w, http.StatusOK, user, "Successfully retrieved current user")
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	if err := sessionauth.Clear(w, r); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to clear session")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Logged out"})
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
