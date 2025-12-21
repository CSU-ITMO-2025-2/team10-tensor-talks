package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config агрегирует все опции конфигурации.
type Config struct {
	Server ServerConfig `mapstructure:"server"`
	Kafka  KafkaConfig  `mapstructure:"kafka"`
}

// ServerConfig описывает настройки HTTP-сервера.
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// KafkaConfig содержит параметры подключения к Kafka.
type KafkaConfig struct {
	Brokers       []string `mapstructure:"brokers"`
	TopicRequest  string   `mapstructure:"topic_request"`
	TopicResponse string   `mapstructure:"topic_response"`
	ConsumerGroup string   `mapstructure:"consumer_group"`
}

// Load загружает конфигурацию из файла и переменных окружения.
func Load() (Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")
	v.AddConfigPath(".")

	v.SetEnvPrefix("MOCK_INTERVIEW_BUILDER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	// Парсим Kafka brokers из строки или массива
	if brokersStr := v.GetString("kafka.brokers"); brokersStr != "" {
		cfg.Kafka.Brokers = strings.Split(brokersStr, ",")
		for i, broker := range cfg.Kafka.Brokers {
			cfg.Kafka.Brokers[i] = strings.TrimSpace(broker)
		}
	} else {
		cfg.Kafka.Brokers = v.GetStringSlice("kafka.brokers")
	}

	if len(cfg.Kafka.Brokers) == 0 {
		cfg.Kafka.Brokers = []string{"kafka:9092"}
	}

	// Значения по умолчанию
	if cfg.Kafka.TopicRequest == "" {
		cfg.Kafka.TopicRequest = "interview.build.request"
	}
	if cfg.Kafka.TopicResponse == "" {
		cfg.Kafka.TopicResponse = "interview.build.response"
	}

	return cfg, nil
}
