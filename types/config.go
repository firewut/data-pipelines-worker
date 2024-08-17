package types

import (
	"encoding/json"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Log           LogConfig     `yaml:"log" json:"-"`
	HTTPAPIServer HTTPAPIServer `yaml:"http_api_server" json:"-"`
	DNSSD         DNSSD         `yaml:"dns_sd" json:"-"`
	Storage       StorageConfig `yaml:"storage" json:"-"`
}

type LogConfig struct {
	Level string `yaml:"level"  json:"-"`
}

type HTTPAPIServer struct {
	Host string `yaml:"host" json:"-"`
	Port int    `yaml:"port" json:"-"`
}

type DNSSD struct {
	ServiceName   string  `yaml:"service_name" json:"-"`
	ServiceType   string  `yaml:"service_type" json:"-"`
	ServiceDomain string  `yaml:"service_domain" json:"-"`
	ServicePort   int     `yaml:"service_port" json:"-"`
	Version       string  `yaml:"version" json:"-"`
	Load          float32 `yaml:"load" json:"-"`
	Available     bool    `yaml:"available" json:"-"`
}

type StorageConfig struct {
	CredentialsPath string `yaml:"storage_credentials_path" json:"-"`
	Bucket          string `yaml:"bucket" json:"bucket"`
	AccessKey       string `yaml:"accessKey" json:"accessKey"`
	Api             string `yaml:"api" json:"api"`
	Path            string `yaml:"path" json:"path"`
	SecretKey       string `yaml:"secretKey" json:"secretKey"`
	Url             string `yaml:"url" json:"url"`
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

	if config.Storage.CredentialsPath != "" {
		credentailsPath := config.Storage.CredentialsPath
		file, err := os.ReadFile(credentailsPath)

		if condition := os.IsNotExist(err); condition {
			// Falllback - check credentials file in the same directory as the config file
			configDir := filepath.Dir(configPath)
			credentialsFilename := filepath.Base(credentailsPath)
			credentailsPath = filepath.Join(configDir, credentialsFilename)

			file, err = os.ReadFile(credentailsPath)
			if err != nil {
				panic(err)
			}
		}

		storageConfig := StorageConfig{
			CredentialsPath: config.Storage.CredentialsPath,
		}
		if err = json.Unmarshal(file, &storageConfig); err != nil {
			panic(err)
		}

		config.Storage = storageConfig
	}

	return config
}
