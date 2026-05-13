package config

import (
	"fmt"
	"os"
	"strings"

	containerpkg "selfcord/internal/container"

	"gopkg.in/yaml.v3"
)

const defaultConfigPath = "config/local.yaml"

type Config struct {
	Env     string  `yaml:"env"`
	API     API     `yaml:"api"`
	Keyring Keyring `yaml:"keyring"`
	WebRTC  WebRTC  `yaml:"webrtc"`
}

type API struct {
	BaseURL          string `yaml:"base_url"`
	WebSocketBaseURL string `yaml:"websocket_base_url"`
}

type Keyring struct {
	Service string `yaml:"service"`
}

type WebRTC struct {
	ICEServers []ICEServer `yaml:"ice_servers"`
}

type ICEServer struct {
	URLs       []string `yaml:"urls"`
	Username   string   `yaml:"username"`
	Credential string   `yaml:"credential"`
}

func MustLoad() Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}

	return cfg
}

func Load() (Config, error) {
	cfg := defaultConfig()

	configPath := strings.TrimSpace(os.Getenv("VOICE_CHAT_CLIENT_CONFIG"))
	if configPath == "" {
		configPath = strings.TrimSpace(os.Getenv("CONFIG_PATH"))
	}
	if configPath == "" {
		configPath = defaultConfigPath
	}

	if _, err := os.Stat(configPath); err == nil {
		payload, err := os.ReadFile(configPath)
		if err != nil {
			return Config{}, fmt.Errorf("client config: read %s: %w", configPath, err)
		}
		if err := yaml.Unmarshal(payload, &cfg); err != nil {
			return Config{}, fmt.Errorf("client config: parse %s: %w", configPath, err)
		}
	} else if !os.IsNotExist(err) || configPath != defaultConfigPath {
		return Config{}, fmt.Errorf("client config: stat %s: %w", configPath, err)
	}

	applyEnvOverrides(&cfg)

	return cfg, nil
}

func (c Config) ContainerConfig() containerpkg.Config {
	iceServers := make([]containerpkg.ICEServer, 0, len(c.WebRTC.ICEServers))
	for _, server := range c.WebRTC.ICEServers {
		iceServers = append(iceServers, containerpkg.ICEServer{
			URLs:       append([]string(nil), server.URLs...),
			Username:   server.Username,
			Credential: server.Credential,
		})
	}

	return containerpkg.Config{
		APIBaseURL:       c.API.BaseURL,
		WebSocketBaseURL: c.API.WebSocketBaseURL,
		KeyringService:   c.Keyring.Service,
		WebRTCICEServers: iceServers,
	}
}

func defaultConfig() Config {
	return Config{
		Env: "local",
		API: API{
			BaseURL: "http://localhost:8082",
		},
		Keyring: Keyring{
			Service: "selfcord",
		},
	}
}

func applyEnvOverrides(cfg *Config) {
	if value := strings.TrimSpace(os.Getenv("VOICE_CHAT_API_BASE_URL")); value != "" {
		cfg.API.BaseURL = value
	}
	if value := strings.TrimSpace(os.Getenv("VOICE_CHAT_WS_BASE_URL")); value != "" {
		cfg.API.WebSocketBaseURL = value
	}
	if value := strings.TrimSpace(os.Getenv("VOICE_CHAT_KEYRING_SERVICE")); value != "" {
		cfg.Keyring.Service = value
	}
}
