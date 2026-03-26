package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	CurrentAccount string          `yaml:"current_account"`
	Accounts       []AccountConfig `yaml:"accounts,omitempty"`
}

type AccountConfig struct {
	Name     string `yaml:"name"`
	Driver   string `yaml:"driver"`
	Host     string `yaml:"host,omitempty"`
	Port     int    `yaml:"port,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	TLS      bool   `yaml:"tls,omitempty"`
	Mailbox  string `yaml:"mailbox,omitempty"`
	SMTPHost string `yaml:"smtp_host,omitempty"`
	SMTPPort int    `yaml:"smtp_port,omitempty"`
	SMTPUsername string `yaml:"smtp_username,omitempty"`
	SMTPPassword string `yaml:"smtp_password,omitempty"`
	SMTPTLS bool   `yaml:"smtp_tls,omitempty"`
}

func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".mailcli.yaml"
	}

	return filepath.Join(home, ".config", "mailcli", "config.yaml")
}

func Marshal(cfg Config) ([]byte, error) {
	return yaml.Marshal(cfg)
}

func Unmarshal(data []byte) (Config, error) {
	var cfg Config
	err := yaml.Unmarshal(data, &cfg)
	return cfg, err
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	return Unmarshal(data)
}

func (c Config) ResolveAccount(name string) (AccountConfig, error) {
	target := name
	if target == "" {
		target = c.CurrentAccount
	}

	for _, account := range c.Accounts {
		if account.Name == target {
			return account, nil
		}
	}

	if target == "" {
		return AccountConfig{}, fmt.Errorf("no account selected")
	}

	return AccountConfig{}, fmt.Errorf("account not found: %s", target)
}
