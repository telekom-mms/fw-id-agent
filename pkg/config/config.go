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

// Valid returns whether TNDHTTPSConfig is valid
func (t *TNDHTTPSConfig) Valid() bool {
	if t.URL == "" || t.Hash == "" {
		return false
	}
	return true
}

// TNDConfig is the trusted network detection configuration in the
// agent configuration
type TNDConfig struct {
	HTTPSServers []TNDHTTPSConfig
}

// Valid returns whether TNDConfig is valid
func (t *TNDConfig) Valid() bool {
	if len(t.HTTPSServers) == 0 {
		return false
	}
	for _, s := range t.HTTPSServers {
		if !s.Valid() {
			return false
		}
	}
	return true
}

// Config is the agent configuration
type Config struct {
	// ServiceURL is the URL used for requests to the service
	ServiceURL string
	// Realm is the client's Kerberos realm used for requests to the service
	Realm string
	// KeepAlive is the default client keep-alive time in minutes
	KeepAlive int
	// LoginTimeout is the client's timeout for login requests to the service in seconds
	LoginTimeout int
	// LogoutTimeout is the client's timeout for logout requests to the service in seconds
	LogoutTimeout int
	// RetryTimer is the client's login retry timer in case of errors in seconds
	RetryTimer int
	// TND is the client's trusted network detection configuration
	TND TNDConfig
	// Verbose specifies whether the client should show verbose output
	Verbose bool
	// MinUserID is the minimum allowed user ID
	MinUserID int
	// StartDelay is the time the agent sleeps before starting in seconds
	StartDelay int
	// Notifications specifies whether the agent should show desktop notifications
	Notifications bool
}

// GetKeepAlive returns the client keep-alive time as Duration
func (c *Config) GetKeepAlive() time.Duration {
	return time.Duration(c.KeepAlive) * time.Minute
}

// GetLoginTimeout returns the client login timeout as Duration
func (c *Config) GetLoginTimeout() time.Duration {
	return time.Duration(c.LoginTimeout) * time.Second
}

// GetLogoutTimeout returns the client logout timeout as Duration
func (c *Config) GetLogoutTimeout() time.Duration {
	return time.Duration(c.LogoutTimeout) * time.Second
}

// GetRetryTimer returns the client retry timer as Duration
func (c *Config) GetRetryTimer() time.Duration {
	return time.Duration(c.RetryTimer) * time.Second
}

// GetStartDelay returns the agent start delay as Duration
func (c *Config) GetStartDelay() time.Duration {
	return time.Duration(c.StartDelay) * time.Second
}

// Valid returns whether Config is valid
func (c *Config) Valid() bool {
	if c == nil ||
		c.ServiceURL == "" ||
		c.Realm == "" ||
		c.KeepAlive < 0 ||
		c.LoginTimeout < 0 ||
		c.LogoutTimeout < 0 ||
		c.RetryTimer < 0 ||
		!c.TND.Valid() ||
		c.MinUserID < 0 ||
		c.StartDelay < 0 {
		return false
	}
	return true
}

// JSON returns Config as JSON
func (c *Config) JSON() ([]byte, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// Default returns a new config with default values
func Default() *Config {
	return &Config{
		KeepAlive:     5,
		LoginTimeout:  15,
		LogoutTimeout: 5,
		RetryTimer:    15,
		MinUserID:     1000,
		StartDelay:    20,
		Notifications: true,
	}
}

// Load loads the json configuration from file path
func Load(path string) (*Config, error) {
	// read file contents
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// parse config
	cfg := Default()
	if err := json.Unmarshal(file, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
