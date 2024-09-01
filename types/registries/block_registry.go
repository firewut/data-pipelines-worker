package registries

import (
	"net/http"
	"sync"

	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/interfaces"
)

var (
	onceBlockRegistry     sync.Once
	blockRegistryInstance *BlockRegistry
)

type BlockRegistry struct {
	sync.Mutex

	Blocks map[string]interfaces.Block
}

func NewBlockRegistry() *BlockRegistry {
	registry := &BlockRegistry{
		Blocks: make(map[string]interfaces.Block),
	}

	registry.DetectBlocks()

	return registry
}

func (br *BlockRegistry) DetectBlocks() {
	br.Lock()
	defer br.Unlock()

	blockDetector := map[interfaces.Block]interfaces.BlockDetector{
		blocks.NewBlockHTTP(): &blocks.DetectorHTTP{
			Client: &http.Client{},
			Url:    "https://google.com",
		},
		blocks.NewBlockOpenAIRequestCompletion(): &blocks.DetectorHTTP{
			Client: &http.Client{},
			Url:    "https://api.openai.com",
		},
	}

	br.Blocks = make(map[string]interfaces.Block)
	for block, detector := range blockDetector {
		block.SetAvailable(false)

		if block.Detect(detector) {
			block.SetAvailable(true)
		}

		br.Blocks[block.GetId()] = block
	}
}

func (br *BlockRegistry) GetBlocks() map[string]interfaces.Block {
	br.Lock()
	defer br.Unlock()

	return br.Blocks
}

func (br *BlockRegistry) GetAvailableBlocks() map[string]interfaces.Block {
	br.Lock()
	defer br.Unlock()

	availableBlocks := make(map[string]interfaces.Block)
	for id, block := range br.Blocks {
		if block.IsAvailable() {
			availableBlocks[id] = block
		}
	}

	return availableBlocks
}

func GetBlockRegistry() *BlockRegistry {
	onceBlockRegistry.Do(func() {
		blockRegistryInstance = NewBlockRegistry()
	})

	return blockRegistryInstance
}
