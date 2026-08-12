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
