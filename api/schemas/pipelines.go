package schemas

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// PipelineInputSchema represents the structure of the pipeline input object in the JSON.
// It contains information about the pipeline's slug and processing ID.
//
// swagger:model
type PipelineInputSchema struct {
	// The slug of the pipeline
	// required: true
	// example: "example-slug"
	Slug string `json:"slug"`

	// The unique processing ID associated with this pipeline
	// example: "d9b2d63d5f23e4d76b7f3f2f25d93a7a"
	ProcessingID uuid.UUID `json:"processing_id,omitempty"`
}

func (p *PipelineInputSchema) ParseForm(form map[string][]string) error {
	// Check for the `pipeline.slug` field and set it
	if slug, exists := form["pipeline.slug"]; exists && len(slug) > 0 {
		p.Slug = slug[0]
		if p.Slug == "" {
			return fmt.Errorf("pipeline.slug is required")
		}
	} else {
		return fmt.Errorf("pipeline.slug is missing")
	}

	if processingID, exists := form["pipeline.processing_id"]; exists && len(processingID) > 0 {
		parsedID, err := uuid.Parse(processingID[0])
		if err != nil {
			return fmt.Errorf("invalid pipeline.processing_id: %v", err)
		}
		p.ProcessingID = parsedID
	}

	return nil
}

// BlockInputSchema represents the structure of the block object in the JSON.
// It contains information about the block's slug, its input data, target index,
// and an optional destination slug.
//
// swagger:model
type BlockInputSchema struct {
	// The slug of the block
	// required: true
	// example: "example-block"
	Slug string `json:"slug"`

	// The input data for the block, represented as a map of key-value pairs
	// required: true
	// example: {"key1": "value1", "key2": 42}
	Input map[string]interface{} `json:"input"`

	// The target index for the block (optional)
	// If omitted or empty, it defaults to -1
	// example: 5
	TargetIndex int `json:"target_index,omitempty,string"`

	// The destination slug for the block (optional)
	// example: "destination-block"
	DestinationSlug string `json:"destination_slug,omitempty"`
}

func (b *BlockInputSchema) ParseForm(form map[string][]string, files map[string]*multipart.FileHeader) error {
	// Parse the block's slug
	if slug, exists := form["block.slug"]; exists && len(slug) > 0 {
		b.Slug = slug[0]
		if b.Slug == "" {
			return fmt.Errorf("block.slug is required")
		}
	} else {
		return fmt.Errorf("block.slug is missing")
	}

	// Parse the target index
	if targetIndex, exists := form["block.target_index"]; exists && len(targetIndex) > 0 {
		var err error
		b.TargetIndex, err = strconv.Atoi(targetIndex[0])
		if err != nil {
			return fmt.Errorf("invalid block.target_index: %v", err)
		}
	} else {
		b.TargetIndex = -1
	}

	// Parse the destination slug
	if destinationSlug, exists := form["block.destination_slug"]; exists && len(destinationSlug) > 0 {
		b.DestinationSlug = destinationSlug[0]
	}

	// Initialize Input map if nil
	if b.Input == nil {
		b.Input = make(map[string]interface{})
	}

	// Iterate through form data to process input fields
	for key, value := range form {
		// Check if the key starts with "block.input." indicating it's part of the block input
		if strings.HasPrefix(key, "block.input.") {
			fieldName := strings.TrimPrefix(key, "block.input.")

			// Check if the value is an array
			if strings.HasSuffix(fieldName, "[]") {
				baseFieldName := strings.TrimSuffix(fieldName, "[]")
				var array []string
				array = append(array, value...)
				b.Input[baseFieldName] = array
			} else {
				// Regular form field
				b.Input[fieldName] = value[0]
			}
		}
	}

	// Now handle files
	for key, fileHeader := range files {
		// Check if the file is part of "block.input." prefix
		if strings.HasPrefix(key, "block.input.") {
			fieldName := strings.TrimPrefix(key, "block.input.")

			// Open the file
			file, err := fileHeader.Open()
			if err != nil {
				return fmt.Errorf("failed to open file %s: %v", key, err)
			}
			defer file.Close()

			// Read the file content into a byte buffer
			fileBytes, err := io.ReadAll(file)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %v", key, err)
			}

			// Store the file content as a byte slice in the Input map
			b.Input[fieldName] = fileBytes
		}
	}

	return nil
}

