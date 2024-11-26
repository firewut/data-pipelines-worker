package dataclasses

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
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

	index    int
	pipeline interfaces.Pipeline
	block    interfaces.Block
}

type BlockInputData struct {
	Condition bool
	Value     interface{}
}

func (b *BlockData) Clone() interfaces.ProcessableBlockData {
	return &BlockData{
		Id:           b.Id,
		Slug:         b.Slug,
		Description:  b.Description,
		InputConfig:  b.InputConfig,
		Input:        b.Input,
		OutputConfig: b.OutputConfig,
		index:        b.index,
		pipeline:     b.pipeline,
		block:        b.block,
	}
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

func (b *BlockData) GetInputIndex() int {
	b.Lock()
	defer b.Unlock()

	return b.index
}
func (b *BlockData) SetInputIndex(index int) {
	b.Lock()
	defer b.Unlock()

	b.index = index
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
) ([]map[string]interface{}, bool, bool, error) {
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
	//
	//  property
	//  	array_input = true
	//      	the function assigns an array to the property

	inputData := make([]map[string]interface{}, 0)
	inputTypeArrayParallel := false

	var schema map[string]interface{}
	err := json.Unmarshal([]byte(b.GetBlock().GetSchemaString()), &schema)
	if err != nil {

		return nil, false, false, err
	}

	if b.GetInputConfig() == nil {
		return inputData, false, false, nil
	}

	b.Lock()
	defer b.Unlock()

	// Check if `property` in `input_config` is not empty
	if _, ok := b.InputConfig["property"]; !ok {
		return inputData, false, false, nil
	}

	// Check if `type` in `input_config` is not empty
	inputTypeArray := false
	if inputType, ok := b.InputConfig["type"]; ok {
		switch inputType {
		case "array":
			inputTypeArray = true
			if inputTypeParallel, ok := b.InputConfig["parallel"]; ok {
				inputTypeArrayParallel = inputTypeParallel.(bool)
			}
		default:
		}
	}

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
						arrayInput := false
						if _, ok := property_config.(map[string]interface{})["array_input"]; ok {
							arrayInput = property_config.(map[string]interface{})["array_input"].(bool)
						}

						if results, ok := pipelineResults[origin]; ok {
							for _, resultValue := range results {
								var rawValue interface{}

								rawValue = resultValue
								if arrayInput {
									rawValue = results
								}

								valueCasted, err := helpers.CastPropertyData(
									property,
									rawValue,
									schema,
								)

								if err != nil {
									// TODO: must be rawValue
									valueCasted = resultValue.String()
								}

								originData := map[string]interface{}{
									property: valueCasted,
								}

								// jsonPath same level as `origin`
								if jsonPath, ok := property_config.(map[string]interface{})["json_path"].(string); ok {
									var data interface{}
									if arrayInput {
										if reflect.TypeOf(valueCasted).Kind() == reflect.Slice && err == nil {
											dataSlice := make([]interface{}, 0)

											val := reflect.ValueOf(valueCasted)
											for i := 0; i < val.Len(); i++ {
												item := val.Index(i).Interface()

												// Convert item to *bytes.Buffer if possible
												var buffer *bytes.Buffer
												switch v := item.(type) {
												case []byte:
													buffer = bytes.NewBuffer(v)
												case string:
													buffer = bytes.NewBufferString(v)
												default:
													continue
												}

												if dataItem, err := HandleResultValue(buffer.Bytes()); err == nil {
													dataSlice = append(dataSlice, dataItem)
												}

											}
											data = dataSlice

										} else {
											data, err = HandleResultValue(resultValue.Bytes())
											if err != nil {
												return nil, inputTypeArray, false, err
											}
										}
									} else {
										data, err = HandleResultValue(resultValue.Bytes())
										if err != nil {
											return nil, inputTypeArray, false, err
										}
									}

									jsonPathData, err := jsonpath.JsonPathLookup(data, jsonPath)
									if err != nil {
										return nil, inputTypeArray, false, err
									}
									if inputTypeArray && reflect.TypeOf(jsonPathData).Kind() == reflect.Slice {
										if _, ok := jsonPathData.([]interface{}); ok {
											originData = make(map[string]interface{})

											for _, data := range jsonPathData.([]interface{}) {
												inputData = append(
													inputData, map[string]interface{}{
														property: data,
													},
												)
											}
										}
									} else {
										originData = map[string]interface{}{
											property: jsonPathData,
										}
									}
								}

								inputData = append(
									inputData, originData,
								)
							}
						} else {
							return inputData, inputTypeArray, false, fmt.Errorf(
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
		inputData = MergeMaps(inputData)
	} else {
		inputDataNotArray := make(map[string]interface{})
		for _, data := range inputData {
			for key, value := range data {
				if _, ok := inputDataNotArray[key]; !ok {
					inputDataNotArray[key] = value
				}
			}
		}

		return []map[string]interface{}{inputDataNotArray}, inputTypeArray, false, nil
	}

	return inputData, inputTypeArray, inputTypeArrayParallel, nil
}

// MergeMaps function definition remains unchanged
func MergeMaps(maps []map[string]interface{}) []map[string]interface{} {
	if len(maps) == 0 {
		return nil
	}

	var result []map[string]interface{}

	// Recursively merge two maps
	merge := func(base, current map[string]interface{}) map[string]interface{} {
		merged := make(map[string]interface{})

		// Start by copying the base map into the merged map
		for key, value := range base {
			merged[key] = value
		}

		// Recursively merge in the current map
		for key, value := range current {
			// Handle byte slice comparison
			if existingVal, exists := merged[key]; exists {
				switch existingVal := existingVal.(type) {
				case []byte:
					if currentVal, ok := value.([]byte); ok {
						// Compare byte slices
						if !bytes.Equal(existingVal, currentVal) {
							merged[key] = value // Update value if they are different
						}
					} else {
						merged[key] = value // Update value if not []byte
					}
				default:
					merged[key] = value // Update value if type is different
				}
			} else {
				merged[key] = value
			}
		}

		return merged
	}

	for _, currentMap := range maps {
		shouldMerge := false

		// Check if current map can be merged with any in the result
		for i, resMap := range result {
			conflictingKeys := false

			for key, val := range currentMap {
				if existingVal, exists := resMap[key]; exists {
					switch existingVal := existingVal.(type) {
					case []byte:
						if currentVal, ok := val.([]byte); ok {
							if !bytes.Equal(existingVal, currentVal) {
								conflictingKeys = true
								break
							}
						} else {
							conflictingKeys = true
							break
						}
					default:
						// Use reflect.DeepEqual to compare complex types safely
						if !reflect.DeepEqual(existingVal, val) {
							conflictingKeys = true
							break
						}
					}
				}
			}

			// If no conflicts, merge the maps
			if !conflictingKeys {
				result[i] = merge(resMap, currentMap)
				shouldMerge = true
				break
			}
		}

		// If no mergeable map found, append a new map
		if !shouldMerge {
			newMap := make(map[string]interface{})

			// Inherit fields from the last map in result (if any)
			if len(result) > 0 {
				newMap = merge(result[len(result)-1], currentMap)
			} else {
				// If result is empty, just copy the current map
				newMap = merge(newMap, currentMap)
			}

			result = append(result, newMap)
		}
	}

	return result
}

func HandleResultValue(resultValue []byte) (interface{}, error) {
	// Convert to string for further checks
	strValue := string(resultValue)

	// Check if the resultValue is a potential JSON (starts with '{', '[', '"', etc.)
	strValue = strings.TrimSpace(strValue) // Remove surrounding whitespaces
	if len(strValue) > 0 && (strValue[0] == '{' || strValue[0] == '[' || strValue[0] == '"') {
		var data interface{}
		// Attempt to unmarshal as JSON
		err := json.Unmarshal(resultValue, &data)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
		}
		return data, nil
	}

	return strValue, nil
}
