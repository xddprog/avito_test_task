package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	HTTP     HTTPConfig
	Postgres PostgresConfig
	Log      LogConfig
}

type LogConfig struct {
	Level  string `env:"LOG_LEVEL" env-default:"INFO"`
	Format string `env:"LOG_FORMAT" env-default:"text"`
}

type HTTPConfig struct {
	Port         string        `env:"HTTP_PORT" env-default:"8080"`
	ReadTimeout  time.Duration `env:"HTTP_READ_TIMEOUT" env-default:"5s"`
	WriteTimeout time.Duration `env:"HTTP_WRITE_TIMEOUT" env-default:"5s"`
}

func (h HTTPConfig) Address() string {
	return ":" + h.Port
}

type PostgresConfig struct {
	Host     string `env:"POSTGRES_HOST" env-default:"localhost"`
	Port     string `env:"POSTGRES_PORT" env-default:"5432"`
	User     string `env:"POSTGRES_USER" env-default:"postgres"`
	Password string `env:"POSTGRES_PASSWORD" env-required:"true"`
	DBName   string `env:"POSTGRES_DB" env-default:"avito_service"`
	SSLMode  string `env:"POSTGRES_SSLMODE" env-default:"disable"`
}

func (p PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		p.User,
		p.Password,
		p.Host,
		p.Port,
		p.DBName,
		p.SSLMode,
	)
}

func Load() (*Config, error) {
	cfg := &Config{}

	if err := cleanenv.ReadConfig(".env", cfg); err != nil {
		if err := cleanenv.ReadEnv(cfg); err != nil {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	return cfg, nil
}
