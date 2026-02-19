package main

import (
	"flag"
	"os"
	"strconv"
)

type Config struct {
	DBPath        string
	MaxUploadSize int64
	DefaultTTL    int
	Port          string
}

func LoadConfig() *Config {
	defaultDBPath := getEnv("DB_PATH", "/dev/shm/messages.db")
	defaultMaxUploadSize := getEnvInt64("MAX_UPLOAD_SIZE", 10<<20) // Default 10MB
	defaultDefaultTTL := getEnvInt("DEFAULT_TTL", 86400)           // Default 1 day
	defaultPort := getEnv("PORT", "8080")

	DBPath := flag.String("db", defaultDBPath, "Path to the database file")
	MaxUploadSize := flag.Int64("max-upload-size", defaultMaxUploadSize, "Maximum upload size in bytes")
	DefaultTTL := flag.Int("default-ttl", defaultDefaultTTL, "Default TTL for messages in seconds")
	Port := flag.String("port", defaultPort, "Port to listen on")

	flag.Parse()

	return &Config{
		DBPath:        *DBPath,
		MaxUploadSize: *MaxUploadSize,
		DefaultTTL:    *DefaultTTL,
		Port:          *Port,
	}

}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	valStr := getEnv(key, "")
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	valStr := getEnv(key, "")
	if val, err := strconv.ParseInt(valStr, 10, 64); err == nil {
		return val
	}
	return fallback
}
