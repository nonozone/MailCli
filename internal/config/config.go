package config

import "gopkg.in/yaml.v3"

type Config struct {
	CurrentAccount string          `yaml:"current_account"`
	Accounts       []AccountConfig `yaml:"accounts,omitempty"`
}

type AccountConfig struct {
	Name   string `yaml:"name"`
	Driver string `yaml:"driver"`
}

func Marshal(cfg Config) ([]byte, error) {
	return yaml.Marshal(cfg)
}

func Unmarshal(data []byte) (Config, error) {
	var cfg Config
	err := yaml.Unmarshal(data, &cfg)
	return cfg, err
}
