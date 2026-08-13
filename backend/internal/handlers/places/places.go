package places

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/SrivathsanRam/IsoMap/internal/onemap"
)

const (
	defaultLimit          = 8
	defaultCacheTTL       = 5 * time.Minute
	defaultRequestTimeout = 5 * time.Second
	maxCacheEntries       = 512

	// Bounds cover Singapore's populated islands and the walking graph's scope.
	minSingaporeLatitude  = 1.1304
	maxSingaporeLatitude  = 1.4780
	minSingaporeLongitude = 103.5935
	maxSingaporeLongitude = 104.1076
)

type searchProvider interface {
	Search(context.Context, string) ([]onemap.SearchResult, error)
}

type Config struct {
	Provider       searchProvider
	CacheTTL       time.Duration
	RequestTimeout time.Duration
	Now            func() time.Time
}

type Handler struct {
	provider       searchProvider
	cacheTTL       time.Duration
	requestTimeout time.Duration
	now            func() time.Time

	cacheMu sync.Mutex
	cache   map[string]cacheEntry
}

type cacheEntry struct {
	expires time.Time
	places  []Place
}

type Place struct {
	ID            string `json:"id"`
	DisplayName   string `json:"display_name"`
	PrimaryText   string `json:"primary_text"`
	SecondaryText string `json:"secondary_text"`
	Latitude      string `json:"lat"`
	Longitude     string `json:"lon"`
	PostalCode    string `json:"postal_code,omitempty"`
	Source        string `json:"source"`
}

type searchResponse struct {
	Places []Place `json:"places"`
}

func New(config Config) *Handler {
	cacheTTL := config.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = defaultCacheTTL
	}
	requestTimeout := config.RequestTimeout
	if requestTimeout <= 0 {
		requestTimeout = defaultRequestTimeout
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}
	return &Handler{
		provider:       config.Provider,
		cacheTTL:       cacheTTL,
		requestTimeout: requestTimeout,
		now:            now,
		cache:          make(map[string]cacheEntry),
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		return
	}

	query, err := validateQuery(r.URL.Query().Get("q"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, err := validateLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if h.provider == nil {
		writeError(w, http.StatusServiceUnavailable, "Place search is not configured.")
		return
	}

	cacheKey := strings.ToLower(query)
	if cached, ok := h.cached(cacheKey); ok {
		writePlaces(w, limited(cached, limit))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.requestTimeout)
	defer cancel()
	results, err := h.provider.Search(ctx, query)
	if err != nil {
		h.writeProviderError(w, ctx, err)
		return
	}

	places := normalizeResults(results)
	h.store(cacheKey, places)
	writePlaces(w, limited(places, limit))
}

func (h *Handler) writeProviderError(w http.ResponseWriter, ctx context.Context, err error) {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded), errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusGatewayTimeout, "Place search timed out. Please try again.")
	case errors.Is(err, onemap.ErrRateLimited):
		w.Header().Set("Retry-After", "30")
		writeError(w, http.StatusTooManyRequests, "Place search is rate limited. Please try again shortly.")
	case errors.Is(err, onemap.ErrNotConfigured):
		writeError(w, http.StatusServiceUnavailable, "Place search is not configured.")
	case errors.Is(err, onemap.ErrAuthentication):
		writeError(w, http.StatusServiceUnavailable, "Place search authentication is unavailable.")
	default:
		writeError(w, http.StatusBadGateway, "Place search is temporarily unavailable.")
	}
}

func validateQuery(raw string) (string, error) {
	query := strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	length := utf8.RuneCountInString(query)
	if length < 2 || length > 256 {
		return "", errors.New("q must be between 2 and 256 characters")
	}
	if strings.IndexFunc(query, unicode.IsControl) >= 0 {
		return "", errors.New("q contains invalid control characters")
	}
	return query, nil
}

func validateLimit(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return defaultLimit, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 || limit > 10 {
		return 0, errors.New("limit must be between 1 and 10")
	}
	return limit, nil
}

func normalizeResults(results []onemap.SearchResult) []Place {
	places := make([]Place, 0, len(results))
	seen := make(map[string]struct{}, len(results))
	for _, result := range results {
		place, ok := normalizeResult(result)
		if !ok {
			continue
		}
		if _, duplicate := seen[place.ID]; duplicate {
			continue
		}
		seen[place.ID] = struct{}{}
		places = append(places, place)
	}
	return places
}

