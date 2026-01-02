package config

import (
	"flag"
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
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
	var configPath string

	configPath = os.Getenv("CONFIG_PATH")

	if configPath == "" {
		path := flag.String("config", "", "path to the configuration file")
		flag.Parse()
		configPath = *path

		// if still don't get the path
		if configPath == "" {
			log.Fatal("Config file path is not set")
		}
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("Config file does't exist at path: %s", configPath)
	}

	var cfg Config
	err := cleanenv.ReadConfig(configPath, &cfg)
	if err != nil {
		log.Fatalf("cannot read config file: %s", err.Error())
	}

	return &cfg
}
