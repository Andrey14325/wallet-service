package config

import (
	"flag"
	"os"
	"time"
)

const (
	defAddress  = ":8080"
	defDatabase = "postgres://wallet:wallet@localhost:5432/wallet?sslmode=disable"
	defInterval = 10 * time.Second
)

type Config struct {
	ServAddress string
	Database    string
	Interval    time.Duration
}

func ParseConfig() Config {
	var cfg Config
	flag.StringVar(&cfg.ServAddress, "addr", defAddress, "Server address")
	flag.StringVar(&cfg.Database, "database", defDatabase, "Database address")
	flag.DurationVar(&cfg.Interval, "interval", defInterval, "Database interval")
	flag.Parse()

	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.Database = dbURL
	}

	return cfg
}
