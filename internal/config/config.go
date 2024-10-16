package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	_ "github.com/ilyakaznacheev/cleanenv"
	"log"
	"os"
	"time"
)

type Config struct {
	Environment string     `yaml:"env"`
	StoragePath string     `yaml:"storage_path"`
	HTTPServer  HTTPServer `yaml:"http_server"`
}

type HTTPServer struct {
	Address     string        `yaml:"address"`
	Timeout     time.Duration `yaml:"timeout"`
	IdleTimeout time.Duration `yaml:"idle_timeout"`
	User        string        `yaml:"user"`
	Password    string        `yaml:"password" env:"HTTP_SERVER_PASSWORD"`
}

func InitConfig() *Config {
	var cfg Config

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		log.Fatal("CONFIG_PATH environment variable not set")
	}

	_, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		log.Fatalf("config file does not exist: %s", configPath)
	}

	err = cleanenv.ReadConfig(configPath, &cfg)
	if err != nil {
		log.Fatalf("cannot read config: %s", err)
	}
	return &cfg
}
