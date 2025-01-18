package main

import (
	"fmt"
	"strings"

	"UD_telegram_miniapp/internal/repository"

	"github.com/spf13/viper"
)

const (
	configPath   = "./"
	configName   = "config"
	configFormat = "yaml"
)

type Config struct {
	Database repository.Config `mapstructure:"database"`
	Server   ServerConfig      `mapstructure:"server"`

	TelegramAuth TelegramAuthConfig `mapstructure:"telegramAuth"`

	LogLevel string `mapstructure:"logLevel"`
}

type ServerConfig struct {
	Host      string `mapstructure:"host"`
	Port      string `mapstructure:"port"`
	DebugMode bool   `mapstructure:"debugMode"`
}

type TelegramAuthConfig struct {
	TelegramBotToken string `mapstructure:"telegramBotToken"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName(configName)
	viper.AddConfigPath(configPath)
	viper.SetConfigType(configFormat)

	viper.AutomaticEnv()
	viper.SetEnvPrefix("APP")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
