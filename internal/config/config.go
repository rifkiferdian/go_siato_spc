package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment string
	Name        string
	Addr        string
	BaseURL     string
	Database    DatabaseConfig
}

type DatabaseConfig struct {
	Host            string
	Port            string
	Name            string
	User            string
	Password        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func Load() (Config, error) {
	_ = godotenv.Load()

	maxOpen, err := envInt("DB_MAX_OPEN_CONNS", 25)
	if err != nil {
		return Config{}, err
	}
	maxIdle, err := envInt("DB_MAX_IDLE_CONNS", 10)
	if err != nil {
		return Config{}, err
	}
	lifetime, err := time.ParseDuration(env("DB_CONN_MAX_LIFETIME", "5m"))
	if err != nil {
		return Config{}, fmt.Errorf("DB_CONN_MAX_LIFETIME: %w", err)
	}

	cfg := Config{
		Environment: env("APP_ENV", "development"),
		Name:        env("APP_NAME", "SIATO SPC"),
		Addr:        env("APP_ADDR", ":8080"),
		BaseURL:     env("APP_BASE_URL", "http://localhost:8080"),
		Database: DatabaseConfig{
			Host:            env("DB_HOST", "127.0.0.1"),
			Port:            env("DB_PORT", "3306"),
			Name:            env("DB_NAME", "siato_spc"),
			User:            env("DB_USER", "root"),
			Password:        os.Getenv("DB_PASSWORD"),
			MaxOpenConns:    maxOpen,
			MaxIdleConns:    maxIdle,
			ConnMaxLifetime: lifetime,
		},
	}

	if cfg.Database.Name == "" || cfg.Database.User == "" {
		return Config{}, fmt.Errorf("DB_NAME dan DB_USER wajib diisi")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	value := env(key, strconv.Itoa(fallback))
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s harus berupa angka: %w", key, err)
	}
	return n, nil
}
