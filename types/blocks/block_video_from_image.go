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

type DetectorVideoFromImage struct {
	BlockDetectorParent
}

func NewDetectorVideoFromImage(
	detectorConfig config.BlockConfigDetector,
) *DetectorVideoFromImage {
	return &DetectorVideoFromImage{
		BlockDetectorParent: NewDetectorParent(detectorConfig),
	}
}

func (d *DetectorVideoFromImage) Detect() bool {
	d.Lock()
	defer d.Unlock()

	return true
}

type ProcessorVideoFromImage struct {
}

func NewProcessorVideoFromImage() *ProcessorVideoFromImage {
	return &ProcessorVideoFromImage{}
}

func (p *ProcessorVideoFromImage) GetRetryCount(_ interfaces.Block) int {
	return 0
}

func (p *ProcessorVideoFromImage) GetRetryInterval(_ interfaces.Block) time.Duration {
	return 0
}
func (p *ProcessorVideoFromImage) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, bool, bool, string, int, error) {
	output := &bytes.Buffer{}
	blockConfig := &BlockVideoFromImageConfig{}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockVideoFromImage).GetBlockConfig(_config)
	userBlockConfig := &BlockVideoFromImageConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	imageBytes, err := helpers.GetValue[[]byte](_data, "image")
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

	// Create a temporary file to store the image from the buffer
	tempImageFile, err := os.CreateTemp("", "image-*.jpg")
	if err != nil {
		return nil, false, false, "", -1, err
	}
	defer os.Remove(tempImageFile.Name())

	// Write the buffer to the temporary file
	imgBuf := bytes.NewBuffer(imageBytes)
	_, err = tempImageFile.Write(imgBuf.Bytes())
	if err != nil {
		return nil, false, false, "", -1, err
	}
	tempImageFile.Close()

	// Create a temporary file to store the output video
	tempVideoFile, err := os.CreateTemp("", "output-*.mp4")
	if err != nil {
		return nil, false, false, "", -1, err
	}
	defer os.Remove(tempVideoFile.Name())

	// Calculate duration from start and end timestamps (seconds)
	duration := float64(blockConfig.End - blockConfig.Start)

	args := []string{
		"-y",         // Overwrite output file without asking
		"-loop", "1", // Loop the image
		"-t", fmt.Sprintf("%.3f", duration), // Set the duration for the image
		"-i", tempImageFile.Name(), // Input image from the temp file
		"-vf", fmt.Sprintf("fps=%d", blockConfig.FPS), // Set video FPS
		"-pix_fmt", "yuv420p", // Set pixel format for compatibility
		"-c:v", "libx264", // Video codec
		"-preset", blockConfig.Preset, // Set encoding speed
		"-crf", fmt.Sprintf("%d", blockConfig.CRF), // Set constant rate factor
		"-f", blockConfig.Format, // Output format as MP4
		tempVideoFile.Name(),
	}

	var stderr bytes.Buffer
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return nil, false, false, "", -1, err
	}

	videoBuffer, err := os.ReadFile(tempVideoFile.Name())
	if err != nil {
		return nil, false, false, "", -1, err
	}

	output.Write(videoBuffer)

	return output, false, false, "", -1, nil
}

type BlockVideoFromImageConfig struct {
	FFMPEGBinary string  `yaml:"ffmpeg_binary" json:"ffmpeg_binary"`
	FPS          int     `yaml:"fps" json:"fps"`
	Format       string  `yaml:"format" json:"format"`
	Preset       string  `yaml:"preset" json:"preset"`
	CRF          int     `yaml:"crf" json:"crf"`
	Start        float64 `yaml:"start" json:"start"`
	End          float64 `yaml:"end" json:"end"`
}

type BlockVideoFromImage struct {
	generics.ConfigurableBlock[BlockVideoFromImageConfig]
	BlockParent
}

var _ interfaces.Block = (*BlockVideoFromImage)(nil)

func (b *BlockVideoFromImage) GetBlockConfig(_config config.Config) *BlockVideoFromImageConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockVideoFromImageConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockVideoFromImage() *BlockVideoFromImage {
	block := &BlockVideoFromImage{
		BlockParent: BlockParent{
			Id:          "video_from_image",
			Name:        "Video from image",
			Description: "Make video from image",
			Version:     "1",
			SchemaString: `{
				"type": "object",
				"properties": {
					"input":{
						"type": "object",
						"description": "Input data",
						"properties": {
							"image": {
								"description": "Image be used as input",
								"type": "string",
								"format": "file"
							},
							"start": {
								"description": "Start time in seconds",
								"type": "number",
								"format": "float",
								"default": 0.0
							},
							"end": {
								"description": "End time in seconds",
								"type": "number",
								"format": "float",
								"default": 1.0
							},
							"fps": {
								"description": "Frames per second",
								"type": "number",
								"format": "integer",
								"default": 30,
								"minimum": 1,
								"maximum": 60
							},
							"format": {
								"description": "Output format",
								"type": "string",
								"default": "mp4",
								"enum": ["mp4"]
							},
							"preset": {
								"description": "Encoding speed",
								"type": "string",
								"default": "veryfast",
								"enum": ["veryfast"]
							},
							"crf": {
								"description": "Constant rate factor",
								"type": "number",
								"format": "integer",
								"default": 23,
								"minimum": 23,
								"maximum": 23
							}
						},
						"required": ["image", "start", "end"]
					},
					"output": {
						"description": "Video generated from image",
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

	block.SetProcessor(NewProcessorVideoFromImage())

	return block
}
