package onemap

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenProviderAuthenticatesAndCachesAcrossConcurrentCalls(t *testing.T) {
	now := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		var credentials map[string]string
		if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
			t.Error(err)
		}
		if credentials["email"] != "developer@example.com" || credentials["password"] != "password" {
			t.Errorf("credentials = %+v", credentials)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":     "fresh-token",
			"expiry_timestamp": fmt.Sprint(now.Add(72 * time.Hour).Unix()),
		})
	}))
	defer server.Close()

	provider := NewTokenProvider(TokenProviderConfig{
		HTTPClient: server.Client(),
		AuthURL:    server.URL,
		Email:      "developer@example.com",
		Password:   "password",
		Now:        func() time.Time { return now },
	})

	const workers = 20
	var wait sync.WaitGroup
	errorsChannel := make(chan error, workers)
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			token, err := provider.Token(context.Background())
			if err != nil {
				errorsChannel <- err
				return
			}
			if token != "fresh-token" {
				errorsChannel <- fmt.Errorf("token = %q", token)
			}
		}()
	}
	wait.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		t.Error(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("authentication calls = %d, want 1", calls.Load())
	}

	provider.Invalidate("fresh-token")
	if _, err := provider.Token(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("authentication calls after invalidation = %d, want 2", calls.Load())
	}
}

func TestTokenProviderUsesStaticFallbackAndRejectsExpiredJWT(t *testing.T) {
	now := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	valid := testJWT(t, now.Add(time.Hour).Unix())
	provider := NewTokenProvider(TokenProviderConfig{
		StaticToken: valid,
		Now:         func() time.Time { return now },
	})
	token, err := provider.Token(context.Background())
	if err != nil || token != valid {
		t.Fatalf("Token() = %q, %v", token, err)
	}

	expired := NewTokenProvider(TokenProviderConfig{
		StaticToken: testJWT(t, now.Add(-time.Hour).Unix()),
		Now:         func() time.Time { return now },
	})
	if _, err := expired.Token(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expired Token() error = %v, want ErrNotConfigured", err)
	}
}

func TestTokenProviderFallsBackWhenAuthenticationIsUnavailable(t *testing.T) {
	now := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	static := testJWT(t, now.Add(time.Hour).Unix())
	provider := NewTokenProvider(TokenProviderConfig{
		HTTPClient:  server.Client(),
		AuthURL:     server.URL,
		Email:       "developer@example.com",
		Password:    "password",
		StaticToken: static,
		Now:         func() time.Time { return now },
	})
	token, err := provider.Token(context.Background())
	if err != nil || token != static {
		t.Fatalf("Token() = %q, %v; want static fallback", token, err)
	}
}

func testJWT(t *testing.T, expiry int64) string {
	t.Helper()
	payload, err := json.Marshal(map[string]int64{"exp": expiry})
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join([]string{
		base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`)),
		base64.RawURLEncoding.EncodeToString(payload),
		"signature",
	}, ".")
}
