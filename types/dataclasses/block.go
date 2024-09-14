package dataclasses

import (
	"bytes"
	"fmt"
	"sync"

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

func (b *BlockData) GetInputConfig() map[string]interface{} {
	b.Lock()
	defer b.Unlock()

	return b.InputConfig
}

func (b *BlockData) GetInputDataFromConfig(
	pipelineResults map[string][]*bytes.Buffer,
) []map[string]interface{} {
	inputData := make([]map[string]interface{}, 0)

	if b.GetInputConfig() == nil {
		return inputData
	}

	b.Lock()
	defer b.Unlock()

	b.Input = make(map[string]interface{})

	// Check if `property` in `input_config` is not empty
	if _, ok := b.InputConfig["property"]; !ok {
		return inputData
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
						if results, ok := pipelineResults[origin]; ok {
							for _, result := range results {
								inputData = append(
									inputData, map[string]interface{}{
										property: result.String(),
									},
								)
							}
						}
					}
				}
			}
		default:
			continue
		}
	}

	return inputData
}
