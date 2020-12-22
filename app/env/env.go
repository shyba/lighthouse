package env

import (
	"github.com/lbryio/lbry.go/extras/errors"

	e "github.com/caarlos0/env"
)

// Config holds the environment configuration used by lighthouse.
type Config struct {
	ChainQueryDsn    string `env:"CHAINQUERY_DSN"`
	SyncStateDir     string `env:"SYNCSTATEDIR"`
	ElasticSearchURL string `env:"ELASTICSEARCHURL"`
	InternalAPIDSN   string `env:"INTERNALAPIS_DSN"`
	APIURL           string `env:"API_URL"`
	APIToken         string `env:"API_TOKEN"`
	SlackHookURL     string `env:"SLACKHOOKURL"`
	SlackChannel     string `env:"SLACKCHANNEL"`
}

// NewWithEnvVars creates an Config from environment variables
func NewWithEnvVars() (*Config, error) {
	cfg := &Config{}
	err := e.Parse(cfg)
	if err != nil {
		return nil, errors.Err(err)
	}

	if cfg.ChainQueryDsn == "" {
		return nil, errors.Err("CHAINQUERY_DSN env var required")
	}

	return cfg, nil
}
