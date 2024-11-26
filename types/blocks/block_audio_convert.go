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

type DetectorAudioConvert struct {
	BlockDetectorParent
}

func NewDetectorAudioConvert(
	detectorConfig config.BlockConfigDetector,
) *DetectorAudioConvert {
	return &DetectorAudioConvert{
		BlockDetectorParent: NewDetectorParent(detectorConfig),
	}
}

func (d *DetectorAudioConvert) Detect() bool {
	d.Lock()
	defer d.Unlock()

	return true
}

type ProcessorAudioConvert struct {
}

func NewProcessorAudioConvert() *ProcessorAudioConvert {
	return &ProcessorAudioConvert{}
}

func (p *ProcessorAudioConvert) GetRetryCount(_ interfaces.Block) int {
	return 0
}

func (p *ProcessorAudioConvert) GetRetryInterval(_ interfaces.Block) time.Duration {
	return 0
}
func (p *ProcessorAudioConvert) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) ([]*bytes.Buffer, bool, bool, string, int, error) {
	var err error

	output := make([]*bytes.Buffer, 0)
	blockConfig := &BlockAudioConvertConfig{}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockAudioConvert).GetBlockConfig(_config)
	userBlockConfig := &BlockAudioConvertConfig{}
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

	// Create a temporary file to store the audio from the buffer
	tempAudioFile, err := os.CreateTemp("", fmt.Sprintf("audio-*%s", audioMimeType.Extension()))
	if err != nil {
		return nil, false, false, "", -1, err
	}
	defer os.Remove(tempAudioFile.Name())
	tempAudioFile.Write(audio)

	// Create a temporary file to store converted audio
	tempConvertedAudioFile, err := os.CreateTemp("", fmt.Sprintf("audio-converted*.%s", blockConfig.Format))
	if err != nil {
		return nil, false, false, "", -1, err
	}
	defer os.Remove(tempConvertedAudioFile.Name())

	acValue := "2"
	if blockConfig.Mono {
		acValue = "1"
	}
	arValue := fmt.Sprintf("%d", blockConfig.SampleRate)
	baValue := blockConfig.BitRate

	args := []string{
		"-y",
		"-i", tempAudioFile.Name(),
		"-ac", acValue,
		"-ar", arValue,
		"-b:a", baValue,
		"-preset", "ultrafast",
		tempConvertedAudioFile.Name(),
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

	// Read the resulting video into a buffer
	videoBuffer, err := os.ReadFile(tempConvertedAudioFile.Name())
	if err != nil {
		return nil, false, false, "", -1, err
	}

	output = append(output, bytes.NewBuffer(videoBuffer))

	return output, false, false, "", -1, nil
}

type BlockAudioConvertConfig struct {
	FFMPEGBinary string `yaml:"ffmpeg_binary" json:"ffmpeg_binary"`
	Format       string `yaml:"format" json:"format"`
	Mono         bool   `yaml:"mono" json:"mono"`
	SampleRate   int    `yaml:"sample_rate" json:"sample_rate"`
	BitRate      string `yaml:"bit_rate" json:"bit_rate"`
}

type BlockAudioConvert struct {
	generics.ConfigurableBlock[BlockAudioConvertConfig] `json:"-" yaml:"-"`
	BlockParent
}

var _ interfaces.Block = (*BlockAudioConvert)(nil)

func (b *BlockAudioConvert) GetBlockConfig(_config config.Config) *BlockAudioConvertConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockAudioConvertConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockAudioConvert() *BlockAudioConvert {
	block := &BlockAudioConvert{
		BlockParent: BlockParent{
			Id:          "audio_convert",
			Name:        "Audio Convert",
			Description: "Convert Audio to to specified format audio",
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
							"format": {
								"description": "Output format",
								"type": "string",
								"default": "mp3",
								"enum": ["mp3"]
							},
							"mono": {
								"description": "Convert audio to mono",
								"type": "boolean",
								"default": false
							},
							"sample_rate": {
								"description": "Sample rate of the audio",
								"type": "integer",
								"default": 44100
							},
							"bit_rate": {
								"description": "Bit rate of the audio",
								"type": "string",
								"default": "64k"
							}
						},
						"required": ["audio"]
					},
					"output": {
						"description": "Converted audio",
						"type": ["string", "null"],
						"format": "file"
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

	block.SetProcessor(NewProcessorAudioConvert())

	return block
}
