package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPort           = "8080"
	defaultEnvironment    = "development"
	defaultAllowedOrigins = "http://localhost:5173,http://127.0.0.1:5173"
)

type Config struct {
	ServiceName    string
	Version        string
	Environment    string
	Addr           string
	AllowedOrigins []string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	LogLevel       slog.Level
}

func Load() Config {
	port := getEnv("HERO3_PORT", defaultPort)

	return Config{
		ServiceName:    "Hero3 API",
		Version:        getEnv("HERO3_VERSION", "0.1.0"),
		Environment:    getEnv("HERO3_ENV", defaultEnvironment),
		Addr:           ":" + port,
		AllowedOrigins: splitCSV(getEnv("HERO3_ALLOWED_ORIGINS", defaultAllowedOrigins)),
		ReadTimeout:    getDurationEnv("HERO3_READ_TIMEOUT", 5*time.Second),
		WriteTimeout:   getDurationEnv("HERO3_WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:    getDurationEnv("HERO3_IDLE_TIMEOUT", 60*time.Second),
		LogLevel:       getLogLevel(getEnv("HERO3_LOG_LEVEL", "info")),
	}
}

func getEnv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func splitCSV(value string) []string {
	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	if duration, err := time.ParseDuration(value); err == nil {
		return duration
	}

	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second
	}

	return fallback
}

func getLogLevel(value string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
