package config

import (
	"time"

	"github.com/spf13/viper"
)

type Server struct {
	Port        string        `mapstructure:"port"`
	IdleTimeout time.Duration `mapstructure:"idle_timeout"`
}

type Cache struct {
	LiveTime time.Duration `mapstructure:"live_time"`
}

type Storage struct {
	Dsn  string `mapstructure:"dsn"`
	Name string `mapstructure:"name"`
}

type Config struct {
	Server  Server  `mapstructure:"server"`
	Cache   Cache   `mapstructure:"cache"`
	Storage Storage `mapstructure:"storage"`
}

func Load(path string) (Config, error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return Config{}, err
	}

	var config Config

	if err := viper.Unmarshal(&config); err != nil {
		return Config{}, err
	}

	return config, nil
}
