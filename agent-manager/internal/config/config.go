package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	Server     ServerConfig     `json:"server"`
	Redis      RedisConfig      `json:"redis"`
	Env        string           `json:"env"`
	Pagination PaginationConfig `json:"pagination"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Address      string        `json:"address"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Address  string `json:"address"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

// PaginationConfig holds pagination configuration
type PaginationConfig struct {
	DefaultPageSize int `json:"default_page_size"`
	MaxPageSize     int `json:"max_page_size"`
}

// Load reads configuration from environment variables with sensible defaults
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Address:      getEnv("SERVER_ADDRESS", ":8080"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Redis: RedisConfig{
			Address:  getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getIntEnv("REDIS_DB", 0),
		},
		Env: getEnv("ENV", "development"),
		Pagination: PaginationConfig{
			DefaultPageSize: getIntEnv("PAGINATION_DEFAULT_PAGE_SIZE", 20),
			MaxPageSize:     getIntEnv("PAGINATION_MAX_PAGE_SIZE", 100),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Server.Address == "" {
		return fmt.Errorf("server address cannot be empty")
	}
	if c.Redis.Address == "" {
		return fmt.Errorf("redis address cannot be empty")
	}
	if c.Server.ReadTimeout < 0 {
		return fmt.Errorf("server read timeout must be positive")
	}
	if c.Server.WriteTimeout < 0 {
		return fmt.Errorf("server write timeout must be positive")
	}
	if c.Server.IdleTimeout < 0 {
		return fmt.Errorf("server idle timeout must be positive")
	}
	if c.Pagination.DefaultPageSize <= 0 {
		return fmt.Errorf("pagination default page size must be positive")
	}
	if c.Pagination.MaxPageSize <= 0 {
		return fmt.Errorf("pagination max page size must be positive")
	}
	if c.Pagination.DefaultPageSize > c.Pagination.MaxPageSize {
		return fmt.Errorf("pagination default page size cannot exceed max page size")
	}
	return nil
}

// getEnv returns environment variable value or default if not set
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getIntEnv returns environment variable as int or default if not set/invalid
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getDurationEnv returns environment variable as duration or default if not set/invalid
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
