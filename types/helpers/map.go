package helpers

import (
	"fmt"

	gjm "github.com/firewut/go-json-map"
	"gopkg.in/yaml.v2"
)

func GetValue[T any](m map[string]interface{}, key string) (T, error) {
	value, err := gjm.GetProperty(m, key)
	if err != nil {
		return *new(T), err
	}

	// Attempt to cast the value to the expected type T
	if v, ok := value.(T); ok {
		return v, nil
	}

	return *new(T), fmt.Errorf("value for key '%s' cannot be cast to %T", key, new(T))
}

func MapToYAMLStruct(data map[string]interface{}, result interface{}) {
	bytes, _ := yaml.Marshal(data)
	yaml.Unmarshal(bytes, result)
}
