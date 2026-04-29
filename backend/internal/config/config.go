package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort               string
	DatabaseURL           string
	CORSOrigins           []string
	EquityCacheTTLSeconds int
	BybitDemo             bool
	BybitCredential       ExchangeCredential
	BitgetDemo            bool
	BitgetCredential      ExchangeCredential
}

type ExchangeCredential struct {
	APIKey     string
	APISecret  string
	Passphrase string
}

func Load() Config {
	_ = godotenv.Load()

	return Config{
		AppPort:               envOrDefault("APP_PORT", "8080"),
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		CORSOrigins:           splitCSV(os.Getenv("CORS_ORIGINS")),
		EquityCacheTTLSeconds: envIntOrDefault("EQUITY_CACHE_TTL_SECONDS", 15),
		BybitDemo:             envBoolOrDefault("BYBIT_DEMO", false),
		BybitCredential: ExchangeCredential{
			APIKey:    os.Getenv("BYBIT_CREDENTIAL_API_KEY"),
			APISecret: os.Getenv("BYBIT_CREDENTIAL_API_SECRET"),
		},
		BitgetDemo: envBoolOrDefault("BITGET_DEMO", false),
		BitgetCredential: ExchangeCredential{
			APIKey:     os.Getenv("BITGET_CREDENTIAL_API_KEY"),
			APISecret:  os.Getenv("BITGET_CREDENTIAL_API_SECRET"),
			Passphrase: os.Getenv("BITGET_CREDENTIAL_PASSPHRASE"),
		},
	}
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func splitCSV(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func envBoolOrDefault(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envIntOrDefault(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
