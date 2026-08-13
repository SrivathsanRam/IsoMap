package places

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/SrivathsanRam/IsoMap/internal/onemap"
)

type fakeSearchProvider struct {
	mu      sync.Mutex
	calls   int
	queries []string
	search  func(context.Context, string) ([]onemap.SearchResult, error)
}

func (p *fakeSearchProvider) Search(ctx context.Context, query string) ([]onemap.SearchResult, error) {
	p.mu.Lock()
	p.calls++
	p.queries = append(p.queries, query)
	p.mu.Unlock()
	return p.search(ctx, query)
}

func TestHandlerValidatesSearchParameters(t *testing.T) {
	provider := &fakeSearchProvider{search: func(context.Context, string) ([]onemap.SearchResult, error) {
		t.Fatal("provider should not be called")
		return nil, nil
	}}
	handler := New(Config{Provider: provider})

	for _, test := range []struct {
		name   string
		target string
	}{
		{name: "missing query", target: "/places/search"},
		{name: "short query", target: "/places/search?q=a"},
		{name: "long query", target: "/places/search?q=" + strings.Repeat("a", 257)},
		{name: "zero limit", target: "/places/search?q=raffles&limit=0"},
		{name: "large limit", target: "/places/search?q=raffles&limit=11"},
		{name: "invalid limit", target: "/places/search?q=raffles&limit=many"},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := serve(handler, http.MethodGet, test.target)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestHandlerNormalizesFiltersAndLimitsOneMapResults(t *testing.T) {
	valid := onemap.SearchResult{
		SearchValue: "REVENUE HOUSE",
		BlockNumber: "55",
		RoadName:    "NEWTON ROAD",
		Building:    "REVENUE HOUSE",
		Address:     "55 NEWTON ROAD REVENUE HOUSE SINGAPORE 307987",
		PostalCode:  "307987",
		Latitude:    "1.31955940176018",
		Longitude:   "103.842624736208",
	}
	provider := &fakeSearchProvider{search: func(_ context.Context, query string) ([]onemap.SearchResult, error) {
		if query != "Revenue House" {
			t.Fatalf("query = %q, want normalized query", query)
		}
		return []onemap.SearchResult{
			valid,
			valid, // duplicate
			{
				SearchValue: "640 ROWELL ROAD",
				BlockNumber: "640",
				RoadName:    "ROWELL ROAD",
				Building:    "NIL",
				Address:     "640 ROWELL ROAD SINGAPORE 200640",
				PostalCode:  "200640",
				Latitude:    "1.30743547948389",
				Longtitude:  "103.854713903431",
			},
			{
				SearchValue: "OUTSIDE SINGAPORE",
				Address:     "OUTSIDE SINGAPORE",
				Latitude:    "2.0",
				Longitude:   "103.8",
			},
		}, nil
	}}
	handler := New(Config{Provider: provider})
	response := serve(handler, http.MethodGet, "/places/search?q=%20Revenue%20%20House%20&limit=2")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", response.Code, response.Body.String())
	}

	var body searchResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Places) != 2 {
		t.Fatalf("places = %d, want 2: %+v", len(body.Places), body.Places)
	}
	first := body.Places[0]
	if first.ID == "" || !strings.HasPrefix(first.ID, "onemap:") {
		t.Fatalf("id = %q", first.ID)
	}
	if first.DisplayName != valid.Address || first.PrimaryText != valid.Building {
		t.Fatalf("first place = %+v", first)
	}
	if first.SecondaryText != valid.Address || first.PostalCode != "307987" || first.Source != "onemap" {
		t.Fatalf("first metadata = %+v", first)
	}
	second := body.Places[1]
	if second.PrimaryText != "640 ROWELL ROAD" || second.Longitude != "103.854713903431" {
		t.Fatalf("second place = %+v", second)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
}

func TestHandlerCachesNormalizedRepeatedQueries(t *testing.T) {
	now := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	provider := &fakeSearchProvider{search: func(context.Context, string) ([]onemap.SearchResult, error) {
		return []onemap.SearchResult{{
			SearchValue: "RAFFLES PLACE",
			Address:     "RAFFLES PLACE SINGAPORE",
			Latitude:    "1.284",
			Longitude:   "103.851",
		}}, nil
	}}
	handler := New(Config{
		Provider: provider,
		CacheTTL: time.Minute,
		Now:      func() time.Time { return now },
	})

	for _, target := range []string{
		"/places/search?q=Raffles%20%20Place&limit=1",
		"/places/search?q=%20raffles%20place%20&limit=10",
	} {
		if response := serve(handler, http.MethodGet, target); response.Code != http.StatusOK {
			t.Fatalf("status = %d; body=%s", response.Code, response.Body.String())
		}
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.calls)
	}

	now = now.Add(time.Minute + time.Second)
	serve(handler, http.MethodGet, "/places/search?q=raffles%20place")
	if provider.calls != 2 {
		t.Fatalf("provider calls after expiry = %d, want 2", provider.calls)
	}
}

func TestHandlerMapsProviderFailuresToSafeErrors(t *testing.T) {
	for _, test := range []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "not configured", err: onemap.ErrNotConfigured, wantStatus: http.StatusServiceUnavailable},
		{name: "authentication", err: onemap.ErrAuthentication, wantStatus: http.StatusServiceUnavailable},
		{name: "rate limited", err: onemap.ErrRateLimited, wantStatus: http.StatusTooManyRequests},
		{name: "bad response", err: onemap.ErrInvalidResponse, wantStatus: http.StatusBadGateway},
		{name: "unavailable", err: errors.New("secret upstream detail"), wantStatus: http.StatusBadGateway},
	} {
		t.Run(test.name, func(t *testing.T) {
			provider := &fakeSearchProvider{search: func(context.Context, string) ([]onemap.SearchResult, error) {
				return nil, test.err
			}}
			response := serve(New(Config{Provider: provider}), http.MethodGet, "/places/search?q=raffles")
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if strings.Contains(response.Body.String(), "secret upstream detail") {
				t.Fatalf("response leaked provider error: %s", response.Body.String())
			}
		})
	}
}

func TestHandlerTimesOutSlowProvider(t *testing.T) {
	provider := &fakeSearchProvider{search: func(ctx context.Context, _ string) ([]onemap.SearchResult, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	response := serve(New(Config{
		Provider:       provider,
		RequestTimeout: 5 * time.Millisecond,
	}), http.MethodGet, "/places/search?q=raffles")
	if response.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want 504; body=%s", response.Code, response.Body.String())
	}
}

func serve(handler http.Handler, method, target string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
