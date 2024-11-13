package blocks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/generics"
	"data-pipelines-worker/types/helpers"
	"data-pipelines-worker/types/interfaces"
)

type DetectorSubtitlesFromTranscription struct {
	BlockDetectorParent
}

func NewDetectorSubtitlesFromTranscription(
	detectorConfig config.BlockConfigDetector,
) *DetectorSubtitlesFromTranscription {
	return &DetectorSubtitlesFromTranscription{
		BlockDetectorParent: NewDetectorParent(detectorConfig),
	}
}

func (d *DetectorSubtitlesFromTranscription) Detect() bool {
	d.Lock()
	defer d.Unlock()

	return true
}

type ProcessorSubtitlesFromTranscription struct {
}

func NewProcessorSubtitlesFromTranscription() *ProcessorSubtitlesFromTranscription {
	return &ProcessorSubtitlesFromTranscription{}
}

func (p *ProcessorSubtitlesFromTranscription) GetRetryCount(_ interfaces.Block) int {
	return 0
}

func (p *ProcessorSubtitlesFromTranscription) GetRetryInterval(_ interfaces.Block) time.Duration {
	return 0
}

type transcriptionSegment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

type transcription struct {
	Segments []transcriptionSegment `json:"segments"`
}

type subtitlesFile interface {
	getType() string
	getBytes() []byte
	fromOpenAITranscription(transcription transcription, config *BlockSubtitlesFromTranscriptionConfig)
}

// Struct to combine everything
type subtitlesAssFile struct {
	ScriptInfo *subtitlesAssScriptInfo
	Style      *subtitlesAssStyle
	Events     []*transcriptionSegment
}

func newSubtitlesAssFile() *subtitlesAssFile {
	return &subtitlesAssFile{
		ScriptInfo: &subtitlesAssScriptInfo{
			Title:          "Transcription Subtitles",
			OriginalScript: "ChatGPT",
			ScriptType:     "v4.00+",
			Collisions:     "Normal",
			PlayDepth:      "0",
		},
		Style:  &subtitlesAssStyle{},
		Events: make([]*transcriptionSegment, 0),
	}
}

func (f *subtitlesAssFile) fromOpenAITranscription(transcription transcription, blockConfig *BlockSubtitlesFromTranscriptionConfig) {
	f.Style = &subtitlesAssStyle{
		Name:            blockConfig.Name,
		Fontname:        blockConfig.Fontname,
		Fontsize:        blockConfig.Fontsize,
		PrimaryColour:   blockConfig.PrimaryColour,
		SecondaryColour: blockConfig.SecondaryColour,
		BackColour:      blockConfig.BackColour,
		Bold:            blockConfig.Bold,
		Italic:          blockConfig.Italic,
		BorderStyle:     blockConfig.BorderStyle,
		Outline:         blockConfig.Outline,
		Shadow:          blockConfig.Shadow,
		Alignment:       blockConfig.Alignment,
		MarginL:         blockConfig.MarginL,
		MarginR:         blockConfig.MarginR,
		MarginV:         blockConfig.MarginV,
	}

	events := make([]*transcriptionSegment, 0)
	for _, segment := range transcription.Segments {
		// start := formatOpenAITranscriptionSegmentTime(segment.Start)
		// end := formatOpenAITranscriptionSegmentTime(segment.End)
		events = append(
			events, &transcriptionSegment{
				Start: segment.Start,
				End:   segment.End,
				Text:  segment.Text,
			},
		)
	}
	f.Events = events
}

func (f *subtitlesAssFile) getType() string {
	return "ass"
}

