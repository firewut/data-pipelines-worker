package dataclasses

import (
	"fmt"

	"data-pipelines-worker/types/interfaces"
)

type BlockData struct {
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
	b.pipeline = pipeline
}

func (b *BlockData) GetPipeline() interfaces.Pipeline {
	return b.pipeline
}

func (b *BlockData) SetBlock(block interfaces.Block) {
	b.block = block
}

func (b *BlockData) GetBlock() interfaces.Block {
	return b.block
}

func (b *BlockData) GetData() interface{} {
	return b
}

func (b *BlockData) SetData(interface{}) {
	// Do nothing
}

func (b *BlockData) SetInputData(inputData interface{}) {
	b.Input = inputData.(map[string]interface{})
}

func (b *BlockData) GetInputData() interface{} {
	return b.Input
}

func (b *BlockData) GetStringRepresentation() string {
	return fmt.Sprintf(
		"BlockData{Id: %s, Slug: %s}",
		b.Id, b.Slug,
	)
}

func (b *BlockData) GetId() string {
	return b.Id
}

func (b *BlockData) GetSlug() string {
	return b.Slug
}
