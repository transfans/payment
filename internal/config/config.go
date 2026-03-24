package config

import "github.com/caarlos0/env/v11"

type Config struct {
	Port              string `env:"PORT"                 envDefault:"8004"`
	DatabaseURL       string `env:"DATABASE_URL,required"`
	SharedJWTSecret   string `env:"SHARED_JWT_SECRET,required"`
	InternalSecret    string `env:"INTERNAL_SECRET,required"`
	ProfileServiceURL string `env:"PROFILE_SERVICE_URL,required"`
	RabbitMQURL       string `env:"RABBITMQ_URL,required"`
}

func Load() (Config, error) {
	var cfg Config
	return cfg, env.Parse(&cfg)
}
