package config

import (
	"os"
	"strings"
)

// Config содержит всю конфигурацию приложения
type Config struct {
	Port     string
	Password string
}

var globalConfig *Config

// Init загружает конфигурацию из переменных окружения
func Init() *Config {
	cfg := &Config{
		Port:     getEnv("TODO_PORT", "7540"),
		Password: strings.TrimSpace(getEnv("TODO_PASSWORD", "")),
	}
	globalConfig = cfg
	return cfg
}

// Get возвращает глобальную конфигурацию
func Get() *Config {
	if globalConfig == nil {
		return Init()
	}
	return globalConfig
}

// getEnv получает переменную окружения или возвращает значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
