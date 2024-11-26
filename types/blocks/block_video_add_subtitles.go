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

type DetectorVideoAddSubtitles struct {
	BlockDetectorParent
}

func NewDetectorVideoAddSubtitles(
	detectorConfig config.BlockConfigDetector,
) *DetectorVideoAddSubtitles {
	return &DetectorVideoAddSubtitles{
		BlockDetectorParent: NewDetectorParent(detectorConfig),
	}
}

func (d *DetectorVideoAddSubtitles) Detect() bool {
	d.Lock()
	defer d.Unlock()

	return true
}

type ProcessorVideoAddSubtitles struct {
}

func NewProcessorVideoAddSubtitles() *ProcessorVideoAddSubtitles {
	return &ProcessorVideoAddSubtitles{}
}

func (p *ProcessorVideoAddSubtitles) GetRetryCount(_ interfaces.Block) int {
	return 0
}

func (p *ProcessorVideoAddSubtitles) GetRetryInterval(_ interfaces.Block) time.Duration {
	return 0
}
func (p *ProcessorVideoAddSubtitles) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) ([]*bytes.Buffer, bool, bool, string, int, error) {
	output := make([]*bytes.Buffer, 0)

	var err error
	blockConfig := &BlockVideoAddSubtitlesConfig{}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockVideoAddSubtitles).GetBlockConfig(_config)
	userBlockConfig := &BlockVideoAddSubtitlesConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	video, err := helpers.GetValue[[]byte](_data, "video")
	if err != nil {
		return nil, false, false, "", -1, err
	}

	videoMimeType, err := helpers.DetectMimeTypeFromBuffer(*bytes.NewBuffer(video))
	if err != nil {
		return nil, false, false, "", -1, err
	}
	if videoMimeType.String() != "video/mp4" {
		return nil, false, false, "", -1, fmt.Errorf("Invalid video format. Only MP4 is supported")
	}

	subtitles, err := helpers.GetValue[[]byte](_data, "subtitles")
	if err != nil {
		return nil, false, false, "", -1, err
	}
	// subtitlesMimeType, err := helpers.DetectMimeTypeFromBuffer(*bytes.NewBuffer(subtitles))
	// if err != nil {
	// 	return nil, false, false, "", -1, err
	// }

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
		{namePattern: fmt.Sprintf("video-*%s", videoMimeType.Extension()), data: video},
		{namePattern: "subtitles-*.ass", data: subtitles},
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
	tempOutputFile, err := os.CreateTemp("", fmt.Sprintf("output-*%s", videoMimeType.Extension()))
	if err != nil {
		return nil, false, false, "", -1, err
	}
	defer os.Remove(tempOutputFile.Name())

	// Build FFmpeg arguments to concatenate the videos
	args := []string{
		"-y",                // Overwrite output file without asking
		"-i", files[0].name, // Input video File
	}

	switch blockConfig.EmbeddingType {
	case "burn":
		args = append(
			args,
			"-vf", fmt.Sprintf("ass=%s", files[1].name),
			"-c:v", "libx264",
			"-crf", "23",
			"-preset", "medium",
		)
	case "mux":
		args = append(
			args,
			"-i", files[1].name,
			"-c:v", "copy",
			"-c:s", "mov_text",
		)
	}

	args = append(
		args,
		"-c:a", "copy", // Copy the video stream from the first input file
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

	output = append(output, bytes.NewBuffer(videoBuffer))

	return output, false, false, "", -1, nil
}

type BlockVideoAddSubtitlesConfig struct {
	FFMPEGBinary  string `yaml:"ffmpeg_binary" json:"ffmpeg_binary"`
	EmbeddingType string `yaml:"embedding_type" json:"embedding_type"`
}

type BlockVideoAddSubtitles struct {
	generics.ConfigurableBlock[BlockVideoAddSubtitlesConfig] `json:"-" yaml:"-"`
	BlockParent
}

var _ interfaces.Block = (*BlockVideoAddSubtitles)(nil)

func (b *BlockVideoAddSubtitles) GetBlockConfig(_config config.Config) *BlockVideoAddSubtitlesConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockVideoAddSubtitlesConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockVideoAddSubtitles() *BlockVideoAddSubtitles {
	block := &BlockVideoAddSubtitles{
		BlockParent: BlockParent{
			Id:          "video_add_subtitles",
			Name:        "Video add Subtitles",
			Description: "Add Subtitles to the Video. Mux or burn subtitles.",
			Version:     "1",
			SchemaString: `{
				"type": "object",
				"properties": {
					"input":{
						"type": "object",
						"description": "Input data",
						"properties": {
							"video": {
								"description": "Video to have subtitles embedded. MP4 for now",
								"type": "string",
								"format": "file"
							},
							"subtitles": {
								"description": "Subtitles file to be embedded. ASS format for now",
								"type": "string",
								"format": "file"
							},
							"embedding_type": {
								"description": "Mux or Burn subtitles",
								"type": "string",
								"enum": ["mux", "burn"],
								"default": "mux"
							}
						},
						"required": ["video", "subtitles"]
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

	block.SetProcessor(NewProcessorVideoAddSubtitles())

	return block
}
