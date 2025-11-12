package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config aggregates runtime options.
type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	AuthService AuthServiceConfig `mapstructure:"auth_service"`
	CORS        CORSConfig        `mapstructure:"cors"`
}

// ServerConfig defines HTTP server options.
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// AuthServiceConfig holds remote auth service details.
type AuthServiceConfig struct {
	BaseURL        string `mapstructure:"base_url"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
}

// CORSConfig describes cross-origin settings.
type CORSConfig struct {
	AllowOrigins []string `mapstructure:"allow_origins"`
	AllowHeaders []string `mapstructure:"allow_headers"`
}

// Load reads configuration from file and environment.
func Load() (Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")
	v.AddConfigPath(".")

	v.SetEnvPrefix("BFF")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	cfg.CORS.AllowOrigins = v.GetStringSlice("cors.allow_origins")
	cfg.CORS.AllowHeaders = v.GetStringSlice("cors.allow_headers")

	return cfg, nil
}
