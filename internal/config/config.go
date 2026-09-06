package config

import (
	"os"
	"path/filepath"
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
		DataDir:     env("DATA_DIR", defaultDataDir()),
		ListenAddr:  env("LISTEN_ADDR", "127.0.0.1:20648"),
		Tick:        duration("TICK", 60*time.Second),
		GraceWindow: duration("GRACE_WINDOW", 30*time.Minute),
	}
}

// defaultDataDir is the platform config dir: ~/Library/Application Support on
// macOS, ~/.config on Linux. `make run` and the installed agent share it, so
// pairing once is enough. Falls back to ./data if the home dir is unknown.
func defaultDataDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "data"
	}
	return filepath.Join(dir, "whatsapp-scheduler")
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
