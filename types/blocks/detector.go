package blocks

import (
	"net/http"

	"data-pipelines-worker/types"
)

var BlockRegistry = map[types.Block]types.BlockDetector{
	NewBlockHTTP(): &DetectorHTTP{
		Client: &http.Client{},
		Url:    "https://google.com",
	},
}

func DetectBlocks() []types.Block {
	detectedBlocks := make([]types.Block, 0)

	for block, detector := range BlockRegistry {
		if block.Detect(detector) {
			validator := types.BlockSchemaValidator{}
			if _, _, err := validator.Validate(block); err == nil {
				block.SetSchema(validator)
			} else {
				types.GetLogger().Warnf(
					"Block (%s) was detected but schema is invalid: %s",
					block.GetId(),
					err,
				)
				continue
			}
			// Mark block as Detected
			detectedBlocks = append(detectedBlocks, block)
		}
	}

	return detectedBlocks
}
