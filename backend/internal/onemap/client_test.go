package onemap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type rotatingTokenSource struct {
	mu            sync.Mutex
	token         string
	invalidations int
}

func (s *rotatingTokenSource) Token(context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.token, nil
}

func (s *rotatingTokenSource) Invalidate(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if token == s.token {
		s.invalidations++
		s.token = "fresh-token"
	}
}

func TestClientSearchUsesOfficialContractAndRetriesRejectedToken(t *testing.T) {
	tokens := &rotatingTokenSource{token: "expired-token"}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/api/common/elastic/search" {
			t.Errorf("path = %q", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("searchVal") != "Revenue House" || query.Get("returnGeom") != "Y" ||
			query.Get("getAddrDetails") != "Y" || query.Get("pageNum") != "1" {
			t.Errorf("query = %v", query)
		}
		if r.Header.Get("Authorization") == "expired-token" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error":   "Authentication token expired.",
				"results": []any{},
			})
			return
		}
		if r.Header.Get("Authorization") != "fresh-token" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]string{{
				"SEARCHVAL": "REVENUE HOUSE",
				"LATITUDE":  "1.3195",
				"LONGITUDE": "103.8426",
			}},
		})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{
		HTTPClient: server.Client(),
		BaseURL:    server.URL,
		Tokens:     tokens,
	})
	if err != nil {
		t.Fatal(err)
	}
	results, err := client.Search(context.Background(), "Revenue House")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].SearchValue != "REVENUE HOUSE" {
		t.Fatalf("results = %+v", results)
	}
	if requests != 2 || tokens.invalidations != 1 {
		t.Fatalf("requests = %d, invalidations = %d", requests, tokens.invalidations)
	}
}

func TestClientSearchMapsRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	client, err := NewClient(ClientConfig{
		HTTPClient: server.Client(),
		BaseURL:    server.URL,
		Tokens:     &rotatingTokenSource{token: "token"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Search(context.Background(), "raffles"); err != ErrRateLimited {
		t.Fatalf("error = %v, want ErrRateLimited", err)
	}
}
