package blocks

import (
	"bytes"
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/xeipuuv/gojsonschema"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/interfaces"
	"data-pipelines-worker/types/validators"
)

type BlockDetectorParent struct {
	sync.Mutex

	Config config.BlockConfigDetector

	stopChan chan struct{} // Channel to signal the stop of the loop.
}

func NewDetectorParent(config config.BlockConfigDetector) BlockDetectorParent {
	return BlockDetectorParent{
		Config:   config,
		stopChan: make(chan struct{}),
	}
}

func (d *BlockDetectorParent) Start(
	block interfaces.Block,
	detectionFunc func() bool,
) {
	d.Lock()
	defer d.Unlock()

	ticker := time.NewTicker(d.Config.CheckInterval)

	go func(_d *BlockDetectorParent) {
		defer ticker.Stop() // Ensure the ticker is stopped when the goroutine exits.

		for {
			select {
			case <-_d.stopChan:
				return
			case <-ticker.C:
				if detectionFunc() {
					block.SetAvailable(true)
				} else {
					block.SetAvailable(false)
				}
			}
		}
	}(d)
}

func (d *BlockDetectorParent) Stop(wg *sync.WaitGroup) {
	defer wg.Done()
	close(d.stopChan)
}

type BlockParent struct {
	sync.Mutex

	Id           string               `json:"id"`
	Name         string               `json:"name"`
	Description  string               `json:"description"`
	Version      string               `json:"version"`
	SchemaString string               `json:"-"`
	SchemaPtr    *gojsonschema.Schema `json:"-"`
	Schema       interface{}          `json:"schema"`
	Available    bool                 `json:"available"`

	processor interfaces.BlockProcessor
}

func (b *BlockParent) GetId() string {
	return b.Id
}

func (b *BlockParent) GetName() string {
	return b.Name
}

func (b *BlockParent) GetDescription() string {
	return b.Description
}

func (b *BlockParent) GetVersion() string {
	return b.Version
}

func (b *BlockParent) GetSchema() *gojsonschema.Schema {
	b.Lock()
	defer b.Unlock()

	return b.SchemaPtr
}

func (b *BlockParent) ApplySchema(schemaString string) error {
	b.Lock()
	defer b.Unlock()

	validator := validators.JSONSchemaValidator{}
	if schemaString == "" {
		return fmt.Errorf("block (%s) schema is nil", b.GetId())
	}

	schemaPtr, schema, err := validator.ValidateSchema(schemaString)
	if err == nil {
		b.SchemaPtr = schemaPtr
		b.Schema = schema
	}

	return err
}

func (b *BlockParent) GetSchemaString() string {
	return b.SchemaString
}

func (b *BlockParent) ValidateSchema(v validators.JSONSchemaValidator) (*gojsonschema.Schema, interface{}, error) {
	b.Lock()
	defer b.Unlock()

	return v.ValidateSchema(b.SchemaString)
}

func (b *BlockParent) GetAvailable() bool {
	b.Lock()
	defer b.Unlock()

	return b.Available
}

func (b *BlockParent) Process(
	processor interfaces.BlockProcessor,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, error) {
	var result *bytes.Buffer = &bytes.Buffer{}

	logger := config.GetLogger()

	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("Block (%s) panic: %v", b.GetId(), r)
		}
	}()

	// Validate data against block schema
	blockSchema := b.GetSchema()
	dataLoader := gojsonschema.NewGoLoader(data)
	validationResult, err := blockSchema.Validate(dataLoader)
	if err != nil {
		logger.Errorf(
			"Block (%s) schema validation error: %v",
			b.GetId(),
			err,
		)
		return result, err
	}
	if !validationResult.Valid() {
		errStr := "Block (%s) schema is invalid for data: %s"
		for _, err := range validationResult.Errors() {
			errStr += fmt.Sprintf("\n- %s", err)
		}
		logger.Errorf(errStr, b.GetId(), data.GetStringRepresentation())
		return result, fmt.Errorf(errStr, b.GetId(), data.GetStringRepresentation())
	}

	return processor.Process(b, data)
}

func (b *BlockParent) SaveOutput(
	data interfaces.ProcessableBlockData,
	output *bytes.Buffer,
	processingId uuid.UUID,
	storage interfaces.Storage,
) (string, error) {
	// Block output is a file named:
	// <pipeline-slug>/<processing-id>/<block-slug>/output.<mimetype>

	pipeline := data.GetPipeline()
	filePath := fmt.Sprintf(
		"%s/%s/%s/output",
		pipeline.GetSlug(),
		processingId,
		data.GetSlug(),
	)

	return storage.PutObjectBytes(
		path.Join(
			storage.GetStorageDirectory(),
			filePath,
		),
		output,
		filePath,
	)
}

func (b *BlockParent) SetAvailable(available bool) {
	b.Lock()
	defer b.Unlock()

	b.Available = available
}

func (b *BlockParent) IsAvailable() bool {
	b.Lock()
	defer b.Unlock()

	return b.Available
}

func (b *BlockParent) SetProcessor(processor interfaces.BlockProcessor) {
	b.Lock()
	defer b.Unlock()

	b.processor = processor
}

func (b *BlockParent) GetProcessor() interfaces.BlockProcessor {
	b.Lock()
	defer b.Unlock()

	return b.processor
}
