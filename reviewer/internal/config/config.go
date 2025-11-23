package config

import (
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type HTTPConfig struct {
	Address string        `yaml:"address" env:"PR_REVIEWER_ADDRESS" env-default:"localhost:8080"`
	Timeout time.Duration `yaml:"timeout" env:"PR_REVIEWER_TIMEOUT" env-default:"10s"`
}

type Config struct {
	LogLevel   string     `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`
	HTTPConfig HTTPConfig `yaml:"pr-reviewer"`
	DBAddress  string     `yaml:"db_address" env:"DB_ADDRESS" env-default:"localhost:5432"`
}

func MustLoad(path string) (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
