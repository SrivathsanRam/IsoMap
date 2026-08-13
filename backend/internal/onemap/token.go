package onemap

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const DefaultAuthURL = "https://www.onemap.gov.sg/api/auth/post/getToken"

const (
	defaultRefreshSkew  = 30 * time.Second
	fallbackRetryDelay  = time.Minute
	maxAuthResponseSize = 64 << 10
)

// TokenSource supplies the access token used by OneMap API calls. Invalidate
// allows a client to request one refresh and retry after OneMap rejects a token.
type TokenSource interface {
	Token(context.Context) (string, error)
	Invalidate(token string)
}

type TokenProviderConfig struct {
	HTTPClient  HTTPDoer
	AuthURL     string
	Email       string
	Password    string
	StaticToken string
	Now         func() time.Time
	RefreshSkew time.Duration
}

// TokenProvider caches OneMap's three-day access token. Its mutex is held while
// refreshing so concurrent searches cannot create a burst of authentication
// requests when a token expires.
type TokenProvider struct {
	mu sync.Mutex

	httpClient  HTTPDoer
	authURL     string
	email       string
	password    string
	staticToken string
	now         func() time.Time
	refreshSkew time.Duration

	cachedToken  string
	cachedExpiry time.Time
}

func NewTokenProvider(config TokenProviderConfig) *TokenProvider {
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: DefaultTimeout}
	}
	authURL := strings.TrimSpace(config.AuthURL)
	if authURL == "" {
		authURL = DefaultAuthURL
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}
	refreshSkew := config.RefreshSkew
	if refreshSkew <= 0 {
		refreshSkew = defaultRefreshSkew
	}

	return &TokenProvider{
		httpClient:  httpClient,
		authURL:     authURL,
		email:       strings.TrimSpace(config.Email),
		password:    config.Password,
		staticToken: strings.TrimSpace(config.StaticToken),
		now:         now,
		refreshSkew: refreshSkew,
	}
}

func (p *TokenProvider) Token(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := p.now()
	if p.cachedToken != "" && (p.cachedExpiry.IsZero() || now.Add(p.refreshSkew).Before(p.cachedExpiry)) {
		return p.cachedToken, nil
	}

	hasCredentials := p.email != "" && p.password != ""
	if hasCredentials {
		token, expiry, err := p.authenticate(ctx)
		if err == nil {
			p.cachedToken = token
			p.cachedExpiry = expiry
			return token, nil
		}
		if token, expiry, ok := p.usableStaticToken(now); ok {
			// Retry authentication soon rather than pinning a fallback token for
			// its full lifetime after a transient authentication failure.
			retryAt := now.Add(fallbackRetryDelay)
			if !expiry.IsZero() && expiry.Before(retryAt) {
				retryAt = expiry
			}
			p.cachedToken = token
			p.cachedExpiry = retryAt
			return token, nil
		}
		return "", err
	}

	if token, expiry, ok := p.usableStaticToken(now); ok {
		p.cachedToken = token
		p.cachedExpiry = expiry
		return token, nil
	}
	return "", ErrNotConfigured
}

func (p *TokenProvider) Invalidate(token string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if token == "" || token == p.cachedToken {
		p.cachedToken = ""
		p.cachedExpiry = time.Time{}
	}
}

func (p *TokenProvider) authenticate(ctx context.Context) (string, time.Time, error) {
	payload, err := json.Marshal(map[string]string{
		"email":    p.email,
		"password": p.password,
	})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("%w: encode credentials", ErrAuthentication)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.authURL, bytes.NewReader(payload))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("%w: create authentication request", ErrAuthentication)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	body, err := readLimited(resp.Body, maxAuthResponseSize)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("%w: read authentication response", ErrInvalidResponse)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return "", time.Time{}, ErrRateLimited
	}
	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, ErrAuthentication
	}

	var data struct {
		AccessToken     string          `json:"access_token"`
		ExpiryTimestamp json.RawMessage `json:"expiry_timestamp"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", time.Time{}, fmt.Errorf("%w: decode authentication response", ErrInvalidResponse)
	}
	token := strings.TrimSpace(data.AccessToken)
	expiryUnix, err := parseUnixTimestamp(data.ExpiryTimestamp)
	if token == "" || err != nil {
		return "", time.Time{}, ErrAuthentication
	}
	expiry := time.Unix(expiryUnix, 0)
	if !p.now().Add(p.refreshSkew).Before(expiry) {
		return "", time.Time{}, ErrAuthentication
	}
	return token, expiry, nil
}

func (p *TokenProvider) usableStaticToken(now time.Time) (string, time.Time, bool) {
	if p.staticToken == "" {
		return "", time.Time{}, false
	}
	expiry, hasExpiry := jwtExpiry(p.staticToken)
	if hasExpiry && !now.Add(p.refreshSkew).Before(expiry) {
		return "", time.Time{}, false
	}
	return p.staticToken, expiry, true
}

func parseUnixTimestamp(raw json.RawMessage) (int64, error) {
	value := strings.TrimSpace(string(raw))
	value = strings.Trim(value, `"`)
	if value == "" || value == "null" {
		return 0, fmt.Errorf("expiry timestamp is missing")
	}
	if integer, err := strconv.ParseInt(value, 10, 64); err == nil {
		return integer, nil
	}
	decimal, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}
	return int64(decimal), nil
}

func jwtExpiry(token string) (time.Time, bool) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, false
	}
	var claims struct {
		Expires json.Number `json:"exp"`
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	if err := decoder.Decode(&claims); err != nil || claims.Expires == "" {
		return time.Time{}, false
	}
	expires, err := claims.Expires.Int64()
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(expires, 0), true
}

func readLimited(reader io.Reader, maximum int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, maximum+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maximum {
		return nil, fmt.Errorf("response exceeds %d bytes", maximum)
	}
	return body, nil
}
