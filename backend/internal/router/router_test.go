package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SrivathsanRam/IsoMap/internal/routes"
)

func TestSetupWiresIsochroneHandler(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	router := Setup(routes.Dependencies{Isochrone: handler})
	request := httptest.NewRequest(http.MethodPost, "/isochrone", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if !called {
		t.Fatal("isochrone handler was not called")
	}
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", response.Code)
	}
}

func TestCORSAllowsDevelopmentAndProductionOrigins(t *testing.T) {
	for _, test := range []struct {
		name             string
		configuredOrigin string
		requestOrigin    string
	}{
		{name: "development", configuredOrigin: "http://localhost:5173", requestOrigin: "http://localhost:5173"},
		{name: "production", configuredOrigin: "https://iso-map.vercel.app/", requestOrigin: "https://iso-map.vercel.app"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("FRONTEND_ORIGINS", test.configuredOrigin)
			t.Setenv("FRONTEND_ORIGIN", "")
			t.Setenv("FRONTEND_URL", "")
			request := httptest.NewRequest(http.MethodGet, "/presets?sort=trending", nil)
			request.Header.Set("Origin", test.requestOrigin)
			response := httptest.NewRecorder()

			cors(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", response.Code)
			}
			if got := response.Header().Get("Access-Control-Allow-Origin"); got != test.requestOrigin {
				t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, test.requestOrigin)
			}
			if got := response.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
				t.Fatalf("Access-Control-Allow-Credentials = %q, want true", got)
			}
		})
	}
}

func TestCORSHandlesPreflight(t *testing.T) {
	t.Setenv("FRONTEND_ORIGINS", "http://localhost:5173,https://iso-map.vercel.app")

	request := httptest.NewRequest(http.MethodOptions, "/isochrone", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	request.Header.Set("Access-Control-Request-Headers", "Content-Type")
	response := httptest.NewRecorder()

	cors(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("preflight reached the application handler")
	})).ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", response.Code)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
	if got := response.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("Access-Control-Allow-Methods is missing")
	}
	if got := response.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatal("Access-Control-Allow-Headers is missing")
	}
}

func TestCORSRejectsUnconfiguredOrigin(t *testing.T) {
	t.Setenv("FRONTEND_ORIGINS", "http://localhost:5173,https://iso-map.vercel.app")

	request := httptest.NewRequest(http.MethodGet, "/presets", nil)
	request.Header.Set("Origin", "https://untrusted.example")
	response := httptest.NewRecorder()

	cors(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(response, request)

	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q for an untrusted origin", got)
	}
}
