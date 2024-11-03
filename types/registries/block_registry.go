package registries

import (
	"context"
	"net/http"
	"sync"

	"github.com/google/uuid"

	"data-pipelines-worker/types/blocks"
	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/interfaces"
)

var (
	onceBlockRegistry     sync.Once
	blockRegistryInstance *BlockRegistry
)

func GetBlockRegistry(forceNewInstance ...bool) *BlockRegistry {
	if len(forceNewInstance) > 0 && forceNewInstance[0] {
		newInstance := NewBlockRegistry()
		blockRegistryInstance = newInstance
		onceBlockRegistry = sync.Once{}
		return newInstance
	}

	onceBlockRegistry.Do(func() {
		blockRegistryInstance = NewBlockRegistry()
	})

	return blockRegistryInstance
}

// BlockRegistry is a registry for detected Blocks
type BlockRegistry struct {
	sync.Mutex

	Id             uuid.UUID
	Blocks         map[string]interfaces.Block
	blocksDetector map[interfaces.Block]interfaces.BlockDetector

	shutdownWg *sync.WaitGroup
}

// Ensure BlockRegistry implements the BlockRegistry
var _ interfaces.BlockRegistry = (*BlockRegistry)(nil)

func NewBlockRegistry() *BlockRegistry {
	registry := &BlockRegistry{
		Id:             uuid.New(),
		Blocks:         make(map[string]interfaces.Block),
		blocksDetector: make(map[interfaces.Block]interfaces.BlockDetector),
		shutdownWg:     &sync.WaitGroup{},
	}

	registry.DetectBlocks()

	return registry
}

