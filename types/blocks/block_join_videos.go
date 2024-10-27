package blocks

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"time"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/generics"
	"data-pipelines-worker/types/helpers"
	"data-pipelines-worker/types/interfaces"
)

type DetectorJoinVideos struct {
	BlockDetectorParent
}

func NewDetectorJoinVideos(
	detectorConfig config.BlockConfigDetector,
) *DetectorJoinVideos {
	return &DetectorJoinVideos{
		BlockDetectorParent: NewDetectorParent(detectorConfig),
	}
}

func (d *DetectorJoinVideos) Detect() bool {
	d.Lock()
	defer d.Unlock()

	return true
}

type ProcessorJoinVideos struct {
}

func NewProcessorJoinVideos() *ProcessorJoinVideos {
	return &ProcessorJoinVideos{}
}

func (p *ProcessorJoinVideos) GetRetryCount(_ interfaces.Block) int {
	return 0
}

func (p *ProcessorJoinVideos) GetRetryInterval(_ interfaces.Block) time.Duration {
	return 0
}
func (p *ProcessorJoinVideos) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, bool, bool, string, int, error) {
	var err error

	output := &bytes.Buffer{}
	blockConfig := &BlockJoinVideosConfig{}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockJoinVideos).GetBlockConfig(_config)
	userBlockConfig := &BlockJoinVideosConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	videos := make([][]byte, 0)
	val := reflect.ValueOf(_data["videos"])
	for i := 0; i < val.Len(); i++ {
		item := val.Index(i).Interface()
		videos = append(videos, item.([]byte))
	}

	// videos, err := helpers.GetValue[[][]byte](_data, "videos")
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

	// Create a temporary file to store the concat list
	tempListFile, err := os.CreateTemp("", "concat-list-*.txt")
	if err != nil {
		return nil, false, false, "", -1, err
	}
	defer os.Remove(tempListFile.Name())

	// Create temporary files for each video and write their contents
	for _, videoBuffer := range videos {
		tempVideoFile, err := os.CreateTemp("", "video-*.mp4")
		if err != nil {
			return nil, false, false, "", -1, err
		}

		// Write the video buffer to the temp file
		if _, err = tempVideoFile.Write(videoBuffer); err != nil {
			return nil, false, false, "", -1, err
		}
		tempVideoFile.Close()

		// Write to the concat list
		fmt.Fprintf(tempListFile, "file '%s'\n", tempVideoFile.Name())
	}
	tempListFile.Close()

	// Create a temporary file to store the output video
	tempOutputFile, err := os.CreateTemp("", "output-*.mp4")
	if err != nil {
		return nil, false, false, "", -1, err
	}
	defer os.Remove(tempOutputFile.Name())

	// Build FFmpeg arguments to concatenate the videos
	args := []string{
		"-y",           // Overwrite output file without asking
		"-f", "concat", // Use the concat demuxer
		"-safe", "0", // Allow unsafe file paths
		"-i", tempListFile.Name(), // Input file list
	}

	if blockConfig.ReEncode {
		// Re-encode the video
		args = append(
			args,
			"-c:v", "libx264", // Set video codec to H.264
			"-crf", "23", // Set constant rate factor
			"-preset", "veryfast", // Set encoding speed
			"-pix_fmt", "yuv420p", // Set pixel format for compatibility
			"-f", blockConfig.Format, // Video Output format
			"-c:a", "aac", // Set audio codec to AAC
			"-b:a", "192k", // Set audio bit rate to 192 kbps
		)
	} else {
		args = append(args, "-c", "copy") // Copy the codec to avoid re-encoding
	}

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

type BlockJoinVideosConfig struct {
	FFMPEGBinary string `yaml:"ffmpeg_binary" json:"ffmpeg_binary"`
	Format       string `yaml:"format" json:"format"`
	ReEncode     bool   `yaml:"re_encode" json:"re_encode"`
}

type BlockJoinVideos struct {
	generics.ConfigurableBlock[BlockJoinVideosConfig]
	BlockParent
}

var _ interfaces.Block = (*BlockJoinVideos)(nil)

func (b *BlockJoinVideos) GetBlockConfig(_config config.Config) *BlockJoinVideosConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockJoinVideosConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockJoinVideos() *BlockJoinVideos {
	block := &BlockJoinVideos{
		BlockParent: BlockParent{
			Id:          "join_videos",
			Name:        "Join Videos",
			Description: "Join multiple videos into one",
			Version:     "1",
			SchemaString: `{
				"type": "object",
				"properties": {
					"input":{
						"type": "object",
						"description": "Input data",
						"properties": {
							"videos": {
								"description": "List of videos to join",
								"type": "array",
								"items": {
									"type": "string",
									"format": "file"
								},
								"minItems": 2
							},
							"re_encode": {
								"description": "If true, re-encode the videos",
								"type": "boolean",
								"default": false
							},
							"format": {
								"description": "Output format",
								"type": "string",
								"default": "mp4",
								"enum": ["mp4"]
							}
						},
						"required": ["videos"]
					},
					"output": {
						"description": "Video generated from list of videos",
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

	block.SetProcessor(NewProcessorJoinVideos())

	return block
}
