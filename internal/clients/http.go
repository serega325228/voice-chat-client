package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type AccessTokenProvider func() string

type ClientConfig struct {
	BaseURL             string
	WebSocketBaseURL    string
	HTTPClient          *http.Client
	AccessTokenProvider AccessTokenProvider
}

type apiClient struct {
	baseURL             string
	websocketBaseURL    string
	httpClient          *http.Client
	accessTokenProvider AccessTokenProvider
}

func newAPIClient(config ClientConfig) (*apiClient, error) {
	baseURL := strings.TrimRight(config.BaseURL, "/")
	if baseURL == "" {
		return nil, fmt.Errorf("client: base URL is required")
	}

	websocketBaseURL := strings.TrimRight(config.WebSocketBaseURL, "/")
	if websocketBaseURL == "" {
		derived, err := deriveWebSocketBaseURL(baseURL)
		if err != nil {
			return nil, err
		}
		websocketBaseURL = derived
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	return &apiClient{
		baseURL:             baseURL,
		websocketBaseURL:    websocketBaseURL,
		httpClient:          httpClient,
		accessTokenProvider: config.AccessTokenProvider,
	}, nil
}

func (c *apiClient) joinURL(path string) (string, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}

	relative, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	return base.ResolveReference(relative).String(), nil
}

func (c *apiClient) joinWebSocketURL(path string) (string, error) {
	base, err := url.Parse(c.websocketBaseURL)
	if err != nil {
		return "", err
	}

	relative, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	return base.ResolveReference(relative).String(), nil
}

func (c *apiClient) newJSONRequest(method, requestURL string, body any, authorized bool) (*http.Request, error) {
	var requestBody io.Reader

	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		requestBody = bytes.NewReader(payload)
	}

	request, err := http.NewRequest(method, requestURL, requestBody)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	if authorized && c.accessTokenProvider != nil {
		if token := strings.TrimSpace(c.accessTokenProvider()); token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
	}

	return request, nil
}

func (c *apiClient) doJSON(request *http.Request, out any) error {
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		message, readErr := io.ReadAll(io.LimitReader(response.Body, 4096))
		if readErr != nil {
			return fmt.Errorf("client: status %d", response.StatusCode)
		}

		if trimmed := strings.TrimSpace(string(message)); trimmed != "" {
			return fmt.Errorf("client: status %d: %s", response.StatusCode, trimmed)
		}

		return fmt.Errorf("client: status %d", response.StatusCode)
	}

	if out == nil {
		io.Copy(io.Discard, response.Body)
		return nil
	}

	if err := json.NewDecoder(response.Body).Decode(out); err != nil && err != io.EOF {
		return err
	}

	return nil
}

func deriveWebSocketBaseURL(baseURL string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	switch parsed.Scheme {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("client: unsupported base URL scheme %q", parsed.Scheme)
	}

	return strings.TrimRight(parsed.String(), "/"), nil
}
