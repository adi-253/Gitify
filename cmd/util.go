package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// SpotifyClient handles all Spotify API requests
type SpotifyClient struct {
	HTTPClient *http.Client
	Token      *SpotfiyToken
}

// NewSpotifyClient loads the saved token and initializes a client
func NewSpotifyClient() (*SpotifyClient, error) {
	data, err := os.ReadFile("token.json")
	if err != nil {
		return nil, fmt.Errorf("token.json not found, please login first: %v", err)
	}

	var token SpotfiyToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token.json: %v", err)
	}

	return &SpotifyClient{
		HTTPClient: &http.Client{},
		Token:      &token,
	}, nil
}

// makeRequest is an internal helper that automatically refreshes the token if expired
func (s *SpotifyClient) makeRequest(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+s.Token.AccessToken)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	// If access token expired
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		fmt.Println("Access token expired. Refreshing...")

		// Call your existing RefreshToken() function
		if err := RefreshToken(); err != nil {
			return nil, fmt.Errorf("failed to refresh token: %v", err)
		}

		// Reload updated token
		updated, err := os.ReadFile("token.json")
		if err != nil {
			return nil, err
		}
		var newToken SpotfiyToken
		if err := json.Unmarshal(updated, &newToken); err != nil {
			return nil, err
		}
		s.Token = &newToken

		// Retry the same request once
		req.Header.Set("Authorization", "Bearer "+s.Token.AccessToken)
		return s.HTTPClient.Do(req)
	}

	return resp, nil
}

// Public helper for GET requests
func (s *SpotifyClient) Get(url string) (*http.Response, error) {
	return s.makeRequest(http.MethodGet, url, nil)
}

// Public helper for POST requests (optional, for later)
func (s *SpotifyClient) Post(url string, body io.Reader) (*http.Response, error) {
	return s.makeRequest(http.MethodPost, url, body)
}
