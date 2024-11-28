package config

import (
	"flag"
	"strings"

	"github.com/caarlos0/env/v6"
	"github.com/rs/zerolog"
)

type Config struct {
	Address        string `env:"RUN_ADDRESS"        envDefault:"localhost:8081"`
	AccrualAddress string `env:"ACCRUAL_SYSTEM_ADDRESS" envDefault:"localhost:8080"`
	Database       string `env:"DATABASE_URI"          envDefault:""`
	JwtSecret      string `env:"JWT_SECRET"          envDefault:""`
	Redis          string `env:"REDIS"          envDefault:"localhost:6379"`
	Kafka          string `env:"KAFKA"          envDefault:"localhost:9092"`
}

// Builder defines the builder for the Config struct.
type Builder struct {
	cfg    *Config
	logger *zerolog.Logger
}

// NewConfigBuilder initializes the ConfigBuilder with default values.
func NewConfigBuilder(log *zerolog.Logger) *Builder {
	return &Builder{
		cfg: &Config{
			Address:        "",
			AccrualAddress: "",
			Database:       "",
			JwtSecret:      "",
			Redis:          "",
			Kafka:          "",
		},
		logger: log,
	}
}

// FromEnv parses environment variables into the ConfigBuilder.
func (b *Builder) FromEnv() *Builder {
	if err := env.Parse(b.cfg); err != nil {
		b.logger.Error().Err(err).Msg("failed to parse environment variables")
	}

	return b
}

// FromFlags parses command line flags into the ConfigBuilder.
func (b *Builder) FromFlags() *Builder {
	flag.StringVar(&b.cfg.Address, "a", b.cfg.Address, "address and port to run server")
	flag.StringVar(&b.cfg.AccrualAddress, "r", b.cfg.AccrualAddress, "accrual system address and port")
	flag.StringVar(&b.cfg.Database, "d", b.cfg.Database, "database DSN")
	flag.StringVar(&b.cfg.JwtSecret, "jwt", b.cfg.JwtSecret, "JWT Secret")
	flag.StringVar(&b.cfg.Redis, "redis", b.cfg.Redis, "Redis connection string")
	flag.StringVar(&b.cfg.Kafka, "kafka", b.cfg.Kafka, "Kafka connection string")
	flag.Parse()

	return b
}

// Build returns the final configuration.
func (b *Builder) Build() *Config {
	if !strings.HasPrefix(b.cfg.AccrualAddress, "http://") && !strings.HasPrefix(b.cfg.AccrualAddress, "https://") {
		b.cfg.AccrualAddress = "http://" + b.cfg.AccrualAddress
	}

	return b.cfg
}
