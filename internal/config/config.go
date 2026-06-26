package config

import "os"

type Config struct {
	Addr        string
	DatabaseURL string
}

func Load() Config {
	return Config{
		Addr:        getEnv("ADDR", ":8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://subs:subs@localhost:5432/subs?sslmode=disable"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
