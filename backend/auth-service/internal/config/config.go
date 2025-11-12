package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config aggregates runtime options.
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	UserStore UserStoreConfig `mapstructure:"user_store"`
	JWT       JWTConfig       `mapstructure:"jwt"`
}

// ServerConfig defines HTTP server settings.
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// UserStoreConfig holds remote service information.
type UserStoreConfig struct {
	BaseURL        string `mapstructure:"base_url"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
}

// JWTConfig encapsulates token settings.
type JWTConfig struct {
	Issuer          string        `mapstructure:"issuer"`
	Audience        string        `mapstructure:"audience"`
	AccessTokenTTL  time.Duration `mapstructure:"-"`
	RefreshTokenTTL time.Duration `mapstructure:"-"`
	Secret          string        `mapstructure:"secret"`

	AccessTokenTTLRaw  string `mapstructure:"access_token_ttl"`
	RefreshTokenTTLRaw string `mapstructure:"refresh_token_ttl"`
}

// Load reads configuration file and environment variables.
func Load() (Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")
	v.AddConfigPath(".")

	v.SetEnvPrefix("AUTH")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.JWT.parseDurations(); err != nil {
		return Config{}, err
	}

	if cfg.JWT.Secret == "" {
		return Config{}, fmt.Errorf("jwt.secret must be provided")
	}

	return cfg, nil
}

func (j *JWTConfig) parseDurations() error {
	access, err := time.ParseDuration(j.AccessTokenTTLRaw)
	if err != nil {
		return fmt.Errorf("parse jwt.access_token_ttl: %w", err)
	}
	refresh, err := time.ParseDuration(j.RefreshTokenTTLRaw)
	if err != nil {
		return fmt.Errorf("parse jwt.refresh_token_ttl: %w", err)
	}
	j.AccessTokenTTL = access
	j.RefreshTokenTTL = refresh
	return nil
}
