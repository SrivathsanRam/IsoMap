package routes

import (
	"encoding/json"
	"net/http"

	"github.com/SrivathsanRam/IsoMap/internal/handlers/addresses"
	authHandlers "github.com/SrivathsanRam/IsoMap/internal/handlers/auth"
	"github.com/SrivathsanRam/IsoMap/internal/handlers/outings"
	placesHandler "github.com/SrivathsanRam/IsoMap/internal/handlers/places"
	"github.com/SrivathsanRam/IsoMap/internal/handlers/presets"
	"github.com/SrivathsanRam/IsoMap/internal/handlers/routing"
	"github.com/SrivathsanRam/IsoMap/internal/handlers/users"
	"github.com/go-chi/chi/v5"
)

type Dependencies struct {
	Isochrone http.Handler
	Places    http.Handler
}

func GetRoutes(dependencies Dependencies) func(r chi.Router) {
	return func(r chi.Router) {
		placesHTTP := dependencies.Places
		if placesHTTP == nil {
			placesHTTP = http.HandlerFunc(placesHandler.HandleSearch)
		}

		r.Post("/auth/google", authHandlers.HandleGoogleLogin)
		r.Get("/auth/me", authHandlers.HandleMe)
		r.Post("/auth/logout", authHandlers.HandleLogout)
		r.Method(http.MethodPost, "/isochrone", dependencies.Isochrone)
		r.Post("/routing/directions", routing.HandleDirections)
		r.Method(http.MethodGet, "/places/search", placesHTTP)
		r.Post("/outings", outings.HandleCreate)
		r.Get("/outings/{token}", outings.HandleGet)
		r.Post("/outings/{token}/members", outings.HandleJoin)
		r.Put("/outings/{token}/members/{memberID}", outings.HandleUpdateMember)
		r.Get("/presets", presets.HandleList)
		r.Post("/presets", presets.HandleCreate)
		r.Post("/presets/{presetID}/vote", presets.HandleVote)
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
