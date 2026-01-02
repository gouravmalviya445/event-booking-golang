package config

import (
	"flag"
	"log"
	"log/slog"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

// http-serve
type HTTPServer struct {
	Port string `yaml:"port" env:"PORT" env-default:"8080"`
}

// db
type Database struct {
	Url    string `yaml:"url" env:"DATABASE_URL"`
	DbName string `yaml:"db_name" env:"DB_NAME"`
}

type Config struct {
	Env        string `yaml:"env" env:"ENV" env-default:"development"`
	Database   `yaml:"database"`
	HTTPServer `yaml:"http_server"`
}

func MustLoad() *Config {
	// 1. Load .env file (if exists) into system environment
	// This ensures both os.Getenv and cleanenv can see these values
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found, relying on system env vars")
	}

	var cfg Config

	// 2. Determine Config Path (Priority: env file -> flag -> empty)
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		// Only parse flags if env var is empty
		// Note: Be careful with flag.Parse() in tests, but fine for main
		var path string
		flag.StringVar(&path, "config", "", "path to the configuration file")
		flag.Parse()
		configPath = path
	}

	// 3. Logic: File vs Environment
	if configPath == "" {
		// No config path provided at all.
		// Assume we are running in "Env Only" mode (Production)
		slog.Info("No config path provided, reading purely from environment")
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			log.Fatalf("Failed to read environment variables: %s", err.Error())
		}
	} else {
		// Config path is provided.
		// Check if the file actually exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			// if a user EXPLICITLY provides a path that is wrong, we should warn and fallback.
			slog.Warn("Config file specified but not found, falling back to environment variables", slog.String("path", configPath))
			if err := cleanenv.ReadEnv(&cfg); err != nil {
				log.Fatalf("Failed to read environment variables: %s", err.Error())
			}
		} else {
			// File exists.
			// cleanenv.ReadConfig reads the file AND overrides with Env vars automatically
			slog.Info("Reading configuration from file", slog.String("path", configPath))
			if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
				log.Fatalf("cannot read config file: %s", err.Error())
			}
		}
	}

	return &cfg
}
