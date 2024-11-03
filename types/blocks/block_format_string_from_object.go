package blocks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/generics"
	"data-pipelines-worker/types/helpers"
	"data-pipelines-worker/types/interfaces"
)

type DetectorFormatStringFromObject struct {
	BlockDetectorParent
}

func NewDetectorFormatStringFromObject(
	detectorConfig config.BlockConfigDetector,
) *DetectorFormatStringFromObject {
	return &DetectorFormatStringFromObject{
		BlockDetectorParent: NewDetectorParent(detectorConfig),
	}
}

func (d *DetectorFormatStringFromObject) Detect() bool {
	d.Lock()
	defer d.Unlock()

	return true
}

type ProcessorFormatStringFromObject struct {
}

func NewProcessorFormatStringFromObject() *ProcessorFormatStringFromObject {
	return &ProcessorFormatStringFromObject{}
}

func (p *ProcessorFormatStringFromObject) GetRetryCount(_ interfaces.Block) int {
	return 0
}

func (p *ProcessorFormatStringFromObject) GetRetryInterval(_ interfaces.Block) time.Duration {
	return 0
}

// formatString formats the template string using the provided variables map.
// It checks for unclosed or improperly formatted placeholders and returns an error if any are found.
func formatString(template string, variables map[string]interface{}) (string, error) {
	var result strings.Builder
	n := len(template)

	for i := 0; i < n; i++ {
		if template[i] == '{' {
			// Check for unclosed brace
			start := i + 1
			if start >= n {
				return "", errors.New("unclosed brace found in template")
			}

			// Find the closing brace
			for j := start; j < n; j++ {
				if template[j] == '}' {
					// Extract the variable name
					varName := template[start:j]

					// Check for double quotes in variable name
					if strings.Contains(varName, "\"") {
						return "", errors.New("variable name contains invalid double quotes")
					}

					// Get the value from the variables map
					if value, exists := variables[varName]; exists {
						switch v := value.(type) {
						case []string:
							// Handle []string (like tags)
							result.WriteString(strings.Join(v, ", "))
						default:
							// For all other types, use fmt.Sprintf to convert
							result.WriteString(fmt.Sprintf("%v", v))
						}
					}
					i = j // Move index to closing brace
					break
				}
			}
			// If no closing brace is found, return an error
			if i < n-1 && template[i] != '}' {
				return "", errors.New("unclosed brace found in template")
			}
		} else {
			// Append the current character to the result
			result.WriteByte(template[i])
		}
	}

	return result.String(), nil
}

func (p *ProcessorFormatStringFromObject) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, bool, bool, string, int, error) {
	output := &bytes.Buffer{}
	blockConfig := &BlockFormatStringFromObjectConfig{}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockFormatStringFromObject).GetBlockConfig(_config)
	userBlockConfig := &BlockFormatStringFromObjectConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	result, err := formatString(blockConfig.Template, _data)
	if err != nil {
		return nil, false, false, "", -1, err
	}

	output = bytes.NewBufferString(result)

	return output, false, false, "", -1, nil
}

type BlockFormatStringFromObjectConfig struct {
	Template string `yaml:"-" json:"template"`
}

type BlockFormatStringFromObject struct {
	generics.ConfigurableBlock[BlockFormatStringFromObjectConfig]
	BlockParent
}

var _ interfaces.Block = (*BlockFormatStringFromObject)(nil)

func (b *BlockFormatStringFromObject) GetBlockConfig(_config config.Config) *BlockFormatStringFromObjectConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockFormatStringFromObjectConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockFormatStringFromObject() *BlockFormatStringFromObject {
	block := &BlockFormatStringFromObject{
		BlockParent: BlockParent{
			Id:          "format_string_from_object",
			Name:        "Format String From Object",
			Description: "Format string from object",
			Version:     "1",
			SchemaString: `{
				"type": "object",
				"properties": {
					"input":{
						"type": "object",
						"description": "Input data",
						"properties": {
							"template": {
								"description": "Template of the input data",
								"type": "string",
								"minLength": 1
							}
						},
						"additionalProperties": true,
						"required": ["template"]
					},
					"output": {
						"description": "Formatted text",
						"type": "string"
					}
				}
			}`,
			SchemaPtr: nil,
			Schema:    nil,
		},
	}

	if err := block.ApplySchema(block.GetSchemaString()); err != nil {
		panic(err)
	}

	block.SetProcessor(NewProcessorFormatStringFromObject())

	return block
}
