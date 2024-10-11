package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ClientConfig defines the generic structure for managing clients with thread-safety
type ClientConfig[T any] struct {
	sync.Mutex
	Client T
}

// SetClient sets the client in a thread-safe way
func (c *ClientConfig[T]) SetClient(client T) {
	c.Lock()
	defer c.Unlock()

	c.Client = client
}

// GetClient retrieves the client in a thread-safe way
func (c *ClientConfig[T]) GetClient() T {
	c.Lock()
	defer c.Unlock()

	return c.Client
}

// Helper function to load token from environment variable or credentials file
func loadToken(envVarName, configPath, credentialsPath string) (string, error) {
	if envVarName != "" {
		token := os.Getenv(envVarName)
		if token != "" {
			return token, nil
		}
	}

	if credentialsPath != "" {
		file, err := os.ReadFile(credentialsPath)
		if err != nil && os.IsNotExist(err) {
			// Try fallback - same directory as config file
			configDir := filepath.Dir(configPath) // Assume configPath is available
			credentialsFilename := filepath.Base(credentialsPath)
			fallbackPath := filepath.Join(configDir, credentialsFilename)

			file, err = os.ReadFile(fallbackPath)
			if err != nil {
				return "", err
			}
		} else if err != nil {
			return "", err
		}

		var tokenStruct openAIToken
		if err := json.Unmarshal(file, &tokenStruct); err != nil {
			return "", err
		}
		return tokenStruct.Token, nil
	}

	return "", fmt.Errorf("no valid token found")
}

// Generic function to initialize the client based on the token
func initializeClient[T any](
	config *ClientConfig[T],
	envVarName,
	configPath,
	credentialsPath string,
	createClient func(token string) (T, error),
) error {
	token, err := loadToken(envVarName, configPath, credentialsPath)
	if err != nil {
		return err
	}

	client, err := createClient(token)
	if err != nil {
		return err
	}

	config.SetClient(client)
	return nil
}
