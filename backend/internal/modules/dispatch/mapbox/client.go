// Package mapbox implements port.DistanceMatrixProvider using the Mapbox
// Distance Matrix API.
//
// The Mapbox API returns driving distances + durations between N origins
// and M destinations. We use it to estimate driver-to-pickup distances
// when creating dispatch offers.
//
// API: GET https://api.mapbox.com/directions-matrix/v1/mapbox/driving/{coordinates}
//   - coordinates: semicolon-separated lon,lat pairs (note: LON,LAT order!)
//   - sources: indices of origins (default: all)
//   - destinations: indices of destinations (default: all)
//   - annotations: "distance,duration"
//
// Response: {
//   "distances": [[meters, ...], ...],  // distances[i][j] = meters from origin i to dest j
//   "durations": [[seconds, ...], ...]
// }
//
// Imports: stdlib only — no third-party HTTP client.
package mapbox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"avex-backend/internal/modules/dispatch/port"
)

// Client implements port.DistanceMatrixProvider using Mapbox.
type Client struct {
	accessToken string
	httpClient  *http.Client
	baseURL     string
}

// New creates a new Mapbox client.
func New(accessToken string) *Client {
	return &Client{
		accessToken: accessToken,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		baseURL:     "https://api.mapbox.com/directions-matrix/v1/mapbox/driving",
	}
}

// matrixResponse is the JSON response from Mapbox.
type matrixResponse struct {
	Code       string      `json:"code"`
	Distances  [][]float64 `json:"distances"`  // meters
	Durations  [][]float64 `json:"durations"`  // seconds
	Message    string      `json:"message,omitempty"`
}

// GetDistanceMatrix returns driving distances + durations from each origin
// to each destination.
func (c *Client) GetDistanceMatrix(ctx context.Context, origins [][2]float64, destinations [][2]float64) ([][]int, [][]int, error) {
	if len(origins) == 0 || len(destinations) == 0 {
		return nil, nil, fmt.Errorf("origins and destinations must be non-empty")
	}
	if len(origins)+len(destinations) > 25 {
		// Mapbox limit: 25 total coordinates per request on free tier.
		return nil, nil, fmt.Errorf("too many coordinates: %d origins + %d destinations (max 25 total)", len(origins), len(destinations))
	}
	if c.accessToken == "" {
		return nil, nil, fmt.Errorf("mapbox access token is not set")
	}

	// Build coordinates string. Mapbox wants LON,LAT order (unusual).
	coords := make([]string, 0, len(origins)+len(destinations))
	for _, o := range origins {
		// o[0] = lat, o[1] = lng
		coords = append(coords, fmt.Sprintf("%f,%f", o[1], o[0]))
	}
	for _, d := range destinations {
		coords = append(coords, fmt.Sprintf("%f,%f", d[1], d[0]))
	}
	coordStr := strings.Join(coords, ";")

	// Build query params.
	params := url.Values{}
	params.Set("access_token", c.accessToken)
	params.Set("annotations", "distance,duration")
	// Sources: indices [0, len(origins))
	sources := make([]string, len(origins))
	for i := range origins {
		sources[i] = strconv.Itoa(i)
	}
	params.Set("sources", strings.Join(sources, ";"))
	// Destinations: indices [len(origins), len(origins)+len(destinations))
	dests := make([]string, len(destinations))
	for i := range destinations {
		dests[i] = strconv.Itoa(len(origins) + i)
	}
	params.Set("destinations", strings.Join(dests, ";"))

	requestURL := c.baseURL + "/" + coordStr + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("mapbox api error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var mr matrixResponse
	if err := json.Unmarshal(body, &mr); err != nil {
		return nil, nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if mr.Code != "Ok" {
		return nil, nil, fmt.Errorf("mapbox code %q: %s", mr.Code, mr.Message)
	}

	// Convert [][]float64 → [][]int
	distances := make([][]int, len(mr.Distances))
	for i, row := range mr.Distances {
		distances[i] = make([]int, len(row))
		for j, v := range row {
			distances[i][j] = int(v)
		}
	}
	durations := make([][]int, len(mr.Durations))
	for i, row := range mr.Durations {
		durations[i] = make([]int, len(row))
		for j, v := range row {
			durations[i][j] = int(v)
		}
	}

	return distances, durations, nil
}

// Compile-time assertion that Client implements DistanceMatrixProvider.
var _ port.DistanceMatrixProvider = (*Client)(nil)
