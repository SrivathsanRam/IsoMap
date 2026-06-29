package router

import (
	"net/http"
	"os"
	"strings"

	"github.com/SrivathsanRam/IsoMap/internal/routes"
	"github.com/go-chi/chi/v5"
)

func Setup() chi.Router {
	r := chi.NewRouter()
	r.Use(cors)
	setUpRoutes(r)
	return r
}

func setUpRoutes(r chi.Router) {
	r.Group(routes.GetRoutes())
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimRight(r.Header.Get("Origin"), "/")
		if allowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-User-ID")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func allowedOrigin(origin string) bool {
	if origin == "" {
		return false
	}

	for _, allowed := range configuredOrigins() {
		if origin == allowed {
			return true
		}
	}

	return false
}

func configuredOrigins() []string {
	rawOrigins := os.Getenv("FRONTEND_ORIGINS")
	if rawOrigins == "" {
		rawOrigins = os.Getenv("FRONTEND_ORIGIN")
	}
	if rawOrigins == "" {
		rawOrigins = os.Getenv("FRONTEND_URL")
	}
	if rawOrigins == "" {
		rawOrigins = "http://localhost:5173"
	}

	origins := []string{}
	for _, rawOrigin := range strings.Split(rawOrigins, ",") {
		origin := strings.TrimRight(strings.TrimSpace(rawOrigin), "/")
		if origin != "" {
			origins = append(origins, origin)
		}
	}

	return origins
}
