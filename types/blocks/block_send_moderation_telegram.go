package blocks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"regexp"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/generics"
	"data-pipelines-worker/types/helpers"
	"data-pipelines-worker/types/interfaces"
)

type TelegramReviewMessage struct {
	Text                string
	ProcessingID        string
	BlockSlug           string
	RegenerateBlockSlug string // Optional
	Buttons             []tgbotapi.InlineKeyboardButton
	Index               int
}

func GenerateTelegramReviewMessage(review TelegramReviewMessage) string {
	template := `Please review: %s
ProcessingId: %s
BlockSlug: %s
Index: %d`
	if review.RegenerateBlockSlug != "" {

		template += `
RegenerateBlockSlug: %s`
		return fmt.Sprintf(template, review.Text, review.ProcessingID, review.BlockSlug, review.Index, review.RegenerateBlockSlug)
	}
	return fmt.Sprintf(template, review.Text, review.ProcessingID, review.BlockSlug, review.Index)
}

func CreateTelegramReviewButton(label, action string, index int) tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("%s:%d", action, index))
}

// For tests only
func FormatTelegramMessage(message string) string {
	return strings.ReplaceAll(message, "\n", "\\n") // Escape newline characters
}

func ParseTelegramReviewMessage(message string) (TelegramReviewMessage, error) {
	reText := regexp.MustCompile(`Please review: (.+?)\nProcessingId:`)
	reProcessingID := regexp.MustCompile(`ProcessingId: ([^\n]+)`)
	reBlockSlug := regexp.MustCompile(`BlockSlug: ([^\n]+)`)
	reIndex := regexp.MustCompile(`Index: (\d+)`) // New regex for Index

	reRegenerateBlockSlug := regexp.MustCompile(`RegenerateBlockSlug: (.+)`)

	parsedMessage := TelegramReviewMessage{}

	if textMatch := reText.FindStringSubmatch(message); len(textMatch) > 1 {
		parsedMessage.Text = textMatch[1]
	} else {
		return parsedMessage, fmt.Errorf("missing or malformed text")
	}

	if processingIDMatch := reProcessingID.FindStringSubmatch(message); len(processingIDMatch) > 1 {
		parsedMessage.ProcessingID = processingIDMatch[1]
	} else {
		return parsedMessage, fmt.Errorf("missing or malformed processing ID")
	}

	if blockSlugMatch := reBlockSlug.FindStringSubmatch(message); len(blockSlugMatch) > 1 {
		parsedMessage.BlockSlug = blockSlugMatch[1]
	} else {
		return parsedMessage, fmt.Errorf("missing or malformed block slug")
	}

	if indexMatch := reIndex.FindStringSubmatch(message); len(indexMatch) > 1 {
		var err error
		parsedMessage.Index, err = strconv.Atoi(indexMatch[1])
		if err != nil {
			return parsedMessage, fmt.Errorf("invalid index value: %s", indexMatch[1])
		}
	} else {
		return parsedMessage, fmt.Errorf("missing or malformed index")
	}

	if regenerateBlockSlugMatch := reRegenerateBlockSlug.FindStringSubmatch(message); len(regenerateBlockSlugMatch) > 1 {
		parsedMessage.RegenerateBlockSlug = regenerateBlockSlugMatch[1]
	}

	return parsedMessage, nil
}

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
) (*bytes.Buffer, bool, bool, string, int, error) {
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
		return output, false, false, "", -1, errors.New("telegram client is not configured")
	}

	review := TelegramReviewMessage{
		Text:                blockConfig.Text,
		ProcessingID:        processingID.String(),
		BlockSlug:           data.GetSlug(),
		RegenerateBlockSlug: blockConfig.RegenerateBlockSlug,
		Index:               data.GetInputIndex(),
	}
	buttons := []tgbotapi.InlineKeyboardButton{
		CreateTelegramReviewButton(
			blockConfig.Approve,
			ShortenedActionApprove,
			data.GetInputIndex(),
		),
		CreateTelegramReviewButton(
			blockConfig.Decline,
			ShortenedActionDecline,
			data.GetInputIndex(),
		),
	}

	// Add extra decisions
	for action, label := range blockConfig.ExtraDecisions {
		switch action {
		case blockConfig.Regenerate:
			if label == "" {
				label = blockConfig.Regenerate
			}

			buttons = append(
				buttons,
				CreateTelegramReviewButton(
					label,
					ShortenedActionRegenerate,
					data.GetInputIndex(),
				),
			)
		}
	}
	review.Buttons = buttons

	keyboard := tgbotapi.NewInlineKeyboardMarkup(review.Buttons)

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
			photo.Caption = GenerateTelegramReviewMessage(review)
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
			GenerateTelegramReviewMessage(review),
		)
		msg.ReplyMarkup = keyboard
		sentMessage, err = client.Send(msg)
	}

	if err != nil {
		return output, false, false, "", -1, err
	}

	sentMessageBytes, err := json.Marshal(map[string]interface{}{
		"sentMessage": sentMessage,
		"sentButtons": buttons,
	})
	if err != nil {
		return output, false, false, "", -1, err
	}
	output = bytes.NewBuffer(sentMessageBytes)

	return output, false, false, "", -1, nil
}

type BlockSendModerationToTelegramConfig struct {
	Text                string            `yaml:"-" json:"text"`
	GroupId             int64             `yaml:"group_id" json:"group_id"`
	RegenerateBlockSlug string            `yaml:"-" json:"regenerate_block_slug"`
	Approve             string            `yaml:"approve" json:"-"`         // Mapping of the buttons to the actions
	Decline             string            `yaml:"decline" json:"-"`         // Mapping of the buttons to the actions
	Regenerate          string            `yaml:"regenerate" json:"-"`      // Mapping of the buttons to the actions
	ExtraDecisions      map[string]string `yaml:"-" json:"extra_decisions"` // Includes Regenerate
}

type BlockSendModerationToTelegram struct {
	generics.ConfigurableBlock[BlockSendModerationToTelegramConfig] `json:"-" yaml:"-"`
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
			Id:          "send_moderation_tg",
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
							"regenerate_block_slug": {
								"description": "Slug of the block to use in regenerate request",
								"type": "string"
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
