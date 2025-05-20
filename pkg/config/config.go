package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	NodeID        string `mapstructure:"node_id"`
	BindAddr      string `mapstructure:"bind_addr"`
	AdvertiseAddr string `mapstructure:"advertise_addr"`
	HTTPAddr      string `mapstructure:"http_addr"`
	DataDir       string `mapstructure:"data_dir"`
	JoinAddr      string `mapstructure:"join_addr"`
}

// LoadConfig loads the configuration from a file
func LoadConfig(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
