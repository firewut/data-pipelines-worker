package blocks

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/generics"
	"data-pipelines-worker/types/helpers"
	"data-pipelines-worker/types/interfaces"
)

type DetectorAudioChunk struct {
	BlockDetectorParent
}

func NewDetectorAudioChunk(
	detectorConfig config.BlockConfigDetector,
) *DetectorAudioChunk {
	return &DetectorAudioChunk{
		BlockDetectorParent: NewDetectorParent(detectorConfig),
	}
}

func (d *DetectorAudioChunk) Detect() bool {
	d.Lock()
	defer d.Unlock()

	return true
}

type ProcessorAudioChunk struct {
}

func NewProcessorAudioChunk() *ProcessorAudioChunk {
	return &ProcessorAudioChunk{}
}

func (p *ProcessorAudioChunk) GetRetryCount(_ interfaces.Block) int {
	return 0
}

func (p *ProcessorAudioChunk) GetRetryInterval(_ interfaces.Block) time.Duration {
	return 0
}
func (p *ProcessorAudioChunk) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) ([]*bytes.Buffer, bool, bool, string, int, error) {
	var err error

	output := make([]*bytes.Buffer, 0)
	blockConfig := &BlockAudioChunkConfig{}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockAudioChunk).GetBlockConfig(_config)
	userBlockConfig := &BlockAudioChunkConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	audio, err := helpers.GetValue[[]byte](_data, "audio")
	if err != nil {
		return nil, false, false, "", -1, err
	}

	audioMimeType, err := helpers.DetectMimeTypeFromBuffer(*bytes.NewBuffer(audio))
	if err != nil {
		return nil, false, false, "", -1, err
	}

	if audioMimeType.String() != "audio/mpeg" {
		return nil, false, false, "", -1, fmt.Errorf("invalid audio format. Only MP3 is supported")
	}

	ffmpegBinary := blockConfig.FFMPEGBinary
	if ffmpegBinary == "" {
		ffmpegBinary, err = config.GetFFmpegBinary()
		if err != nil {
			return nil, false, false, "", -1, err
		}
	}
	if _, err := os.Stat(ffmpegBinary); os.IsNotExist(err) {
		return nil, false, false, "", -1, fmt.Errorf("FFmpeg binary not found: %s", ffmpegBinary)
	}

	// Determine the chunk duration (default to 10 minutes)
	duration, err := time.ParseDuration(blockConfig.Duration)
	if err != nil {
		return nil, false, false, "", -1, err
	}
	if duration.Seconds() <= 0 {
		duration = time.Duration(time.Minute * 10) // Default to 10 minutes if no duration is specified
	}

	// Create a temporary file to store the audio from the buffer
	tempAudioFile, err := os.CreateTemp("", fmt.Sprintf("audio-*%s", audioMimeType.Extension()))
	if err != nil {
		return nil, false, false, "", -1, err
	}
	defer os.Remove(tempAudioFile.Name())
	tempAudioFile.Write(audio)

	// Create a temporary directory to store chunks
	tempDir, err := os.MkdirTemp("", "audio_chunks")
	if err != nil {
		return nil, false, false, "", -1, err
	}
	defer os.RemoveAll(tempDir)

	// Use FFmpeg to split the audio file into smaller chunks
	args := []string{
		"-y",
		"-i", tempAudioFile.Name(),
		"-f", "segment",
		"-segment_time", fmt.Sprintf("%f", duration.Seconds()),
		"-c", "copy",
		fmt.Sprintf("%s/segment%%03d.mp3", tempDir), // Output pattern for chunks
	}

	var stderr bytes.Buffer
	cmd := exec.Command(ffmpegBinary, args...)
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		fmt.Printf("FFmpeg error: %v\n", err)
		fmt.Printf("FFmpeg stderr: %s\n", stderr.String()) // Capture full error message

		return nil, false, false, "", -1, fmt.Errorf("ffmpeg error: %v\nstderr: %s", err, stderr.String())
	}

	// Collect the file paths of the created chunks
	chunks := []string{}
	files, err := os.ReadDir(tempDir)
	if err != nil {
		return nil, false, false, "", -1, err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		chunks = append(chunks, fmt.Sprintf("%s/%s", tempDir, file.Name()))
	}

	// For this example, we return a simple output with chunk paths
	for _, chunk := range chunks {
		// Read each chunk and write it to the output buffer (or use another approach)
		fmt.Println("> Chunk:", chunk)
		chunkData, err := os.ReadFile(chunk)
		if err != nil {
			return nil, false, false, "", -1, err
		}
		output = append(output, bytes.NewBuffer(chunkData))
	}

	return output, false, false, "", -1, nil
}

type BlockAudioChunkConfig struct {
	FFMPEGBinary string `yaml:"ffmpeg_binary" json:"ffmpeg_binary"`
	Duration     string `yaml:"duration" json:"duration"`
}

type BlockAudioChunk struct {
	generics.ConfigurableBlock[BlockAudioChunkConfig] `json:"-" yaml:"-"`
	BlockParent
}

var _ interfaces.Block = (*BlockAudioChunk)(nil)

func (b *BlockAudioChunk) GetBlockConfig(_config config.Config) *BlockAudioChunkConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockAudioChunkConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockAudioChunk() *BlockAudioChunk {
	block := &BlockAudioChunk{
		BlockParent: BlockParent{
			Id:          "audio_chunk",
			Name:        "Audio Chunk",
			Description: "Split Audio to chunks of specified duration",
			Version:     "1",
			SchemaString: `{
				"type": "object",
				"properties": {
					"input":{
						"type": "object",
						"description": "Input data",
						"properties": {
							"audio": {
								"description": "Audio file to convert. Must be MP3",
								"type": "string",
								"format": "file"
							},
							"duration": {
								"description": "Duration of the audio",
								"type": "string",
								"default": "10m"
							}
						},
						"required": ["audio"]
					},
					"output": {
						"description": "Chunked audio",
						"type": "array",
						"items": {
							"type": ["string", "null"],
							"format": "file"
						}
					}
				}
			}`,
			SchemaPtr: nil,
			Schema:    nil,
		},
	}

	if err := block.ApplySchema(block.GetSchemaString()); err != nil {
		panic(err)
	}

	block.SetProcessor(NewProcessorAudioChunk())

	return block
}
