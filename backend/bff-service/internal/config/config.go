package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

/*
Пакет config отвечает за загрузку конфигурации BFF-сервиса.

Источники:
  - YAML-файл `config/config.yaml`;
  - переменные окружения с префиксом `BFF_` (перекрывают значения из файла).

Настройки:
  - адрес HTTP-сервера;
  - параметры подключения к auth-service;
  - CORS (разрешённые origin и заголовки).
*/

// Config агрегирует все опции конфигурации BFF.
// Используется на этапе старта сервиса для настройки HTTP-сервера, CORS и клиента auth-service.
type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	AuthService AuthServiceConfig `mapstructure:"auth_service"`
	CORS        CORSConfig        `mapstructure:"cors"`
}

// ServerConfig описывает настройки HTTP-сервера BFF.
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// AuthServiceConfig содержит параметры подключения к auth-service.
type AuthServiceConfig struct {
	BaseURL        string `mapstructure:"base_url"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
}

// CORSConfig описывает настройки CORS.
type CORSConfig struct {
	AllowOrigins []string `mapstructure:"allow_origins"`
	AllowHeaders []string `mapstructure:"allow_headers"`
}

// Load загружает конфигурацию из файла и окружения.
// Порядок приоритетов:
//  1. переменные окружения с префиксом BFF_,
//  2. значения из `config/config.yaml`.
//
// После чтения конфигурации дополнительно заполняются списки CORS-оригинов/заголовков.
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