func (f *subtitlesAssFile) getBytes() []byte {
	buf := bytes.NewBufferString(``)

	content := fmt.Sprintf(`[Script Info]
Title: %s
Original Script: %s
ScriptType: %s
Collisions: %s
PlayDepth: %s

[Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, BackColour, Bold, Italic, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV
Style: %s,%s,%d,%s,%s,%s,%d,%d,%d,%.1f,%.1f,%d,%d,%d,%d

[Events]
Format: Marked, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text`,
		f.ScriptInfo.Title, f.ScriptInfo.OriginalScript, f.ScriptInfo.ScriptType,
		f.ScriptInfo.Collisions, f.ScriptInfo.PlayDepth,
		f.Style.Name, f.Style.Fontname, f.Style.Fontsize, f.Style.PrimaryColour,
		f.Style.SecondaryColour, f.Style.BackColour, f.Style.Bold, f.Style.Italic,
		f.Style.BorderStyle, f.Style.Outline, f.Style.Shadow, f.Style.Alignment,
		f.Style.MarginL, f.Style.MarginR, f.Style.MarginV)

	// Add the events
	for _, event := range f.Events {
		content += fmt.Sprintf("\nDialogue: 0,%s,%s,%s,,0,0,0,,%s",
			formatOpenAITranscriptionSegmentTime(event.Start),
			formatOpenAITranscriptionSegmentTime(event.End),
			f.Style.Name,
			event.Text,
		)
	}

	buf.WriteString(content)
	return buf.Bytes()
}

// Struct to represent script info section
type subtitlesAssScriptInfo struct {
	Title          string
	OriginalScript string
	ScriptType     string
	Collisions     string
	PlayDepth      string
}

// Struct to represent style section
type subtitlesAssStyle struct {
	Name            string
	Fontname        string
	Fontsize        int
	PrimaryColour   string
	SecondaryColour string
	BackColour      string
	Bold            int
	Italic          int
	BorderStyle     int
	Outline         float64
	Shadow          float64
	Alignment       int
	MarginL         int
	MarginR         int
	MarginV         int
}

// Function to format time as HH:MM:SS.MS (ASS subtitle format)
func formatOpenAITranscriptionSegmentTime(seconds float64) string {
	// Convert seconds to time.Duration
	d := time.Duration(seconds * float64(time.Second))
	// Format the duration into HH:MM:SS.MS (with exactly 2 digits after the decimal point)
	return fmt.Sprintf("%02d:%02d:%02d.%02d", int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60, (d.Milliseconds()%1000)/10)
}

func (p *ProcessorSubtitlesFromTranscription) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, bool, bool, string, int, error) {
	output := &bytes.Buffer{}
	blockConfig := &BlockSubtitlesFromTranscriptionConfig{}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockSubtitlesFromTranscription).GetBlockConfig(_config)
	userBlockConfig := &BlockSubtitlesFromTranscriptionConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	transcriptionBytes, err := helpers.GetValue[[]byte](_data, "transcription")
	if err != nil {
		return nil, false, false, "", -1, err
	}

	var (
		transcription transcription
		_             subtitlesFile
	)

	switch blockConfig.InputFormat {
	case "openai_verbose_json":
		// Parse the JSON data
		err := json.Unmarshal([]byte(transcriptionBytes), &transcription)
		if err != nil {
			return output, false, false, "", -1, err
		}
	}

	switch blockConfig.OutputFormat {
	case "ass":
		subtitlesFile := newSubtitlesAssFile()
		subtitlesFile.fromOpenAITranscription(transcription, blockConfig)
		output.Write(subtitlesFile.getBytes())
	case "srt":
		// Not yet implemented
	}

	return output, false, false, "", -1, nil
}

type BlockSubtitlesFromTranscriptionConfig struct {
	Transcription   string  `yaml:"-" json:"transcription"`
	InputFormat     string  `yaml:"input_format" json:"input_format"`
	OutputFormat    string  `yaml:"output_format" json:"output_format"`
	Name            string  `yaml:"name" json:"-"`
	Fontname        string  `yaml:"font_name" json:"font_name"`
	Fontsize        int     `yaml:"font_size" json:"font_size"`
	PrimaryColour   string  `yaml:"primary_colour" json:"primary_colour"`
	SecondaryColour string  `yaml:"secondary_colour" json:"secondary_colour"`
	BackColour      string  `yaml:"back_colour" json:"back_colour"`
	Bold            int     `yaml:"bold" json:"bold"`
	Italic          int     `yaml:"italic" json:"italic"`
	BorderStyle     int     `yaml:"border_style" json:"border_style"`
	Outline         float64 `yaml:"outline" json:"outline"`
	Shadow          float64 `yaml:"shadow" json:"shadow"`
	Alignment       int     `yaml:"alignment" json:"alignment"`
	MarginL         int     `yaml:"margin_l" json:"margin_l"`
	MarginR         int     `yaml:"margin_r" json:"margin_r"`
	MarginV         int     `yaml:"margin_v" json:"margin_v"`
}

