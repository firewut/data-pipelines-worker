package config

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sync"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v2"
)

const (
	CONFIG_FILE             = "config/config.yaml"
	PIPELINES_CATALOGUE_DIR = "config/pipelines"
)

var (
	config     Config
	onceConfig sync.Once
)

type Config struct {
	Log           LogConfig      `yaml:"log" json:"-"`
	HTTPAPIServer HTTPAPIServer  `yaml:"http_api_server" json:"-"`
	DNSSD         DNSSD          `yaml:"dns_sd" json:"-"`
	Storage       StorageConfig  `yaml:"storage" json:"-"`
	Pipeline      PipelineConfig `yaml:"pipeline" json:"-"`
	OpenAI        OpenAIConfig   `yaml:"openai" json:"-"`
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

type PipelineConfig struct {
	StoragePath string               `yaml:"pipeline_validation_schema_path" json:"-"`
	Catalogue   string               `yaml:"pipeline_catalogue" json:"-"`
	SchemaPtr   *gojsonschema.Schema `yaml:"-" json:"-"`
}

type openAIToken struct {
	Token string `json:"token"`
}

type OpenAIConfig struct {
	CredentialsPath string `yaml:"openai_credentials_path" json:"-"`
	Token           string `yaml:"-" json:"-"`
}

func NewConfig() Config {
	config := Config{}

	httpAPIPort := flag.Int("http-api-port", 8080, "HTTP API port")

	flag.Parse()

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
			// Fallback - check credentials file in the same directory as the config file
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

	if config.Pipeline.StoragePath != "" {
		pipelineConfigPath := config.Pipeline.StoragePath
		file, err := os.ReadFile(pipelineConfigPath)

		if condition := os.IsNotExist(err); condition {
			// Fallback - check credentials file in the same directory as the config file
			configDir := filepath.Dir(configPath)
			pipelineConfigFilename := filepath.Base(pipelineConfigPath)
			pipelineConfigPath = filepath.Join(configDir, pipelineConfigFilename)

			file, err = os.ReadFile(pipelineConfigPath)
			if err != nil {
				panic(err)
			}
		}

		schemaLoader := gojsonschema.NewStringLoader(string(file))
		schemaPtr, err := gojsonschema.NewSchema(schemaLoader)
		if err != nil {
			panic(err)
		}

		if config.Pipeline.Catalogue != "" {
			// Check directory exists
			_, err := os.ReadDir(config.Pipeline.Catalogue)
			if condition := os.IsNotExist(err); condition {
				configDir := filepath.Dir(configPath)

				config.Pipeline.Catalogue = filepath.Join(
					configDir, filepath.Base(
						PIPELINES_CATALOGUE_DIR,
					),
				)
			}
		}

		pipelineConfig := PipelineConfig{
			StoragePath: config.Pipeline.StoragePath,
			Catalogue:   config.Pipeline.Catalogue,
			SchemaPtr:   schemaPtr,
		}

		config.Pipeline = pipelineConfig
	}

	if config.OpenAI.CredentialsPath != "" {
		credentailsPath := config.OpenAI.CredentialsPath
		file, err := os.ReadFile(credentailsPath)

		if condition := os.IsNotExist(err); condition {
			// Fallback - check credentials file in the same directory as the config file
			configDir := filepath.Dir(configPath)
			credentialsFilename := filepath.Base(credentailsPath)
			credentailsPath = filepath.Join(configDir, credentialsFilename)

			file, err = os.ReadFile(credentailsPath)
			if err != nil {
				panic(err)
			}
		}

		openAIConfig := OpenAIConfig{
			CredentialsPath: config.OpenAI.CredentialsPath,
		}

		token := openAIToken{}
		if err = json.Unmarshal(file, &token); err != nil {
			panic(err)
		}

		openAIConfig.Token = token.Token
		config.OpenAI = openAIConfig
	}

	if httpAPIPort != nil {
		config.HTTPAPIServer.Port = *httpAPIPort
		config.DNSSD.ServicePort = *httpAPIPort
	}

	return config
}

func GetConfig() Config {
	onceConfig.Do(func() {
		config = NewConfig()
	})
	return config
}
