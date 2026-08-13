package onemap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultBaseURL        = "https://www.onemap.gov.sg"
	DefaultTimeout        = 5 * time.Second
	maxSearchResponseSize = 2 << 20
)

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type ClientConfig struct {
	HTTPClient HTTPDoer
	BaseURL    string
	Tokens     TokenSource
}

type Client struct {
	httpClient HTTPDoer
	searchURL  string
	tokens     TokenSource
}

type SearchResult struct {
	SearchValue string `json:"SEARCHVAL"`
	BlockNumber string `json:"BLK_NO"`
	RoadName    string `json:"ROAD_NAME"`
	Building    string `json:"BUILDING"`
	Address     string `json:"ADDRESS"`
	PostalCode  string `json:"POSTAL"`
	Latitude    string `json:"LATITUDE"`
	Longitude   string `json:"LONGITUDE"`
	Longtitude  string `json:"LONGTITUDE"`
}

func NewClient(config ClientConfig) (*Client, error) {
	if config.Tokens == nil {
		return nil, ErrNotConfigured
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: DefaultTimeout}
	}
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid OneMap base URL")
	}

	return &Client{
		httpClient: httpClient,
		searchURL:  baseURL + "/api/common/elastic/search",
		tokens:     config.Tokens,
	}, nil
}

func (c *Client) Search(ctx context.Context, query string) ([]SearchResult, error) {
	for attempt := 0; attempt < 2; attempt++ {
		token, err := c.tokens.Token(ctx)
		if err != nil {
			return nil, err
		}
		results, err := c.searchOnce(ctx, query, token)
		if err == nil {
			return results, nil
		}
		if attempt == 0 && isAuthenticationError(err) {
			c.tokens.Invalidate(token)
			continue
		}
		return nil, err
	}
	return nil, ErrAuthentication
}

func (c *Client) searchOnce(ctx context.Context, query string, token string) ([]SearchResult, error) {
	endpoint, err := url.Parse(c.searchURL)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid search URL", ErrUnavailable)
	}
	parameters := endpoint.Query()
	parameters.Set("searchVal", query)
	parameters.Set("returnGeom", "Y")
	parameters.Set("getAddrDetails", "Y")
	parameters.Set("pageNum", "1")
	endpoint.RawQuery = parameters.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: create search request", ErrUnavailable)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	body, err := readLimited(resp.Body, maxSearchResponseSize)
	if err != nil {
		return nil, fmt.Errorf("%w: read search response", ErrInvalidResponse)
	}
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, ErrAuthentication
	case http.StatusTooManyRequests:
		return nil, ErrRateLimited
	default:
		return nil, fmt.Errorf("%w: HTTP %d", ErrUnavailable, resp.StatusCode)
	}

	var data struct {
		Error   string         `json:"error"`
		Results []SearchResult `json:"results"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("%w: decode search response", ErrInvalidResponse)
	}
	if message := strings.ToLower(strings.TrimSpace(data.Error)); message != "" {
		switch {
		case strings.Contains(message, "token"), strings.Contains(message, "auth"):
			return nil, ErrAuthentication
		case strings.Contains(message, "limit"), strings.Contains(message, "rate"):
			return nil, ErrRateLimited
		default:
			return nil, ErrUnavailable
		}
	}
	if data.Results == nil {
		data.Results = []SearchResult{}
	}
	return data.Results, nil
}

func isAuthenticationError(err error) bool {
	return errors.Is(err, ErrAuthentication)
}
