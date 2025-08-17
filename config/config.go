package config

import (
	"fmt"
	"log/slog"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Config represents the comprehensive application configuration
type Config struct {
	ServerPort       string `validate:"required"`
	LogLevel         slog.Level
	GinMode          string   `validate:"required,oneof=debug release test"`
	EthereumRPCURL   string   `validate:"required,url"`
	EthereumWSURL    string   `validate:"required,url"`
	RedisURL         string   `validate:"required,url"`
	KafkaBrokers     []string `validate:"required"`
	WatchedAddresses []string `validate:"required"`
}

// Validate performs structural validation on the configuration
func (c *Config) Validate() error {
	validate := validator.New()

	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	return nil
}

// LoadConfig loads and validates the application configuration
func LoadConfig() (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server_port", "8080")
	v.SetDefault("log_level", "info")
	v.SetDefault("gin_mode", "debug")

	// Blockchain and infrastructure defaults
	v.SetDefault("ethereum_rpc_url", "") // Allow empty, will be validated
	v.SetDefault("ethereum_ws_url", "")  // Allow empty, will be validated
	v.SetDefault("redis_url", "redis://localhost:6379/0")
	v.SetDefault("kafka_brokers", []string{"localhost:9092"})

	// Watched addresses default (empty list)
	v.SetDefault("watched_addresses", []string{})

	// Retry configuration defaults
	v.SetDefault("retry.base_delay", 100)
	v.SetDefault("retry.max_delay", 5000)
	v.SetDefault("retry.max_retries", 5)

	// Configure config file search paths
	v.SetConfigName(".env") // name of config file (without extension)
	v.SetConfigType("env")  // REQUIRED if the config file does not have the extension in the name
	v.AddConfigPath(".")    // path to look for the config file in
	v.AddConfigPath("./config")

	// Read config file if it exists
	if err := v.ReadInConfig(); err != nil {
		// It's okay if no config file is found
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Tell Viper to automatically override values from environment variables
	v.AutomaticEnv()

	// Bind environment variables
	envVars := []struct {
		key, envName string
	}{
		{"server_port", "SERVER_PORT"},
		{"log_level", "LOG_LEVEL"},
		{"gin_mode", "GIN_MODE"},
		{"ethereum_rpc_url", "ETHEREUM_RPC_URL"},
		{"ethereum_ws_url", "ETHEREUM_WS_URL"},
		{"redis_url", "REDIS_URL"},
		{"kafka_brokers", "KAFKA_BROKERS"},
		{"watched_addresses", "WATCHED_ADDRESSES"},
		{"retry.base_delay", "RETRY_BASE_DELAY"},
		{"retry.max_delay", "RETRY_MAX_DELAY"},
		{"retry.max_retries", "RETRY_MAX_RETRIES"},
	}

	for _, ev := range envVars {
		if err := v.BindEnv(ev.key, ev.envName); err != nil {
			return nil, fmt.Errorf("failed to bind environment variable %s: %w", ev.envName, err)
		}
	}

	// Prepare configuration
	config := &Config{
		ServerPort:       v.GetString("server_port"),
		LogLevel:         getLogLevel(v.GetString("log_level")),
		GinMode:          v.GetString("gin_mode"),
		EthereumRPCURL:   v.GetString("ethereum_rpc_url"),
		EthereumWSURL:    v.GetString("ethereum_ws_url"),
		RedisURL:         v.GetString("redis_url"),
		KafkaBrokers:     v.GetStringSlice("kafka_brokers"),
		WatchedAddresses: v.GetStringSlice("watched_addresses"),
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// getLogLevel converts string log level to slog.Level
func getLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
