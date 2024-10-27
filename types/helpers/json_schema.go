package helpers

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
)

// Function to extract type and format from the JSON schema
func extractTypeAndFormat(propMap map[string]interface{}) (string, string, error) {
	var propType string
	var propFormat string

	if t, exists := propMap["type"]; exists {
		switch v := t.(type) {
		case string:
			propType = v
		case []interface{}:
			propType = v[0].(string) // Assume at least one type is present
		}
	}

	if format, ok := propMap["format"]; ok {
		propFormat = format.(string)
	}

	if propType == "" {
		return "", "", errors.New("type is required")
	}

	return propType, propFormat, nil
}

// Function to get the array item type
func getArrayItemType(propMap map[string]interface{}) (string, error) {
	if items, ok := propMap["items"]; ok {
		itemMap, ok := items.(map[string]interface{})
		if !ok {
			return "", errors.New("items must be an object")
		}
		itemType, _, err := extractTypeAndFormat(itemMap)
		if err != nil {
			return "", err
		}
		return itemType, nil
	}
	return "", errors.New("array type must have items defined")
}

// Function to cast data to the specified type based on the JSON schema
func CastDataToType(data interface{}, schema map[string]interface{}) (interface{}, error) {
	propType, propFormat, err := extractTypeAndFormat(schema)

	if err != nil {
		return nil, err
	}

	defer func() {
		if r := recover(); r != nil {
			// Handle the panic, return an error
			err = fmt.Errorf("panic occurred: %v", r)
		}
	}()

	// Handle different formats as needed
	switch propType {
	case "string":
		if propFormat == "file" {
			// Handle special case for file format
			switch v := data.(type) {
			case *bytes.Buffer:
				return v.Bytes(), nil // Convert *bytes.Buffer to []byte for file format
			case []byte:
				return v, nil // Directly return []byte
			case string:
				return []byte(v), nil // Convert string to []byte
			}
		}
		if str, ok := data.(string); ok {
			return str, nil
		}
		return nil, errors.New("data is not a valid string")
	case "integer":
		return int(data.(float64)), nil // Assuming data is provided as float64 (common for JSON numbers)
	case "number":
		return data.(float64), nil
	case "boolean":
		return data.(bool), nil
	case "null":
		return nil, nil
	case "array":
		itemSchema, ok := schema["items"].(map[string]interface{})
		if !ok {
			return nil, errors.New("array type must have items defined")
		}

		result := []interface{}{}

		// Use reflection to check the type of data
		val := reflect.ValueOf(data)
		if val.Kind() != reflect.Slice {
			return nil, errors.New("data must be an array")
		}

		// Iterate over the elements of the slice
		for i := 0; i < val.Len(); i++ {
			item := val.Index(i).Interface() // Get the element at index i

			// Recursively cast the item using the item schema
			castedItem, err := CastDataToType(item, itemSchema)
			if err != nil {
				return nil, err
			}
			result = append(result, castedItem)
		}
		return result, nil

	case "object":
		result := make(map[string]interface{})
		dataMap, ok := data.(map[string]interface{})
		if !ok {
			return nil, errors.New("data must be an object")
		}
		for key, value := range dataMap {
			castedValue, err := CastDataToType(value, schema["properties"].(map[string]interface{})[key].(map[string]interface{}))
			if err != nil {
				return nil, err
			}
			result[key] = castedValue
		}
		return result, nil
	default:
		return nil, errors.New("unsupported type for casting")
	}
}

// Function to cast a property from data given its name
func CastPropertyData(propertyName string, data interface{}, schema map[string]interface{}) (interface{}, error) {
	// Find the property schema for the given property name
	if inputMap, ok := schema["properties"].(map[string]interface{})["input"].(map[string]interface{}); ok {
		if propMap, ok := inputMap["properties"].(map[string]interface{})[propertyName].(map[string]interface{}); ok {
			return CastDataToType(data, propMap)
		}
	}

	return nil, fmt.Errorf("property '%s' not found in schema", propertyName)
}