// UnmarshalJSON for BlockInputSchema to handle empty string for TargetIndex
func (b *BlockInputSchema) UnmarshalJSON(data []byte) error {
	type Alias BlockInputSchema
	aux := &struct {
		TargetIndex *string `json:"target_index"` // Pointer to string for handling empty value
		*Alias
	}{
		Alias: (*Alias)(b),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Check if TargetIndex is nil (not present) or an empty string
	if aux.TargetIndex == nil || *aux.TargetIndex == "" {
		b.TargetIndex = -1 // Set to -1 if nil or empty
	} else {
		// Convert the string to an int
		var value int
		if _, err := fmt.Sscanf(*aux.TargetIndex, "%d", &value); err != nil {
			return fmt.Errorf("invalid target_index: %v", err)
		}
		b.TargetIndex = value
	}

	return nil
}

// PipelineStartInputSchema represents the structure of the entire JSON payload.
// It contains the `Pipeline` and `Block` data, which are used to start the pipeline process.
//
// swagger:model
type PipelineStartInputSchema struct {
	// The pipeline information, represented by the PipelineInputSchema model
	// required: true
	Pipeline PipelineInputSchema `json:"pipeline"`

	// The block information, represented by the BlockInputSchema model
	// required: true
	Block BlockInputSchema `json:"block"`
}

func (p *PipelineStartInputSchema) ParseForm(r *http.Request) error {
	// Parse the multipart form if not already done
	if r.MultipartForm == nil {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			return fmt.Errorf("unable to parse multipart form: %v", err)
		}
	}

	// Create a map for storing files
	files := make(map[string]*multipart.FileHeader)

	// Iterate over form files to include them in the `files` map
	for key, headers := range r.MultipartForm.File {
		if len(headers) > 0 {
			files[key] = headers[0] // Only consider the first file per key
		}
	}

	// Parse the pipeline and block using their respective methods
	if err := p.Pipeline.ParseForm(r.Form); err != nil {
		return fmt.Errorf("error parsing pipeline: %v", err)
	}
	if err := p.Block.ParseForm(r.Form, files); err != nil {
		return fmt.Errorf("error parsing block: %v", err)
	}
	return nil
}

// GetProcessingID returns the processing ID for the pipeline.
// If the Pipeline has a valid ProcessingID, it returns that, otherwise, it generates a new one.
//
// This method ensures that each pipeline process is associated with a unique ProcessingID.
// If the ProcessingID is missing, a new one is created and assigned to the Pipeline.
func (p *PipelineStartInputSchema) GetProcessingID() uuid.UUID {
	processingId := uuid.New()

	if p.Pipeline.ProcessingID != uuid.Nil {
		processingId = p.Pipeline.ProcessingID
	} else {
		p.Pipeline.ProcessingID = processingId
	}

	return processingId
}

// PipelineStartOutputSchema represents the structure of the output JSON
// when a new pipeline is started. It contains the unique processing ID
// that is generated or provided for the pipeline.
//
// swagger:model
type PipelineStartOutputSchema struct {
	// The unique processing ID for the pipeline
	// required: true
	// example: "d9b2d63d-5f23-e4d7-6b7f-3f2f25d93a7a"
	ProcessingID uuid.UUID `json:"processing_id"`
}

// PipelineResumeOutputSchema represents the structure of the output JSON
// when resuming a pipeline. It includes the processing ID associated
// with the pipeline that was previously started.
//
// swagger:model
type PipelineResumeOutputSchema struct {
	// The unique processing ID for the resumed pipeline
	// required: true
	// example: "d9b2d63d-5f23-e4d7-6b7f-3f2f25d93a7a"
	ProcessingID uuid.UUID `json:"processing_id"`
}
