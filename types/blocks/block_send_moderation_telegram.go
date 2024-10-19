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

type DetectorTelegramBot struct {
	BlockDetectorParent

	Client *tgbotapi.BotAPI
}

func NewDetectorTelegramBot(client *tgbotapi.BotAPI, detectorConfig config.BlockConfigDetector) *DetectorTelegramBot {
	return &DetectorTelegramBot{
		BlockDetectorParent: NewDetectorParent(detectorConfig),
		Client:              client,
	}
}

func (d *DetectorTelegramBot) Detect() bool {
	d.Lock()
	defer d.Unlock()

	if d.Client == nil {
		return false
	}

	_, err := d.Client.GetMe()
	return err == nil
}

type ProcessorSendModerationToTelegram struct {
}

func NewProcessorSendModerationToTelegram() *ProcessorSendModerationToTelegram {
	return &ProcessorSendModerationToTelegram{}
}

func (p *ProcessorSendModerationToTelegram) GetRetryCount(_ interfaces.Block) int {
	return 0
}

func (p *ProcessorSendModerationToTelegram) GetRetryInterval(_ interfaces.Block) time.Duration {
	return 0
}

func (p *ProcessorSendModerationToTelegram) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) (*bytes.Buffer, bool, bool, error) {
	output := &bytes.Buffer{}
	blockConfig := &BlockSendModerationToTelegramConfig{}

	processingID := uuid.Nil
	if ctx.Value(interfaces.ContextKeyProcessingID{}) != nil {
		processingID = ctx.Value(interfaces.ContextKeyProcessingID{}).(uuid.UUID)
	}

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockSendModerationToTelegram).GetBlockConfig(_config)
	userBlockConfig := &BlockSendModerationToTelegramConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	client := _config.Telegram.GetClient()
	if client == nil {
		return output, false, false, errors.New("telegram client is not configured")
	}

	textContent := blockConfig.Text
	buttons := []tgbotapi.InlineKeyboardButton{}
	approveButton := tgbotapi.NewInlineKeyboardButtonData(
		blockConfig.Approve,
		helpers.CreateCallbackData(
			ShortenedActionApprove,
			data.GetInputIndex(),
			processingID.String(),
			data.GetSlug(),
		),
	)
	declineButton := tgbotapi.NewInlineKeyboardButtonData(
		blockConfig.Decline,
		helpers.CreateCallbackData(
			ShortenedActionDecline,
			data.GetInputIndex(),
			processingID.String(),
			data.GetSlug(),
		),
	)
	buttons = append(buttons, approveButton, declineButton)

	// Add extra decisions
	for action, label := range blockConfig.ExtraDecisions {
		switch action {
		case blockConfig.Regenerate:
			if label == "" {
				label = blockConfig.Regenerate
			}

			regenerateButton := tgbotapi.NewInlineKeyboardButtonData(
				label,
				helpers.CreateCallbackData(
					ShortenedActionRegenerate,
					data.GetInputIndex(),
					processingID.String(),
					data.GetSlug(),
				),
			)
			buttons = append(buttons, regenerateButton)
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(buttons...),
	)

	// Initialize a variable for the sent message and error handling
	var sentMessage tgbotapi.Message
	var err error

	// Attempt to retrieve and decode the image
	imageBytes, imgErr := helpers.GetValue[[]byte](_data, "image")

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
			photo.Caption = fmt.Sprintf(
				"Please review the:\n\n%s\n\nID: %s.",
				textContent,
				processingID,
			)
			photo.ReplyMarkup = keyboard
			sentMessage, err = client.Send(photo)
		} else {
			// If decoding failed, fallback to text message
			err = decodeErr
		}
	}

	// If there's no image or an error occurred during image handling, send text message
	if err != nil || imgErr != nil {
		msg := tgbotapi.NewMessage(
			blockConfig.GroupId,
			fmt.Sprintf(
				"Please review the:\n\n%s\n\nID: %s.",
				textContent,
				processingID,
			),
		)
		msg.ReplyMarkup = keyboard
		sentMessage, err = client.Send(msg)
	}

	if err != nil {
		return output, false, false, err
	}

	sentMessageBytes, err := json.Marshal(map[string]interface{}{
		"sentMessage": sentMessage,
		"sentButtons": buttons,
	})
	if err != nil {
		return output, false, false, err
	}
	output = bytes.NewBuffer(sentMessageBytes)

	return output, false, false, nil
}

type BlockSendModerationToTelegramConfig struct {
	Text           string            `yaml:"-" json:"text"`
	GroupId        int64             `yaml:"group_id" json:"group_id"`
	Approve        string            `yaml:"approve" json:"-"`         // Mapping of the buttons to the actions
	Decline        string            `yaml:"decline" json:"-"`         // Mapping of the buttons to the actions
	Regenerate     string            `yaml:"regenerate" json:"-"`      // Mapping of the buttons to the actions
	ExtraDecisions map[string]string `yaml:"-" json:"extra_decisions"` // Includes Regenerate
}

type BlockSendModerationToTelegram struct {
	generics.ConfigurableBlock[BlockSendModerationToTelegramConfig]
	BlockParent
}

var _ interfaces.Block = (*BlockSendModerationToTelegram)(nil)

func (b *BlockSendModerationToTelegram) GetBlockConfig(_config config.Config) *BlockSendModerationToTelegramConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockSendModerationToTelegramConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockSendModerationToTelegram() *BlockSendModerationToTelegram {
	block := &BlockSendModerationToTelegram{
		BlockParent: BlockParent{
			Id:          "send_moderation_telegram",
			Name:        "Send Moderation to Telegram",
			Description: "Send Moderation Request to Telegram",
			Version:     "1",
			SchemaString: `{
				"type": "object",
				"properties": {
					"input": {
						"type": "object",
						"description": "Input parameters",
						"properties": {
							"text": {
								"description": "Text content to be moderated",
								"type": "string",
								"minLength": 10
							},
							"image": {
					        	"description": "Image content to be moderated",
								"type": "string",
								"format": "file"
							},
							"extra_decisions": {
								"description": "Extra decisions to be made",
								"type": "object",
								"properties": {
									"regenerate": {
										"description": "Regenerate the Content",
										"type": "string",
										"default": "Regenerate"
									}
								}
							},
							"group_id": {
								"description": "Group ID to send the message to",
								"type": "integer"
							}
						},
						"required": ["text", "group_id"]
					},
					"output": {
						"description": "Moderation request output",
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

	block.SetProcessor(NewProcessorSendModerationToTelegram())

	return block
}