func (br *BlockRegistry) DetectBlocks() {
	br.Lock()

	_config := config.GetConfig()

	httpBlock := blocks.NewBlockHTTP()
	openAIRequestCompletionBlock := blocks.NewBlockOpenAIRequestCompletion()
	openAIRequestTTSBlock := blocks.NewBlockOpenAIRequestTTS()
	openAIRequestTranscriptionBlock := blocks.NewBlockOpenAIRequestTranscription()
	openAIRequestImageBlock := blocks.NewBlockOpenAIRequestImage()
	imageAddTextBlock := blocks.NewBlockImageAddText()
	imageResizeBlock := blocks.NewBlockImageResize()
	imageBlurBlock := blocks.NewBlockImageBlur()
	stopPipelineBlock := blocks.NewBlockStopPipeline()
	sendModerationToTelegramBlock := blocks.NewBlockSendModerationToTelegram()
	fetchModerationFromTelegramBlock := blocks.NewBlockFetchModerationFromTelegram()
	textReplaceBlock := blocks.NewBlockTextReplace()
	textAddPrefixOrSuffixBlock := blocks.NewBlockTextAddPrefixOrSuffix()
	videoFromImageBlock := blocks.NewBlockVideoFromImage()
	joinVideosBlock := blocks.NewBlockJoinVideos()
	videoAddAudioBlock := blocks.NewBlockVideoAddAudio()
	sendMessageToTelegramBlock := blocks.NewBlockSendMessageToTelegram()
	formatStringFromObjectBlock := blocks.NewBlockFormatStringFromObject()

	br.blocksDetector = map[interfaces.Block]interfaces.BlockDetector{
		httpBlock: blocks.NewDetectorHTTP(
			&http.Client{},
			_config.Blocks[httpBlock.GetId()].Detector,
		),
		openAIRequestCompletionBlock: blocks.NewDetectorOpenAI(
			_config.OpenAI.GetClient(),
			_config.Blocks[openAIRequestCompletionBlock.GetId()].Detector,
		),
		openAIRequestTTSBlock: blocks.NewDetectorOpenAI(
			_config.OpenAI.GetClient(),
			_config.Blocks[openAIRequestTTSBlock.GetId()].Detector,
		),
		openAIRequestTranscriptionBlock: blocks.NewDetectorOpenAI(
			_config.OpenAI.GetClient(),
			_config.Blocks[openAIRequestTranscriptionBlock.GetId()].Detector,
		),
		openAIRequestImageBlock: blocks.NewDetectorOpenAI(
			_config.OpenAI.GetClient(),
			_config.Blocks[openAIRequestImageBlock.GetId()].Detector,
		),
		imageAddTextBlock: blocks.NewDetectorImageAddText(
			_config.Blocks[imageAddTextBlock.GetId()].Detector,
		),
		imageResizeBlock: blocks.NewDetectorImageResize(
			_config.Blocks[imageResizeBlock.GetId()].Detector,
		),
		imageBlurBlock: blocks.NewDetectorImageBlur(
			_config.Blocks[imageBlurBlock.GetId()].Detector,
		),
		stopPipelineBlock: blocks.NewDetectorStopPipeline(
			_config.Blocks[stopPipelineBlock.GetId()].Detector,
		),
		sendModerationToTelegramBlock: blocks.NewDetectorTelegramBot(
			_config.Telegram.GetClient(),
			_config.Blocks[sendModerationToTelegramBlock.GetId()].Detector,
		),
		fetchModerationFromTelegramBlock: blocks.NewDetectorTelegramBot(
			_config.Telegram.GetClient(),
			_config.Blocks[fetchModerationFromTelegramBlock.GetId()].Detector,
		),
		textReplaceBlock: blocks.NewDetectorTextReplace(
			_config.Blocks[textReplaceBlock.GetId()].Detector,
		),
		textAddPrefixOrSuffixBlock: blocks.NewDetectorTextAddPrefixOrSuffix(
			_config.Blocks[textAddPrefixOrSuffixBlock.GetId()].Detector,
		),
		videoFromImageBlock: blocks.NewDetectorVideoFromImage(
			_config.Blocks[videoFromImageBlock.GetId()].Detector,
		),
		joinVideosBlock: blocks.NewDetectorJoinVideos(
			_config.Blocks[joinVideosBlock.GetId()].Detector,
		),
		videoAddAudioBlock: blocks.NewDetectorVideoAddAudio(
			_config.Blocks[videoAddAudioBlock.GetId()].Detector,
		),
		sendMessageToTelegramBlock: blocks.NewDetectorTelegramBot(
			_config.Telegram.GetClient(),
			_config.Blocks[sendMessageToTelegramBlock.GetId()].Detector,
		),
		formatStringFromObjectBlock: blocks.NewDetectorFormatStringFromObject(
			_config.Blocks[formatStringFromObjectBlock.GetId()].Detector,
		),
	}

	br.Blocks = make(map[string]interfaces.Block)
	br.Unlock()

	startUpWg := &sync.WaitGroup{}

	for block, detector := range br.blocksDetector {
		startUpWg.Add(1)

		go func() {
			br.Lock()
			defer br.Unlock()
			defer startUpWg.Done()

			block.SetAvailable(false)
			if detector.Detect() {
				block.SetAvailable(true)
			}

			br.shutdownWg.Add(1)
			detector.Start(block, detector.Detect)

			br.Blocks[block.GetId()] = block
		}()
	}

	startUpWg.Wait()
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

	// Stop the Detector
	if detector, ok := br.blocksDetector[br.Blocks[id]]; ok {
		detector.Stop(br.shutdownWg)
	}

	delete(br.Blocks, id)
}

func (br *BlockRegistry) DeleteAll() {
	br.Lock()
	defer br.Unlock()

	for id := range br.Blocks {
		delete(br.Blocks, id)
	}
}

func (br *BlockRegistry) Shutdown(ctx context.Context) error {
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

func (br *BlockRegistry) IsAvailable(block interfaces.Block) bool {
	availableBlocks := br.GetAvailableBlocks()
	_, ok := availableBlocks[block.GetId()]
	block.SetAvailable(ok)

	return ok
}
