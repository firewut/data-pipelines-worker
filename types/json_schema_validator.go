package types

import "github.com/xeipuuv/gojsonschema"

type JSONSchemaValidator struct {
}

func (b *JSONSchemaValidator) Validate(schemaString string) (*gojsonschema.Schema, interface{}, error) {
	gojsonschema.FormatCheckers.Add("file", FileFormatChecker{})

	schemaLoader := gojsonschema.NewStringLoader(schemaString)
	schemaPtr, err := gojsonschema.NewSchema(schemaLoader)
	schema, _ := schemaLoader.LoadJSON()

	return schemaPtr, schema, err
}
