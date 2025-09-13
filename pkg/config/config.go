package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Machine struct {
	ID             int               `yaml:"id"`
	Username       string            `yaml:"username"`
	Hostname       string            `yaml:"hostname"`
	Port           int               `yaml:"port"`
	AddKeysToAgent bool              `yaml:"add_keys_to_agent"`
}

type Config struct {
	Machines []Machine `yaml:"machines"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