type BlockSubtitlesFromTranscription struct {
	generics.ConfigurableBlock[BlockSubtitlesFromTranscriptionConfig] `json:"-" yaml:"-"`
	BlockParent
}

var _ interfaces.Block = (*BlockSubtitlesFromTranscription)(nil)

func (b *BlockSubtitlesFromTranscription) GetBlockConfig(_config config.Config) *BlockSubtitlesFromTranscriptionConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockSubtitlesFromTranscriptionConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockSubtitlesFromTranscription() *BlockSubtitlesFromTranscription {
	block := &BlockSubtitlesFromTranscription{
		BlockParent: BlockParent{
			Id:          "subtitles_from_transcription",
			Name:        "Subtitles from Transcription",
			Description: "Make Subtitles from Transcription",
			Version:     "1",
			SchemaString: `{
				"type": "object",
				"properties": {
					"input":{
						"type": "object",
						"description": "Input data",
						"properties": {
							"transcription": {
								"description": "Transcription file",
								"type": "string",
								"format": "file"
							},
							"input_format": {
								"description": "Transcription input format",
								"type": "string",
								"default": "openai_verbose_json",
								"enum": ["openai_verbose_json"]
							},
							"output_format": {
								"description": "Subtitles output format",
								"type": "string",
								"default": "ass",
								"enum": ["ass"]
							},
							"name": {
								"type": "string",
								"description": "The name of the text style.",
								"default": "Default",
								"enum": ["Default"]
							},
							"font_name": {
								"type": "string",
								"description": "The name of the font.",
								"default": "Arial",
								"enum": ["Arial"]
							},
							"font_size": {
								"type": "integer",
								"default": 30,
								"description": "The size of the font."
							},
							"primary_colour": {
								"type": "string",
								"default": "&H00FFFFFF",
								"description": "The primary color of the text."
							},
							"secondary_colour": {
								"type": "string",
								"default": "&H00000000",
								"description": "The secondary color of the text."
							},
							"back_colour": {
								"type": "string",
								"default": "&H00000000",
								"description": "The background color of the text."
							},
							"bold": {
								"type": "integer",
								"default": -1,
								"description": "Whether the text is bold. -1 for true, 0 for false."
							},
							"italic": {
								"type": "integer",
								"default": 0,
								"description": "Whether the text is italic. 0 for false, 1 for true."
							},
							"border_style": {
								"type": "integer",
								"default": 1,
								"description": "The style of the border."
							},
							"outline": {
								"type": "number",
								"default": 1.0,
								"description": "The width of the outline."
							},
							"shadow": {
								"type": "number",
								"default": 0.0,
								"description": "The size of the shadow."
							},
							"alignment": {
								"type": "integer",
								"default": 2,
								"description": "The alignment of the text."
							},
							"margin_l": {
								"type": "integer",
								"default": 10,
								"description": "The left margin of the text."
							},
							"margin_r": {
								"type": "integer",
								"default": 10,
								"description": "The right margin of the text."
							},
							"margin_v": {
								"type": "integer",
								"default": 10,
								"description": "The vertical margin of the text."
							}
						},
						"required": ["transcription"]
					},
					"output": {
						"description": "Subtitles generated from Transcription",
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

	block.SetProcessor(NewProcessorSubtitlesFromTranscription())

	return block
}
