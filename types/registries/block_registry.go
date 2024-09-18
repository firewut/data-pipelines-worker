package registries

import (
	"context"
	"net/http"
	"sync"

	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/interfaces"
)

var (
	onceBlockRegistry     sync.Once
	blockRegistryInstance *BlockRegistry
)

func GetBlockRegistry() *BlockRegistry {
	onceBlockRegistry.Do(func() {
		blockRegistryInstance = NewBlockRegistry()
	})

	return blockRegistryInstance
}

// BlockRegistry is a registry for detected Blocks
type BlockRegistry struct {
	sync.Mutex

	Blocks         map[string]interfaces.Block
	blocksDetector map[interfaces.Block]interfaces.BlockDetector

	shutdownWg *sync.WaitGroup
}

func NewBlockRegistry() *BlockRegistry {
	registry := &BlockRegistry{
		Blocks:         make(map[string]interfaces.Block),
		blocksDetector: make(map[interfaces.Block]interfaces.BlockDetector),
		shutdownWg:     &sync.WaitGroup{},
	}

	registry.DetectBlocks()

	return registry
}

func (br *BlockRegistry) DetectBlocks() {
	br.Lock()
	defer br.Unlock()

	_config := config.GetConfig()

	httpBlock := blocks.NewBlockHTTP()
	openAIBlock := blocks.NewBlockOpenAIRequestCompletion()

	br.blocksDetector = map[interfaces.Block]interfaces.BlockDetector{
		httpBlock: blocks.NewDetectorHTTP(
			&http.Client{},
			_config.Blocks[httpBlock.GetId()].Detector,
		),
		openAIBlock: blocks.NewDetectorHTTP(
			&http.Client{},
			_config.Blocks[openAIBlock.GetId()].Detector,
		),
	}

	br.Blocks = make(map[string]interfaces.Block)
	for block, detector := range br.blocksDetector {
		block.SetAvailable(false)

		if detector.Detect() {
			block.SetAvailable(true)
		}

		br.shutdownWg.Add(1)
		detector.Start(block, detector.Detect)

		br.Blocks[block.GetId()] = block
	}
}

func (br *BlockRegistry) Add(block interfaces.Block) {
	br.Lock()
	defer br.Unlock()

	br.Blocks[block.GetId()] = block
}

func (br *BlockRegistry) Get(id string) interfaces.Block {
	br.Lock()
	defer br.Unlock()

	return br.Blocks[id]
}

func (br *BlockRegistry) GetAll() map[string]interfaces.Block {
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

func (br *BlockRegistry) Delete(id string) {
	br.Lock()
	defer br.Unlock()

	delete(br.Blocks, id)
}

func (br *BlockRegistry) DeleteAll() {
	br.Lock()
	defer br.Unlock()

	for id := range br.Blocks {
		delete(br.Blocks, id)
	}
}

func (br *BlockRegistry) Shutdown(context context.Context) error {
	br.Lock()
	defer br.Unlock()

	for _, detector := range br.blocksDetector {
		detector.Stop(br.shutdownWg)
	}

	// Any Processing Pipelines will transfer requests
	// to the other Workers
	for _, block := range br.Blocks {
		block.SetAvailable(false)
	}

	br.shutdownWg.Wait()

	return nil
}
