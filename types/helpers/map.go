package helpers

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v2"

	gjm "github.com/firewut/go-json-map"
)

func GetValue[T any](m map[string]interface{}, key string) (T, error) {
	// Does not work with array of arrays, e.g. [][]byte"
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

func MapToJSONStruct(data map[string]interface{}, result interface{}) {
	bytes, _ := json.Marshal(data)
	json.Unmarshal(bytes, result)
}
