package container

import (
	"fmt"
	"net/http"
	client "selfcord/internal/clients"
	service "selfcord/internal/services"
	storage "selfcord/internal/storages"
	"strings"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
)

const (
	defaultAPIBaseURL  = "http://localhost:8082"
	defaultKeyringName = "selfcord"
)

type Config struct {
	APIBaseURL       string
	WebSocketBaseURL string
	KeyringService   string
	WebRTCICEServers []ICEServer
}

type ICEServer struct {
	URLs       []string
	Username   string
	Credential string
}

func DefaultConfig() Config {
	return Config{
		APIBaseURL:     defaultAPIBaseURL,
		KeyringService: defaultKeyringName,
	}
}

type Container struct {
	config Config

	httpClientOnce       sync.Once
	httpClient           *http.Client
	tokenStorageOnce     sync.Once
	tokenStorage         *storage.TokenStorage
	authClientOnce       sync.Once
	authClient           *client.AuthClient
	authClientErr        error
	authServiceOnce      sync.Once
	authService          *service.AuthService
	authServiceErr       error
	signalingClientOnce  sync.Once
	signalingClient      *client.SignalingClient
	signalingClientErr   error
	signalingServiceOnce sync.Once
	signalingService     *service.SignalingService
	signalingServiceErr  error
}

func New(config Config) *Container {
	config = normalizeConfig(config)

	return &Container{
		config: config,
	}
}

func (c *Container) TokenStorage() *storage.TokenStorage {
	c.tokenStorageOnce.Do(func() {
		c.tokenStorage = storage.NewTokenStorage(c.config.KeyringService)
	})

	return c.tokenStorage
}

func (c *Container) HTTPClient() *http.Client {
	c.httpClientOnce.Do(func() {
		c.httpClient = &http.Client{Timeout: 10 * time.Second}
	})

	return c.httpClient
}

func (c *Container) AuthClient() (*client.AuthClient, error) {
	c.authClientOnce.Do(func() {
		authClient, err := client.NewAuthClient(c.clientConfig(), client.DefaultAuthRoutes())
		if err != nil {
			c.authClientErr = fmt.Errorf("container: init auth client: %w", err)
			return
		}

		c.authClient = authClient
	})

	return c.authClient, c.authClientErr
}

func (c *Container) AuthService() (*service.AuthService, error) {
	c.authServiceOnce.Do(func() {
		authClient, err := c.AuthClient()
		if err != nil {
			c.authServiceErr = err
			return
		}

		c.authService = service.NewAuthService(authClient, c.TokenStorage())
	})

	return c.authService, c.authServiceErr
}

func (c *Container) SignalingClient() (*client.SignalingClient, error) {
	c.signalingClientOnce.Do(func() {
		signalingClient, err := client.NewSignalingClient(c.clientConfig(), client.DefaultSignalingRoutes())
		if err != nil {
			c.signalingClientErr = fmt.Errorf("container: init signaling client: %w", err)
			return
		}

		c.signalingClient = signalingClient
	})

	return c.signalingClient, c.signalingClientErr
}

func (c *Container) SignalingService() (*service.SignalingService, error) {
	c.signalingServiceOnce.Do(func() {
		signalingClient, err := c.SignalingClient()
		if err != nil {
			c.signalingServiceErr = err
			return
		}

		c.signalingService = service.NewSignalingService(signalingClient, func() (service.WebRTCPeer, error) {
			return service.NewManagedSessionPeer(service.WebRTCConfig{
				ICEServers: c.webRTCICEServers(),
			}, service.DefaultSoundConfig())
		})
	})

	return c.signalingService, c.signalingServiceErr
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

func (c *Container) webRTCICEServers() []webrtc.ICEServer {
	servers := make([]webrtc.ICEServer, 0, len(c.config.WebRTCICEServers))
	for _, server := range c.config.WebRTCICEServers {
		if len(server.URLs) == 0 {
			continue
		}
		servers = append(servers, webrtc.ICEServer{
			URLs:       append([]string(nil), server.URLs...),
			Username:   server.Username,
			Credential: server.Credential,
		})
	}

	return servers
}
