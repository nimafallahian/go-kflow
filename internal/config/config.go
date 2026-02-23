package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Config holds the runtime configuration for the indexer service.
type Config struct {
	KafkaBrokers   []string `env:"KAFKA_BROKERS,notEmpty" envSeparator:","`
	KafkaTopic     string   `env:"KAFKA_TOPIC" envDefault:"messages"`
	KafkaGroupID   string   `env:"KAFKA_GROUP_ID" envDefault:"indexer-group"`
	ElasticURLs    []string `env:"ELASTIC_URLS,notEmpty" envSeparator:","`
	ElasticIndex   string   `env:"ELASTIC_INDEX" envDefault:"messages"`
	WorkerCount    int      `env:"WORKER_COUNT" envDefault:"5"`
	LogLevel       string   `env:"LOG_LEVEL" envDefault:"INFO"`
}

// Load parses environment variables into Config.
func Load() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse env config: %w", err)
	}
	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = 1
	}
	return &cfg, nil
}

