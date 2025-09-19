package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

func getEnv(key, defaultVal string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	if value, ok := os.LookupEnv(key); ok {
		if v, err := strconv.Atoi(value); err == nil {
			return v
		}
	}
	return defaultVal
}

func getEnvAsBool(key string, defaultVal bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		if v, err := strconv.ParseBool(value); err == nil {
			return v
		}
	}
	return defaultVal
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	if value, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultVal
}

func getEnvAsStringSlice(key string, defaults []string) []string {
	if value, ok := os.LookupEnv(key); ok {
		parts := strings.Split(value, ",")
		filtered := make([]string, 0, len(parts))
		for _, part := range parts {
			p := strings.TrimSpace(part)
			if p != "" {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) > 0 {
			return filtered
		}
	}
	return defaults
}