func normalizeResult(result onemap.SearchResult) (Place, bool) {
	latitude, err := strconv.ParseFloat(strings.TrimSpace(result.Latitude), 64)
	if err != nil || math.IsNaN(latitude) || math.IsInf(latitude, 0) ||
		latitude < minSingaporeLatitude || latitude > maxSingaporeLatitude {
		return Place{}, false
	}
	longitudeText := strings.TrimSpace(result.Longitude)
	if longitudeText == "" {
		longitudeText = strings.TrimSpace(result.Longtitude)
	}
	longitude, err := strconv.ParseFloat(longitudeText, 64)
	if err != nil || math.IsNaN(longitude) || math.IsInf(longitude, 0) ||
		longitude < minSingaporeLongitude || longitude > maxSingaporeLongitude {
		return Place{}, false
	}

	building := meaningful(result.Building)
	block := meaningful(result.BlockNumber)
	road := meaningful(result.RoadName)
	searchValue := meaningful(result.SearchValue)
	address := meaningful(result.Address)
	postalCode := meaningful(result.PostalCode)

	primary := building
	if primary == "" {
		primary = strings.TrimSpace(strings.Join(nonEmpty(block, road), " "))
	}
	if primary == "" {
		primary = firstNonEmpty(searchValue, address)
	}
	displayName := firstNonEmpty(address, searchValue, primary)
	if displayName == "" || primary == "" {
		return Place{}, false
	}
	secondary := displayName
	if strings.EqualFold(primary, displayName) {
		secondary = ""
	}

	latText := strconv.FormatFloat(latitude, 'f', -1, 64)
	lonText := strconv.FormatFloat(longitude, 'f', -1, 64)
	digest := sha256.Sum256([]byte(strings.ToLower(displayName) + "\x00" + latText + "\x00" + lonText))
	return Place{
		ID:            "onemap:" + hex.EncodeToString(digest[:12]),
		DisplayName:   displayName,
		PrimaryText:   primary,
		SecondaryText: secondary,
		Latitude:      latText,
		Longitude:     lonText,
		PostalCode:    postalCode,
		Source:        "onemap",
	}, true
}

func meaningful(value string) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	switch strings.ToUpper(value) {
	case "", "NIL", "NULL", "N/A", "NA", "-":
		return ""
	default:
		return value
	}
}

func nonEmpty(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (h *Handler) cached(key string) ([]Place, bool) {
	h.cacheMu.Lock()
	defer h.cacheMu.Unlock()
	entry, ok := h.cache[key]
	if !ok {
		return nil, false
	}
	if !h.now().Before(entry.expires) {
		delete(h.cache, key)
		return nil, false
	}
	return append([]Place(nil), entry.places...), true
}

func (h *Handler) store(key string, places []Place) {
	h.cacheMu.Lock()
	defer h.cacheMu.Unlock()
	now := h.now()
	if len(h.cache) >= maxCacheEntries {
		var oldestKey string
		var oldestExpiry time.Time
		for cachedKey, entry := range h.cache {
			if !now.Before(entry.expires) {
				delete(h.cache, cachedKey)
				continue
			}
			if oldestKey == "" || entry.expires.Before(oldestExpiry) {
				oldestKey = cachedKey
				oldestExpiry = entry.expires
			}
		}
		if len(h.cache) >= maxCacheEntries && oldestKey != "" {
			delete(h.cache, oldestKey)
		}
	}
	h.cache[key] = cacheEntry{
		expires: now.Add(h.cacheTTL),
		places:  append([]Place(nil), places...),
	}
}

func limited(places []Place, limit int) []Place {
	if len(places) > limit {
		places = places[:limit]
	}
	return append([]Place(nil), places...)
}

func writePlaces(w http.ResponseWriter, places []Place) {
	if places == nil {
		places = []Place{}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(searchResponse{Places: places})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": message})
}

var (
	defaultHandlerOnce sync.Once
	defaultHandler     http.Handler
)

// HandleSearch lazily constructs the environment-backed handler. Lazy setup is
// required because cmd/server loads its .env file after package initialization.
func HandleSearch(w http.ResponseWriter, r *http.Request) {
	defaultHandlerOnce.Do(func() {
		httpClient := &http.Client{Timeout: onemap.DefaultTimeout}
		client, err := onemap.NewClient(onemap.ClientConfig{
			HTTPClient: httpClient,
			Tokens:     onemap.DefaultTokenSource(),
		})
		if err != nil {
			defaultHandler = New(Config{})
			return
		}
		defaultHandler = New(Config{Provider: client})
	})
	defaultHandler.ServeHTTP(w, r)
}
