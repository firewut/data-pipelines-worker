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

type DetectorAudioFromVideo struct {
	BlockDetectorParent
}

func NewDetectorAudioFromVideo(
	detectorConfig config.BlockConfigDetector,
) *DetectorAudioFromVideo {
	return &DetectorAudioFromVideo{
		BlockDetectorParent: NewDetectorParent(detectorConfig),
	}
}

func (d *DetectorAudioFromVideo) Detect() bool {
	d.Lock()
	defer d.Unlock()

	return true
}

type ProcessorAudioFromVideo struct {
}

func NewProcessorAudioFromVideo() *ProcessorAudioFromVideo {
	return &ProcessorAudioFromVideo{}
}

func (p *ProcessorAudioFromVideo) GetRetryCount(_ interfaces.Block) int {
	return 0
}

func (p *ProcessorAudioFromVideo) GetRetryInterval(_ interfaces.Block) time.Duration {
	return 0
}

func (p *ProcessorAudioFromVideo) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, bool, bool, string, int, error) {
	output := &bytes.Buffer{}
	blockConfig := &BlockAudioFromVideoConfig{}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockAudioFromVideo).GetBlockConfig(_config)
	userBlockConfig := &BlockAudioFromVideoConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	videoBytes, err := helpers.GetValue[[]byte](_data, "video")
	if err != nil {
		return nil, false, false, "", -1, err
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

	// Write the video to the temp file
	tempVideoFile, err := os.CreateTemp("", "video-*.mp4")
	if err != nil {
		return nil, false, false, "", -1, err
	}
	defer os.Remove(tempVideoFile.Name())

	if _, err = tempVideoFile.Write(videoBytes); err != nil {
		return nil, false, false, "", -1, err
	}
	tempVideoFile.Close()

	// Create a temporary file to store the output audio
	tempOutputFile, err := os.CreateTemp("", "output-*.mp4")
	if err != nil {
		return nil, false, false, "", -1, err
	}
	defer os.Remove(tempOutputFile.Name())

	// Build FFmpeg arguments to concatenate the videos
	args := []string{
		"-y",                       // Overwrite output file without asking
		"-i", tempVideoFile.Name(), // Input video File
		"-q:a", "0", // Audio quality
		"-map", "a", // Map audio stream,
		"-f", blockConfig.Format, // Output format
	}

	if blockConfig.Start > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.3f", blockConfig.Start))
	}
	if blockConfig.End > 0 && blockConfig.End > blockConfig.Start {
		args = append(args, "-t", fmt.Sprintf("%.3f", blockConfig.End))
	}

	args = append(args, tempOutputFile.Name())

	var stderr bytes.Buffer
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return nil, false, false, "", -1, err
	}

	videoBuffer, err := os.ReadFile(tempOutputFile.Name())
	if err != nil {
		return nil, false, false, "", -1, err
	}

	output.Write(videoBuffer)

	return output, false, false, "", -1, nil
}

type BlockAudioFromVideoConfig struct {
	FFMPEGBinary string  `yaml:"ffmpeg_binary" json:"ffmpeg_binary"`
	Start        float64 `yaml:"start" json:"start"`
	End          float64 `yaml:"end" json:"end"`
	Format       string  `yaml:"format" json:"format"`
}

type BlockAudioFromVideo struct {
	generics.ConfigurableBlock[BlockAudioFromVideoConfig] `json:"-" yaml:"-"`
	BlockParent
}

var _ interfaces.Block = (*BlockAudioFromVideo)(nil)

func (b *BlockAudioFromVideo) GetBlockConfig(_config config.Config) *BlockAudioFromVideoConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockAudioFromVideoConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockAudioFromVideo() *BlockAudioFromVideo {
	block := &BlockAudioFromVideo{
		BlockParent: BlockParent{
			Id:          "audio_from_video",
			Name:        "Audio from Video",
			Description: "Extact audio from video",
			Version:     "1",
			SchemaString: `{
				"type": "object",
				"properties": {
					"input":{
						"type": "object",
						"description": "Input data",
						"properties": {
							"video": {
								"description": "Video to get audio track from. If start and end are set, resulting audio will be trimmed.",
								"type": "string",
								"format": "file"
							},
							"start": {
								"description": "Start time in seconds",
								"type": "number",
								"format": "float",
								"default": -1.0
							},
							"end": {
								"description": "End time in seconds",
								"type": "number",
								"format": "float",
								"default": -1.0
							},
							"format": {
								"description": "Output format",
								"type": "string",
								"default": "mp3",
								"enum": ["mp3"]
							}
						},
						"required": ["video"]
					},
					"output": {
						"description": "Audio generated from Video",
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

	block.SetProcessor(NewProcessorAudioFromVideo())

	return block
}
