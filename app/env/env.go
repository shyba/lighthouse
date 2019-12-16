package env

import (
	"github.com/lbryio/lbry.go/extras/errors"

	e "github.com/caarlos0/env"
)

type Config struct {
	ChainQueryDsn    string `env:"CHAINQUERY_DSN"`
	SyncStateDir     string `env:"SYNCSTATEDIR"`
	ElasticSearchURL string `env:"ELASTICSEARCHURL"`
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
