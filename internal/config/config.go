package config

import (
	"encoding/json"
	"os"
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
