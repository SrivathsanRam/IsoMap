package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestGetRoutesWiresPlaceSearchHandler(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Query().Get("q") != "raffles" {
			t.Errorf("q = %q", r.URL.Query().Get("q"))
		}
		w.WriteHeader(http.StatusNoContent)
	})
	router := chi.NewRouter()
	router.Group(GetRoutes(Dependencies{Places: handler}))

	request := httptest.NewRequest(http.MethodGet, "/places/search?q=raffles", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if !called {
		t.Fatal("place search handler was not called")
	}
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", response.Code)
	}
}
