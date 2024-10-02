package config

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sync"
	"time"

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
	Log           LogConfig      `yaml:"log" json:"-"`
	HTTPAPIServer HTTPAPIServer  `yaml:"http_api_server" json:"-"`
	DNSSD         DNSSD          `yaml:"dns_sd" json:"-"`
	Storage       StorageConfig  `yaml:"storage" json:"-"`
	Pipeline      PipelineConfig `yaml:"pipeline" json:"-"`
	OpenAI        *OpenAIConfig  `yaml:"openai" json:"-"`

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
	sync.Mutex

	CredentialsPath string `yaml:"openai_credentials_path" json:"-"`
	Token           string `yaml:"-" json:"-"`
	Client          *openai.Client
}

func (o *OpenAIConfig) SetClient(client *openai.Client) {
	o.Lock()
	defer o.Unlock()

	o.Client = client
}

func (o *OpenAIConfig) GetClient() *openai.Client {
	o.Lock()
	defer o.Unlock()

	return o.Client
}

type BlockConfig struct {
	Detector    BlockConfigDetector    `yaml:"detector" json:"-"`
	Reliability BlockConfigReliability `yaml:"reliability" json:"-"`
	Config      map[string]interface{} `yaml:"config" json:"-"`
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

	openAIConfig := &OpenAIConfig{}
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

		openAIConfig.CredentialsPath = config.OpenAI.CredentialsPath

		token := openAIToken{}
		if err = json.Unmarshal(file, &token); err != nil {
			panic(err)
		}

		openAIConfig.Token = token.Token
		openAIConfig.Client = openai.NewClient(token.Token)
	}
	config.OpenAI = openAIConfig

	if httpAPIPort != nil {
		config.HTTPAPIServer.Port = *httpAPIPort
		config.DNSSD.ServicePort = *httpAPIPort
	}

	return config
}
