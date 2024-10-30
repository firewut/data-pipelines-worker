package blocks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/generics"
	"data-pipelines-worker/types/helpers"
	"data-pipelines-worker/types/interfaces"
)

type TelegramMessage struct {
	Text         string
	Media        string
	ProcessingID string
	BlockSlug    string
	Index        int
}

func GenerateTelegramMessage(message TelegramMessage) string {
	template := `%s
ProcessingId: %s
BlockSlug: %s
Index: %d`
	return fmt.Sprintf(template, message.Text, message.ProcessingID, message.BlockSlug, message.Index)
}

type ProcessorSendMessageToTelegram struct {
}

func NewProcessorSendMessageToTelegram() *ProcessorSendMessageToTelegram {
	return &ProcessorSendMessageToTelegram{}
}

func (p *ProcessorSendMessageToTelegram) GetRetryCount(_ interfaces.Block) int {
	return 0
}

func (p *ProcessorSendMessageToTelegram) GetRetryInterval(_ interfaces.Block) time.Duration {
	return 0
}

func (p *ProcessorSendMessageToTelegram) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, bool, bool, string, int, error) {
	output := &bytes.Buffer{}
	blockConfig := &BlockSendMessageToTelegramConfig{}

	processingID := uuid.Nil
	if ctx.Value(interfaces.ContextKeyProcessingID{}) != nil {
		processingID = ctx.Value(interfaces.ContextKeyProcessingID{}).(uuid.UUID)
	}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockSendMessageToTelegram).GetBlockConfig(_config)
	userBlockConfig := &BlockSendMessageToTelegramConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	client := _config.Telegram.GetClient()
	if client == nil {
		return output, false, false, "", -1, errors.New("telegram client is not configured")
	}

	message := TelegramMessage{
		Text:         blockConfig.Text,
		ProcessingID: processingID.String(),
		BlockSlug:    data.GetSlug(),
		Index:        data.GetInputIndex(),
	}

	// Initialize a variable for the sent message and error handling
	var sentMessage tgbotapi.Message
	var err error

	// Attempt to retrieve and decode the image
	imageBytes, imgErr := helpers.GetValue[[]byte](_data, "image")
	videoBytes, videoErr := helpers.GetValue[[]byte](_data, "video")

	if imgErr == nil {
		imgBuf := bytes.NewBuffer(imageBytes)
		_, format, decodeErr := image.Decode(imgBuf)
		if decodeErr == nil {
			// Valid image, prepare the photo message
			imgFile := tgbotapi.FileBytes{
				Name:  fmt.Sprintf("image.%s", format),
				Bytes: imageBytes,
			}
			photo := tgbotapi.NewPhoto(blockConfig.GroupId, imgFile)
			photo.Caption = GenerateTelegramMessage(message)
			sentMessage, err = client.Send(photo)
		}
	}

	// If there was an error with the image or it was invalid, try to send a video
	if err != nil || imgErr != nil {
		if videoErr == nil && len(videoBytes) > 0 {
			videoFile := tgbotapi.FileBytes{
				Name:  "video.mp4",
				Bytes: videoBytes,
			}
			video := tgbotapi.NewVideo(blockConfig.GroupId, videoFile)
			video.Caption = GenerateTelegramMessage(message)
			sentMessage, err = client.Send(video)
		}
	}

	// If there's no image, no video, or an error occurred during image/video handling, send text message
	if err != nil {
		msg := tgbotapi.NewMessage(
			blockConfig.GroupId,
			GenerateTelegramMessage(message),
		)
		sentMessage, err = client.Send(msg)
	}

	if err != nil {
		return output, false, false, "", -1, err
	}

	sentMessageBytes, err := json.Marshal(map[string]interface{}{
		"sentMessage": sentMessage,
	})
	if err != nil {
		return output, false, false, "", -1, err
	}
	output = bytes.NewBuffer(sentMessageBytes)

	return output, false, false, "", -1, nil
}

type BlockSendMessageToTelegramConfig struct {
	Text    string `yaml:"-" json:"text"`
	GroupId int64  `yaml:"group_id" json:"group_id"`
}

type BlockSendMessageToTelegram struct {
	generics.ConfigurableBlock[BlockSendMessageToTelegramConfig]
	BlockParent
}

var _ interfaces.Block = (*BlockSendMessageToTelegram)(nil)

func (b *BlockSendMessageToTelegram) GetBlockConfig(_config config.Config) *BlockSendMessageToTelegramConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockSendMessageToTelegramConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockSendMessageToTelegram() *BlockSendMessageToTelegram {
	block := &BlockSendMessageToTelegram{
		BlockParent: BlockParent{
			Id:          "send_message_tg",
			Name:        "Send Message to Telegram",
			Description: "Send Message to Telegram. In case of an image or video, the priority is given to the image.",
			Version:     "1",
			SchemaString: `{
				"type": "object",
				"properties": {
					"input": {
						"type": "object",
						"description": "Input parameters",
						"properties": {
							"text": {
								"description": "Text content",
								"type": "string",
								"minLength": 10
							},
							"image": {
					        	"description": "Image content",
								"type": "string",
								"format": "file"
							},
							"video": {
								"description": "Video content",
								"type": "string",
								"format": "file"
							},
							"group_id": {
								"description": "Group ID to send the message to",
								"type": "integer"
							}
						},
						"required": ["text", "group_id"]
					},
					"output": {
						"description": "Send result output",
						"type": "string",
						"format": "file"
					}
				},
				"required": ["input"]
			}`,
			SchemaPtr: nil,
			Schema:    nil,
		},
	}

	if err := block.ApplySchema(block.GetSchemaString()); err != nil {
		panic(err)
	}

	block.SetProcessor(NewProcessorSendMessageToTelegram())

	return block
}
