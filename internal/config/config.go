package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env         string `yaml:"env"`
	StoragePath string `yaml:"storage_path" env_required:"true"`
	HttpServer  struct {
		Address     string        `yaml:"address" env_default:"localhost" env_required:"true"`
		Port        string        `yaml:"port" env_default:"8082" env_required:"true"`
		Timeout     time.Duration `yaml:"timeout" env_default:"4s" env_required:"true"`
		IdleTimeout time.Duration `yaml:"iddle_timeout" env_default:"60s" env_required:"true"`
	} `yaml:"http_server"`
	Database struct {
		Host     string `yaml:"host" env_required:"true"`
		Port     string `yaml:"port" env_required:"true"`
		User     string `yaml:"user" env_required:"true"`
		Password string `yaml:"password" env_required:"true"`
		Name     string `yaml:"database" env_required:"true"`
		SSLMode  string `yaml:"ssl_mode" env_required:"true"`
	}
}

func MustLoad() *Config {
	configPath := "config/local.yaml"
	if configPath == "" {
		log.Fatal("CONFIG_PATH is not set")
	}

	// check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exist: %s", configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("error reading config file: %s", err)
	}

	return &cfg
}
