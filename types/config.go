package types

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Log           LogConfig     `yaml:"log" json:"-"`
	HTTPAPIServer HTTPAPIServer `yaml:"http_api_server" json:"-"`
	DNSSD         DNSSD         `yaml:"dns_sd" json:"-"`
}

type LogConfig struct {
	Level string `yaml:"level"  json:"-"`
}

type HTTPAPIServer struct {
	Host string `yaml:"host" json:"-"`
	Port int    `yaml:"port" json:"-"`
}

type DNSSD struct {
	ServiceName   string `yaml:"service_name" json:"-"`
	ServiceType   string `yaml:"service_type" json:"-"`
	ServiceDomain string `yaml:"service_domain" json:"-"`
	ServicePort   int    `yaml:"service_port" json:"-"`
	Version       string `yaml:"version" json:"-"`
	Load          string `yaml:"load" json:"-"`
	Available     bool   `yaml:"available" json:"-"`
}

func NewConfig() Config {
	config := Config{}

	var configPath string
	configPath = os.Getenv("CONFIG_FILE")
	if configPath == "" {
		configPath = CONFIG_FILE
	}

	file, err := os.Open(configPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	d := yaml.NewDecoder(file)
	if err := d.Decode(&config); err != nil {
		panic(err)
	}

	return config
}
