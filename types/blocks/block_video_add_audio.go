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

type DetectorVideoAddAudio struct {
	BlockDetectorParent
}

func NewDetectorVideoAddAudio(
	detectorConfig config.BlockConfigDetector,
) *DetectorVideoAddAudio {
	return &DetectorVideoAddAudio{
		BlockDetectorParent: NewDetectorParent(detectorConfig),
	}
}

func (d *DetectorVideoAddAudio) Detect() bool {
	d.Lock()
	defer d.Unlock()

	return true
}

type ProcessorVideoAddAudio struct {
}

func NewProcessorVideoAddAudio() *ProcessorVideoAddAudio {
	return &ProcessorVideoAddAudio{}
}

func (p *ProcessorVideoAddAudio) GetRetryCount(_ interfaces.Block) int {
	return 0
}

func (p *ProcessorVideoAddAudio) GetRetryInterval(_ interfaces.Block) time.Duration {
	return 0
}
func (p *ProcessorVideoAddAudio) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, bool, bool, string, int, error) {
	var err error

	output := &bytes.Buffer{}
	blockConfig := &BlockVideoAddAudioConfig{}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockVideoAddAudio).GetBlockConfig(_config)
	userBlockConfig := &BlockVideoAddAudioConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	video, err := helpers.GetValue[[]byte](_data, "video")
	if err != nil {
		return nil, false, false, "", -1, err
	}

	audio, err := helpers.GetValue[[]byte](_data, "audio")
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

	type blockTmpFile struct {
		name        string
		data        []byte
		namePattern string
	}

	files := []*blockTmpFile{
		{namePattern: "video-*.mp4", data: video},
		{namePattern: "audio-*.mp3", data: audio},
	}
	for _, tmpFile := range files {
		tempFile, err := os.CreateTemp("", tmpFile.namePattern)
		if err != nil {
			return nil, false, false, "", -1, err
		}
		defer os.Remove(tempFile.Name())
		tmpFile.name = tempFile.Name()

		// Write the video buffer to the temp file
		if _, err = tempFile.Write(tmpFile.data); err != nil {
			return nil, false, false, "", -1, err
		}
		tempFile.Close()
	}

	// Create a temporary file to store the output video
	tempOutputFile, err := os.CreateTemp("", "output-*.mp4")
	if err != nil {
		return nil, false, false, "", -1, err
	}
	defer os.Remove(tempOutputFile.Name())

	// Build FFmpeg arguments to concatenate the videos
	args := []string{
		"-y",                // Overwrite output file without asking
		"-i", files[0].name, // Input video File
		"-i", files[1].name, // Input audio File
	}

	if blockConfig.ReplaceOriginalAudio {
		args = append(args, "-map", "0:v") // Map the video stream from the first input file
	} else {
		args = append(args, "-map", "0") // Map all streams from the first input file
	}

	args = append(
		args,
		"-map", "1:a", // Map the audio stream from the second input file
		"-c:v", "copy", // Copy the video stream from the first input file
	)

	args = append(args, tempOutputFile.Name())

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
	videoBuffer, err := os.ReadFile(tempOutputFile.Name())
	if err != nil {
		return nil, false, false, "", -1, err
	}

	output.Write(videoBuffer)

	return output, false, false, "", -1, nil
}

type BlockVideoAddAudioConfig struct {
	FFMPEGBinary         string `yaml:"ffmpeg_binary" json:"ffmpeg_binary"`
	ReplaceOriginalAudio bool   `yaml:"replace_original_audio" json:"replace_original_audio"`
}

type BlockVideoAddAudio struct {
	generics.ConfigurableBlock[BlockVideoAddAudioConfig] `json:"-" yaml:"-"`
	BlockParent
}

var _ interfaces.Block = (*BlockVideoAddAudio)(nil)

func (b *BlockVideoAddAudio) GetBlockConfig(_config config.Config) *BlockVideoAddAudioConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockVideoAddAudioConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockVideoAddAudio() *BlockVideoAddAudio {
	block := &BlockVideoAddAudio{
		BlockParent: BlockParent{
			Id:          "video_add_audio",
			Name:        "Video add Audio",
			Description: "Add Audio to the Video.",
			Version:     "1",
			SchemaString: `{
				"type": "object",
				"properties": {
					"input":{
						"type": "object",
						"description": "Input data",
						"properties": {
							"video": {
								"description": "List of videos to join",
								"type": "string",
								"format": "file"
							},
							"audio": {
								"description": "Audio file to add to the video",
								"type": "string",
								"format": "file"
							},
							"replace_original_audio": {
								"description": "Replace original audio with the new audio",
								"type": "boolean",
								"default": false
							}
						},
						"required": ["video", "audio"]
					},
					"output": {
						"description": "Video with added Audio",
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

	block.SetProcessor(NewProcessorVideoAddAudio())

	return block
}
