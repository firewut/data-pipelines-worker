package blocks

var BlockRegistry = map[Block][]interface{}{
	NewBlockHTTP(): {
		nil, "https://google.com",
	},
}

func DetectBlocks() []Block {
	detectedBlocks := make([]Block, 0)

	for block, detectParams := range BlockRegistry {
		if block.Detect(detectParams) {
			detectedBlocks = append(detectedBlocks, block)
		}
	}

	return detectedBlocks
}
