package config

import (
	"flag"
	"os"
)

const (
	defAddress  = ":8080"
	defDatabase = "postgres://wallet:wallet@localhost:5432/wallet?sslmode=disable"
)

type Config struct {
	ServAddress string
	Database    string
}

func ParseConfig() Config {
	var cfg Config
	flag.StringVar(&cfg.ServAddress, "addr", defAddress, "Server address")
	flag.StringVar(&cfg.Database, "database", defDatabase, "Database address")
	flag.Parse()

	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.Database = dbURL
	}

	return cfg
}
