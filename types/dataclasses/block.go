package dataclasses

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/oliveagle/jsonpath"

	"data-pipelines-worker/types/helpers"
	"data-pipelines-worker/types/interfaces"
)

type BlockData struct {
	sync.Mutex

	Id           string                 `json:"id"`
	Slug         string                 `json:"slug"`
	Description  string                 `json:"description"`
	InputConfig  map[string]interface{} `json:"input_config"`
	Input        map[string]interface{} `json:"input"`
	OutputConfig map[string]interface{} `json:"output_config"`

	pipeline interfaces.Pipeline
	block    interfaces.Block
}

type BlockInputData struct {
	Condition bool
	Value     interface{}
}

func (b *BlockData) SetPipeline(pipeline interfaces.Pipeline) {
	b.Lock()
	defer b.Unlock()

	b.pipeline = pipeline
}

func (b *BlockData) GetPipeline() interfaces.Pipeline {
	b.Lock()
	defer b.Unlock()

	return b.pipeline
}

func (b *BlockData) SetBlock(block interfaces.Block) {
	b.Lock()
	defer b.Unlock()

	b.block = block
}

func (b *BlockData) GetBlock() interfaces.Block {
	b.Lock()
	defer b.Unlock()

	return b.block
}

func (b *BlockData) GetData() interface{} {
	b.Lock()
	defer b.Unlock()

	return b
}

func (b *BlockData) SetData(interface{}) {
	// Do nothing
}

func (b *BlockData) SetInputData(inputData interface{}) {
	b.Lock()
	defer b.Unlock()

	b.Input = inputData.(map[string]interface{})
}

func (b *BlockData) GetInputData() interface{} {
	b.Lock()
	defer b.Unlock()

	return b.Input
}

func (b *BlockData) GetStringRepresentation() string {
	b.Lock()
	defer b.Unlock()

	return fmt.Sprintf(
		"BlockData{Id: %s, Slug: %s}",
		b.Id, b.Slug,
	)
}

func (b *BlockData) GetId() string {
	b.Lock()
	defer b.Unlock()

	return b.Id
}

func (b *BlockData) GetSlug() string {
	b.Lock()
	defer b.Unlock()

	return b.Slug
}

func (b *BlockData) GetInput() map[string]interface{} {
	b.Lock()
	defer b.Unlock()

	return b.Input
}

func (b *BlockData) GetInputConfig() map[string]interface{} {
	b.Lock()
	defer b.Unlock()

	return b.InputConfig
}

func (b *BlockData) GetInputDataByPriority(
	blockInputDataConfig []interface{},
) []map[string]interface{} {
	result := make([]map[string]interface{}, 0)
	for _, item := range blockInputDataConfig {
		if blockInput, ok := item.(BlockInputData); ok {
			if blockInput.Condition {
				mapInput := blockInput.Value.([]map[string]interface{})
				for _, mappedInput := range mapInput {
					for key, value := range mappedInput {
						// Check if key exists in result
						if len(result) > 0 {
							for _, resultValue := range result {
								if _, ok := resultValue[key]; !ok {
									resultValue[key] = value
								}
							}
						} else {
							result = append(result, mappedInput)
						}
					}
				}
			}
		}
	}

	return result
}

func (b *BlockData) CastValueToValidType(
	value *bytes.Buffer,
	types []string,
) interface{} {

	return value.String()
}

