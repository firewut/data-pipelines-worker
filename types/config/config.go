package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
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

func GetConfig(forceNewInstance ...bool) Config {
	if len(forceNewInstance) > 0 && forceNewInstance[0] {
		newInstance := NewConfig()
		config = newInstance
		onceConfig = sync.Once{}
		return newInstance
	}

	onceConfig.Do(func() {
		config = NewConfig()
	})

	return config
}

type Config struct {
	Log           LogConfig       `yaml:"log" json:"-"`
	HTTPAPIServer HTTPAPIServer   `yaml:"http_api_server" json:"-"`
	DNSSD         DNSSD           `yaml:"dns_sd" json:"-"`
	Storage       StorageConfig   `yaml:"storage" json:"-"`
	Pipeline      PipelineConfig  `yaml:"pipeline" json:"-"`
	OpenAI        *OpenAIConfig   `yaml:"openai" json:"-"`
	Telegram      *TelegramConfig `yaml:"telegram" json:"-"`

	Blocks map[string]BlockConfig `yaml:"blocks" json:"-"`
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
	Local LocalStorageConfig `yaml:"local" json:"-"`
	Minio MinioStorageConfig `yaml:"minio" json:"-"`
}

type LocalStorageConfig struct {
	RootPath string `yaml:"root_path" json:"-"`
}

type MinioStorageConfig struct {
	CredentialsPath string `yaml:"credentials_path" json:"-"`
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
	ClientConfig[*openai.Client]

	CredentialsPath string `yaml:"credentials_path" json:"-"`
	EnvVarName      string `yaml:"env_var_name" json:"-"`
	Token           string `yaml:"-" json:"-"`
}

type TelegramConfig struct {
	ClientConfig[*tgbotapi.BotAPI]

	CredentialsPath string `yaml:"credentials_path" json:"-"`
	EnvVarName      string `yaml:"env_var_name" json:"-"`
	Token           string `yaml:"-" json:"-"`
	BotName         string `yaml:"bot_name" json:"bot_name"`
}

type BlockConfig struct {
	Detector          BlockConfigDetector    `yaml:"detector" json:"-"`
	Reliability       BlockConfigReliability `yaml:"reliability" json:"-"`
	ParallelAvailable bool                   `yaml:"parallel_available" json:"-"`
	Config            map[string]interface{} `yaml:"config" json:"-"`
}

type BlockConfigDetector struct {
	CheckInterval time.Duration          `yaml:"check_interval" json:"-"`
	Conditions    map[string]interface{} `yaml:"conditions" json:"-"`
}

type BlockConfigReliability struct {
	Policy       string      `yaml:"policy" json:"-"`
	PolicyConfig interface{} `yaml:"-" json:"-"`
}

func (b *BlockConfigReliability) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type blockConfigReliability BlockConfigReliability // create a new type to avoid recursion
	var tmp blockConfigReliability
	if err := unmarshal(&tmp); err != nil {
		return err
	}

	switch tmp.Policy {
	case "exponential_backoff":
		var policyConfig BlockConfigReliabilityExponentialBackoff
		if err := unmarshal(&policyConfig); err != nil {
			return err
		}
		b.PolicyConfig = policyConfig
	}

	b.Policy = tmp.Policy
	return nil
}

type BlockConfigReliabilityExponentialBackoff struct {
	MaxRetries int   `yaml:"max_retries" json:"-"`
	RetryDelay int   `yaml:"retry_delay" json:"-"`
	RetryCodes []int `yaml:"retry_codes" json:"-"`
}

func NewConfig() Config {
	godotenv.Load()

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

	if config.Storage.Minio.CredentialsPath != "" {
		credentailsPath := config.Storage.Minio.CredentialsPath
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

		minioStorageConfig := MinioStorageConfig{
			CredentialsPath: config.Storage.Minio.CredentialsPath,
		}
		if err = json.Unmarshal(file, &minioStorageConfig); err != nil {
			panic(err)
		}

		config.Storage.Minio = minioStorageConfig
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

	// Initialize OpenAI client
	openAIConfig := config.OpenAI

	err = initializeClient(
		&openAIConfig.ClientConfig,
		config.OpenAI.EnvVarName,
		configPath,
		config.OpenAI.CredentialsPath,
		func(token string) (*openai.Client, error) {
			openAIConfig.Token = token
			return openai.NewClient(token), nil
		},
	)
	if err != nil {
		// Handle error appropriately
		fmt.Printf(
			"Failed to initialize OpenAI client: %v\n",
			err,
		)
	}

	// Initialize Telegram client
	telegramConfig := config.Telegram

	err = initializeClient(
		&telegramConfig.ClientConfig,
		config.Telegram.EnvVarName,
		configPath,
		config.Telegram.CredentialsPath,
		func(token string) (*tgbotapi.BotAPI, error) {
			if token == "" {
				return nil, fmt.Errorf("empty token")
			}
			telegramConfig.Token = token
			return tgbotapi.NewBotAPI(token)
		},
	)
	if err != nil {
		// Handle error appropriately
		fmt.Printf(
			"Failed to initialize Telegram client: %v\n",
			err,
		)
	}

	if httpAPIPort != nil {
		config.HTTPAPIServer.Port = *httpAPIPort
		config.DNSSD.ServicePort = *httpAPIPort
	}

	return config
}
