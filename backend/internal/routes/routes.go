package routes

import (
	"encoding/json"
	"net/http"

	"github.com/CVWO/sample-go-app/internal/handlers/users"
	"github.com/go-chi/chi/v5"
)

type authRequest struct {
	Username string `json:"username"`
}

func GetRoutes() func(r chi.Router) {
	return func(r chi.Router) {
		r.Post("/login", login)
		r.Post("/signup", signup)
		r.Get("/users", func(w http.ResponseWriter, req *http.Request) {
			response, _ := users.HandleList(w, req)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		})
	}
}

func login(w http.ResponseWriter, req *http.Request) {
	var body authRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		write(w, http.StatusBadRequest, "Invalid request")
		return
	}
	if body.Username != "Tester" {
		write(w, http.StatusUnauthorized, "Invalid username")
		return
	}
	write(w, http.StatusOK, "Login successful")
}

func signup(w http.ResponseWriter, req *http.Request) {
	write(w, http.StatusOK, "Signup successful")
}

func write(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}
