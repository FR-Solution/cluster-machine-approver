package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

func Get(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
