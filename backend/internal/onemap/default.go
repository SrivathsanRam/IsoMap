package onemap

import (
	"net/http"
	"os"
	"sync"
)

var (
	defaultTokensOnce sync.Once
	defaultTokens     TokenSource
)

// DefaultTokenSource returns the process-wide, lazily initialized token
// provider. Lazy initialization lets cmd/server load its environment file
// before credentials are read, while sharing one refreshed token across
// search and routing requests.
func DefaultTokenSource() TokenSource {
	defaultTokensOnce.Do(func() {
		defaultTokens = NewTokenProvider(TokenProviderConfig{
			HTTPClient:  &http.Client{Timeout: DefaultTimeout},
			Email:       os.Getenv("ONEMAP_API_EMAIL"),
			Password:    os.Getenv("ONEMAP_API_PASSWORD"),
			StaticToken: os.Getenv("ONEMAP_API_KEY"),
		})
	})
	return defaultTokens
}
