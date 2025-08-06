package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"log"
	"os"
	"time"
)

type Config struct {
	Env         string `yaml:"env" env-default:"local"`
	StoragePath string `yaml:"storage_path" env-required:"true"`
	//LogPath     string `yaml:"log_path" env-required:"false"` TODO(): add service -> send logs to mongo
	MigrationsPath string `yaml:"migrations_path" env-required:"true"`
	Mongo          `yaml:"mongo"`
	JwtConfig      `yaml:"jwt"`
	HttpServer     `yaml:"http_server"`
}

type HttpServer struct {
	Address     string        `yaml:"address" env-default:":9010"`
	Timeout     time.Duration `yaml:"timeout" env-default:"4s"`
	IdleTimeout time.Duration `yaml:"idle-timeout" env-default:"60s"`
}

type JwtConfig struct {
	Secret         string        `yaml:"secret" env-required:"true"`
	AccessTimeout  time.Duration `yaml:"access_timeout" env-default:"1h"`
	RefreshTimeout time.Duration `yaml:"refresh_timeout" env-default:"6h"`
}

type Mongo struct {
	Host       string `yaml:"host" env-required:"true"`
	Port       int    `yaml:"port" env-required:"true"`
	Username   string `yaml:"username" env-required:"true"`
	Password   string `yaml:"password" env-required:"true"`
	AuthSource string `yaml:"auth_source" env-required:"true"`
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		log.Fatal("CONFIG_PATH is not set")
	}

	// check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("CONFIG_PATH does not exist: %s", configPath)
	}

	var config Config

	if err := cleanenv.ReadConfig(configPath, &config); err != nil {
		log.Fatalf("Cannot read config: %s", err)
	}

	return &config
}
