package config

import (
	"errors"
	"os"
	"strconv"
)

type Config struct {
	Address   string
	DSN       string
	JWTSecret string
}

func Load() Config {
	return Config{
		Address:   getenv("APPPLOT_ADDRESS", "127.0.0.1:8080"),
		DSN:       getenv("APPPLOT_DSN", ""),
		JWTSecret: getenv("APPPLOT_JWT_SECRET", ""),
	}
}

func (c Config) Validate() error {
	if c.DSN == "" {
		return errors.New("APPPLOT_DSN is required")
	}
	if len(c.JWTSecret) < 32 {
		return errors.New("APPPLOT_JWT_SECRET must be at least 32 chars")
	}
	return nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