func (b *BlockData) GetInputConfigData(
	pipelineResults map[string][]*bytes.Buffer,
) ([]map[string]interface{}, error) {
	//  input_config.type = "array"
	//      the function returns an array of maps
	//          []map[string]interface{}{
	//              map[string]interface{}{
	//    	            "url": "https://localhost:8080",
	//    	            "body": "hello 1",
	//    	        },
	//    	        map[string]interface{}{
	//    	            "url": "https://localhost:8081",
	//    	            "body": "hello 2",
	//    	        }
	//          }
	//  input_config.type != "array"
	//      the function returns an array of maps with one property joined
	//          []map[string]interface{}{
	//              map[string]interface{}{
	//    	            "url": "https://localhost:8080",
	//    	            "body": "hello 1",
	//    	        }
	//          }

	inputData := make([]map[string]interface{}, 0)

	var schema map[string]interface{}
	err := json.Unmarshal([]byte(b.GetBlock().GetSchemaString()), &schema)
	if err != nil {
		fmt.Println("Error unmarshaling schema:", err)
		return nil, err
	}

	if b.GetInputConfig() == nil {
		return inputData, nil
	}

	b.Lock()
	defer b.Unlock()

	// b.Input = make(map[string]interface{})

	// Check if `property` in `input_config` is not empty
	if _, ok := b.InputConfig["property"]; !ok {
		return inputData, nil
	}

	// Check if `type` in `input_config` is not empty
	inputTypeArray := false
	if inputType, ok := b.InputConfig["type"]; ok {
		switch inputType {
		case "array":
			inputTypeArray = true
		default:
		}
	}
	_ = inputTypeArray

	for property, property_config := range b.InputConfig {
		if property_config == nil {
			continue
		}

		// switch property config key
		switch property {
		case "property":
			if propertyData, ok := property_config.(map[string]interface{}); ok {
				// propertyData is map[url:map[origin:test-block-first-slug]]
				for property, property_config := range propertyData {
					if property_config == nil {
						continue
					}

					// Check origin in PropertyData exists in pipelineResults
					if origin, ok := property_config.(map[string]interface{})["origin"].(string); ok {
						if results, ok := pipelineResults[origin]; ok {
							for _, resultValue := range results {
								valueCasted, err := helpers.CastPropertyData(
									property,
									resultValue,
									schema,
								)
								if err != nil {
									valueCasted = resultValue.String()
								}

								originData := map[string]interface{}{
									property: valueCasted,
								}

								// jsonPath same level as `origin`
								if jsonPath, ok := property_config.(map[string]interface{})["jsonPath"].(string); ok {
									var data interface{}
									err := json.Unmarshal(resultValue.Bytes(), &data)
									if err != nil {
										fmt.Println("Error unmarshaling JSON:", err)
										return nil, err
									}

									jsonPathData, err := jsonpath.JsonPathLookup(data, jsonPath)
									if err != nil {
										return nil, err
									}

									originData = map[string]interface{}{
										property: jsonPathData,
									}
								}

								inputData = append(
									inputData, originData,
								)
							}
						} else {
							return inputData, fmt.Errorf(
								"origin %s not found in pipelineResults",
								origin,
							)
						}
					}
				}
			}
		default:
			continue
		}
	}

	if inputTypeArray {
		inputData = mergeMaps(inputData)
	} else {
		inputDataNotArray := make(map[string]interface{})
		for _, data := range inputData {
			for key, value := range data {
				if _, ok := inputDataNotArray[key]; !ok {
					inputDataNotArray[key] = value
				}
			}
		}

		return []map[string]interface{}{inputDataNotArray}, nil
	}

	return inputData, nil
}

func mergeMaps(maps []map[string]interface{}) []map[string]interface{} {
	if len(maps) == 0 {
		return nil
	}

	// Create a map to group values by keys
	grouped := make(map[string][]interface{})

	// Collect values by keys
	for _, m := range maps {
		for key, value := range m {
			grouped[key] = append(grouped[key], value)
		}
	}

	// Determine the number of result maps
	numResults := 0
	for _, values := range grouped {
		if numResults < len(values) {
			numResults = len(values)
		}
	}

	// Create result maps by combining values at each index
	results := make([]map[string]interface{}, numResults)
	for i := 0; i < numResults; i++ {
		resultMap := make(map[string]interface{})
		for key, values := range grouped {
			if i < len(values) {
				resultMap[key] = values[i]
			}
		}
		results[i] = resultMap
	}

	return results
}
