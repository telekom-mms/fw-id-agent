package config

import (
	"encoding/json"
	"os"
	"time"
)

// TNDHTTPSConfig is a https configuration in the
// trusted network detection configuration
type TNDHTTPSConfig struct {
	URL  string
	Hash string
}

// TNDConfig is the trusted network detection configuration in the
// agent configuration
type TNDConfig struct {
	HTTPSServers []TNDHTTPSConfig
}

// Config is the agent configuration
type Config struct {
	ServiceURL string
	Realm      string
	KeepAlive  int
	Timeout    int
	RetryTimer int
	TND        TNDConfig
	Verbose    bool
	MinUserID  int
	StartDelay int
}

// GetKeepAlive returns the client keep-alive time as Duration
func (c *Config) GetKeepAlive() time.Duration {
	return time.Duration(c.KeepAlive) * time.Minute
}

// GetTimeout returns the client timeout as Duration
func (c *Config) GetTimeout() time.Duration {
	return time.Duration(c.Timeout) * time.Second
}

// GetRetryTimer returns the client retry timer as Duration
func (c *Config) GetRetryTimer() time.Duration {
	return time.Duration(c.RetryTimer) * time.Second
}

// Load loads the json configuration from file path
func Load(path string) (*Config, error) {
	// read file contents
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// parse config
	cfg := &Config{}
	if err := json.Unmarshal(file, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
