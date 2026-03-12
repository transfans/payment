package config

import "github.com/caarlos0/env/v11"

type Config struct {
	Port               string `env:"PORT"                 envDefault:"8004"`
	DatabaseURL        string `env:"DATABASE_URL,required"`
	SharedJWTSecret    string `env:"SHARED_JWT_SECRET,required"`
	InternalSecret     string `env:"INTERNAL_SECRET,required"`
	ProfileServiceURL  string `env:"PROFILE_SERVICE_URL"  envDefault:"http://profile:8002"`
}

func Load() (Config, error) {
	var cfg Config
	return cfg, env.Parse(&cfg)
}
