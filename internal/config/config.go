package config

import (
	"github.com/spf13/viper"
)

// Config holds all configuration for our program
type Config struct {
	// Node configuration
	NodeID   string `mapstructure:"node_id"`
	BindAddr string `mapstructure:"bind_addr"`
	DataDir  string `mapstructure:"data_dir"`

	// Raft configuration
	RaftBindAddr string `mapstructure:"raft_bind_addr"`
	JoinAddr     string `mapstructure:"join_addr"`

	// HTTP API configuration
	HTTPAddr string `mapstructure:"http_addr"`
}

// LoadConfig loads the configuration from a file
func LoadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}

	return config, nil
}
