package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"log"
	"time"
)

type Config struct {
	Env         string `env:"APP_ENV" env-required:"true"`
	StoragePath string `env:"STORAGE_PATH" env-required:"true"`
	HttpServer  struct {
		Address     string        `env:"HTTP_ADDRESS"     env-default:"localhost" env-required:"true"`
		Port        string        `env:"HTTP_PORT"        env-default:"8082"     env-required:"true"`
		Timeout     time.Duration `env:"HTTP_TIMEOUT"     env-default:"4s"       env-required:"true"`
		IdleTimeout time.Duration `env:"HTTP_IDLE_TIMEOUT" env-default:"60s"     env-required:"true"`
	}
	Database struct {
		Host     string `env:"DB_HOST"     env-required:"true"`
		Port     string `env:"DB_PORT"     env-required:"true"`
		User     string `env:"DB_USER"     env-required:"true"`
		Password string `env:"DB_PASSWORD" env-required:"true"`
		Name     string `env:"DB_NAME"     env-required:"true"`
		SSLMode  string `env:"DB_SSLMODE" env-required:"true"`
	}
}

func MustLoad() *Config {
	var cfg Config

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Fatalf("cannot read environment: %s", err)
	}

	return &cfg
}
