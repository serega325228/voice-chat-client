package container

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	client "voice-chat-client/internal/clients"
	service "voice-chat-client/internal/services"
	storage "voice-chat-client/internal/storages"
)

const (
	defaultAPIBaseURL  = "http://localhost:8080"
	defaultKeyringName = "voice-chat-client"
)

type Config struct {
	APIBaseURL       string
	WebSocketBaseURL string
	KeyringService   string
}

func DefaultConfig() Config {
	return Config{
		APIBaseURL:       getenv("VOICE_CHAT_API_BASE_URL", defaultAPIBaseURL),
		WebSocketBaseURL: strings.TrimSpace(os.Getenv("VOICE_CHAT_WS_BASE_URL")),
		KeyringService:   getenv("VOICE_CHAT_KEYRING_SERVICE", defaultKeyringName),
	}
}

type Container struct {
	config Config

	mu sync.Mutex

	httpClient       *http.Client
	tokenStorage     *storage.TokenStorage
	authClient       *client.AuthClient
	authService      *service.AuthService
	signalingClient  *client.SignalingClient
	signalingService *service.SignalingService
}

func New(config Config) *Container {
	config = normalizeConfig(config)

	return &Container{
		config: config,
	}
}

func (c *Container) TokenStorage() *storage.TokenStorage {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.tokenStorage == nil {
		c.tokenStorage = storage.NewTokenStorage(c.config.KeyringService)
	}

	return c.tokenStorage
}

func (c *Container) HTTPClient() *http.Client {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	return c.httpClient
}

func (c *Container) AuthClient() (*client.AuthClient, error) {
	c.mu.Lock()
	if c.authClient != nil {
		authClient := c.authClient
		c.mu.Unlock()
		return authClient, nil
	}
	c.mu.Unlock()

	config := c.clientConfig()
	authClient, err := client.NewAuthClient(config, client.DefaultAuthRoutes())
	if err != nil {
		return nil, fmt.Errorf("container: init auth client: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.authClient == nil {
		c.authClient = authClient
	}

	return c.authClient, nil
}

func (c *Container) AuthService() (*service.AuthService, error) {
	c.mu.Lock()
	if c.authService != nil {
		authService := c.authService
		c.mu.Unlock()
		return authService, nil
	}
	c.mu.Unlock()

	authClient, err := c.AuthClient()
	if err != nil {
		return nil, err
	}

	authService := service.NewAuthService(authClient, c.TokenStorage())

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.authService == nil {
		c.authService = authService
	}

	return c.authService, nil
}

func (c *Container) SignalingClient() (*client.SignalingClient, error) {
	c.mu.Lock()
	if c.signalingClient != nil {
		signalingClient := c.signalingClient
		c.mu.Unlock()
		return signalingClient, nil
	}
	c.mu.Unlock()

	config := c.clientConfig()
	signalingClient, err := client.NewSignalingClient(config, client.DefaultSignalingRoutes())
	if err != nil {
		return nil, fmt.Errorf("container: init signaling client: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.signalingClient == nil {
		c.signalingClient = signalingClient
	}

	return c.signalingClient, nil
}

func (c *Container) SignalingService() (*service.SignalingService, error) {
	c.mu.Lock()
	if c.signalingService != nil {
		signalingService := c.signalingService
		c.mu.Unlock()
		return signalingService, nil
	}
	c.mu.Unlock()

	signalingClient, err := c.SignalingClient()
	if err != nil {
		return nil, err
	}

	signalingService := service.NewSignalingService(signalingClient, nil)

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.signalingService == nil {
		c.signalingService = signalingService
	}

	return c.signalingService, nil
}

func (c *Container) clientConfig() client.ClientConfig {
	return client.ClientConfig{
		BaseURL:          c.config.APIBaseURL,
		WebSocketBaseURL: c.config.WebSocketBaseURL,
		HTTPClient:       c.HTTPClient(),
		AccessTokenProvider: func() string {
			return c.TokenStorage().GetAccess()
		},
	}
}

func normalizeConfig(config Config) Config {
	defaults := DefaultConfig()

	config.APIBaseURL = strings.TrimSpace(config.APIBaseURL)
	if config.APIBaseURL == "" {
		config.APIBaseURL = defaults.APIBaseURL
	}

	config.WebSocketBaseURL = strings.TrimSpace(config.WebSocketBaseURL)

	config.KeyringService = strings.TrimSpace(config.KeyringService)
	if config.KeyringService == "" {
		config.KeyringService = defaults.KeyringService
	}

	return config
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}
