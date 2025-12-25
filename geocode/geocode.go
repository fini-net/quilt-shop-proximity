package geocode

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Coordinates represents GPS coordinates
type Coordinates struct {
	Latitude  float64
	Longitude float64
}

// Result represents the result of a geocoding operation
type Result struct {
	Coords *Coordinates
	Error  error
}

// nominatimResponse represents the JSON response from Nominatim API
type nominatimResponse struct {
	Lat string `json:"lat"`
	Lon string `json:"lon"`
}

// rateLimiter ensures we don't exceed 1 request per second to Nominatim
var rateLimiter = time.NewTicker(1 * time.Second)

// GeocodeAddress queries the Nominatim API to get GPS coordinates for an address
func GeocodeAddress(address string) Result {
	// Wait for rate limiter
	<-rateLimiter.C

	// URL encode the address
	encodedAddress := url.QueryEscape(address)
	apiURL := fmt.Sprintf("https://nominatim.openstreetmap.org/search?format=json&q=%s&limit=1", encodedAddress)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create request with required User-Agent header
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return Result{Error: fmt.Errorf("failed to create request: %w", err)}
	}
	req.Header.Set("User-Agent", "quilt-shop-proximity/1.0 (github.com/chicks-net/quilt-shop-proximity)")

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return Result{Error: fmt.Errorf("HTTP request failed: %w", err)}
	}
	defer resp.Body.Close()

	// Handle HTTP errors
	if resp.StatusCode == 429 {
		return Result{Error: fmt.Errorf("rate limited by Nominatim (HTTP 429)")}
	}
	if resp.StatusCode != 200 {
		return Result{Error: fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)}
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{Error: fmt.Errorf("failed to read response: %w", err)}
	}

	// Parse JSON response
	var results []nominatimResponse
	if err := json.Unmarshal(body, &results); err != nil {
		return Result{Error: fmt.Errorf("failed to parse JSON: %w", err)}
	}

	// Check if we got any results
	if len(results) == 0 {
		return Result{Error: fmt.Errorf("no results found for address")}
	}

	// Parse latitude and longitude from strings
	var lat, lon float64
	if _, err := fmt.Sscanf(results[0].Lat, "%f", &lat); err != nil {
		return Result{Error: fmt.Errorf("failed to parse latitude: %w", err)}
	}
	if _, err := fmt.Sscanf(results[0].Lon, "%f", &lon); err != nil {
		return Result{Error: fmt.Errorf("failed to parse longitude: %w", err)}
	}

	return Result{
		Coords: &Coordinates{
			Latitude:  lat,
			Longitude: lon,
		},
	}
}
