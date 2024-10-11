package blocks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

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

	approveButton := tgbotapi.NewInlineKeyboardButtonData(
		blockConfig.Approve,
		fmt.Sprintf("approve:%s", processingID),
	)
	rejectButton := tgbotapi.NewInlineKeyboardButtonData(
		blockConfig.Reject,
		fmt.Sprintf("reject:%s", processingID),
	)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(approveButton, rejectButton),
	)
	msg := tgbotapi.NewMessage(
		blockConfig.ChannelId,
		fmt.Sprintf("Please review the:\n\n%s\n\nID: %s", textContent, processingID),
	)
	msg.ReplyMarkup = keyboard

	sentMessage, err := client.Send(msg)
	if err != nil {
		return output, false, false, err
	}

	sentMessageBytes, err := json.Marshal(sentMessage)
	if err != nil {
		return output, false, false, err
	}

	output = bytes.NewBuffer(sentMessageBytes)
	return output, false, false, err
}

type BlockSendModerationToTelegramConfig struct {
	Text      string `yaml:"text" json:"text"`
	ChannelId int64  `yaml:"channel_id" json:"channel_id"`
	Approve   string `yaml:"approve" json:"-"`
	Reject    string `yaml:"reject" json:"-"`
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
			Id:          "send_moderation_to_telegram",
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
								"description": "Text to convert to audio",
								"type": "string",
								"minLength": 10
							},
							"channel_id": {
								"description": "Channel ID to send the message to",
								"type": "integer"
							}
						},
						"required": ["text", "channel_id"]
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
