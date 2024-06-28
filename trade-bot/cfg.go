package main

import "github.com/caarlos0/env/v10"

type Config struct {
	ApiKey    string `env:"API_KEY"`
	SecretKey string `env:"SECRET_KEY"`
}

var (
	cfg *Config
)

func GetCfg() *Config {
	if cfg == nil {
		_ = env.Parse(&cfg)
	}

	return cfg
}
