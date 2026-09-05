package config

import (
	"os"
	"time"
)

type Config struct {
	DataDir     string
	ListenAddr  string
	Tick        time.Duration
	GraceWindow time.Duration
}

func Load() Config {
	return Config{
		DataDir:     env("DATA_DIR", "data"),
		ListenAddr:  env("LISTEN_ADDR", "127.0.0.1:8080"),
		Tick:        duration("TICK", 60*time.Second),
		GraceWindow: duration("GRACE_WINDOW", 30*time.Minute),
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func duration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
