package blocks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/generics"
	"data-pipelines-worker/types/helpers"
	"data-pipelines-worker/types/interfaces"
)

type acknowledgedCallbackData struct {
	sync.Mutex
	items map[string]bool
}

func newAcknowledgedCallbackData() *acknowledgedCallbackData {
	return &acknowledgedCallbackData{
		items: make(map[string]bool),
	}
}

func (a *acknowledgedCallbackData) get(key string) (bool, bool) {
	a.Lock()
	defer a.Unlock()

	val, exists := a.items[key]
	return val, exists
}

func (a *acknowledgedCallbackData) set(key string, value bool) {
	a.Lock()
	defer a.Unlock()

	a.items[key] = value
}

type ModerationAction string

const (
	ModerationActionApprove    ModerationAction = "approve"
	ModerationActionDecline    ModerationAction = "decline"
	ModerationActionUnknown    ModerationAction = "unknown"
	ModerationActionRegenerate ModerationAction = "regenerate"

	ShortenedActionApprove    = "a" // Shortened version for approve
	ShortenedActionDecline    = "d" // Shortened version for decline
	ShortenedActionRegenerate = "r" // Shortened version for regenerate
)

// Map for full action strings to moderation actions
var moderationActionMap = map[string]ModerationAction{
	string(ModerationActionApprove):    ModerationActionApprove,
	string(ModerationActionDecline):    ModerationActionDecline,
	string(ModerationActionUnknown):    ModerationActionUnknown,
	string(ModerationActionRegenerate): ModerationActionRegenerate,

	// Add shortened actions if you want to map them directly
	ShortenedActionApprove:    ModerationActionApprove,
	ShortenedActionDecline:    ModerationActionDecline,
	ShortenedActionRegenerate: ModerationActionRegenerate,
}

var acknowledgedCallbacks = newAcknowledgedCallbackData()

func GetModerationAction(action string) ModerationAction {
	if val, exists := moderationActionMap[action]; exists {
		return val
	}
	return ModerationActionUnknown
}

type ProcessorFetchModerationFromTelegram struct {
}

func NewProcessorFetchModerationFromTelegram() *ProcessorFetchModerationFromTelegram {
	return &ProcessorFetchModerationFromTelegram{}
}

func (p *ProcessorFetchModerationFromTelegram) GetRetryCount(block interfaces.Block) int {
	return block.(*BlockFetchModerationFromTelegram).GetBlockConfig(config.GetConfig()).RetryCount
}

func (p *ProcessorFetchModerationFromTelegram) GetRetryInterval(block interfaces.Block) time.Duration {
	return block.(*BlockFetchModerationFromTelegram).GetBlockConfig(config.GetConfig()).RetryInterval
}

func (p *ProcessorFetchModerationFromTelegram) Process(
	ctx context.Context,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) ([]*bytes.Buffer, bool, bool, string, int, error) {
	output := make([]*bytes.Buffer, 0)
	blockConfig := &BlockFetchModerationFromTelegramConfig{}

	processingID := uuid.Nil
	if ctx.Value(interfaces.ContextKeyProcessingID{}) != nil {
		processingID = ctx.Value(interfaces.ContextKeyProcessingID{}).(uuid.UUID)
	}
	moderationDecision := NewModerationDecision(processingID)

	_config := config.GetConfig()
	_data := data.GetInputData().(map[string]interface{})

	defaultBlockConfig := block.(*BlockFetchModerationFromTelegram).GetBlockConfig(_config)
	userBlockConfig := &BlockFetchModerationFromTelegramConfig{}
	helpers.MapToJSONStruct(_data, userBlockConfig)
	helpers.MergeStructs(defaultBlockConfig, userBlockConfig, blockConfig)

	// Stop the pipeline if Moderation is not Approved
	stopPipeline := blockConfig.StopPipelineIfDecline

	client := _config.Telegram.GetClient()
	if client == nil {
		return output, stopPipeline, false, "", -1, errors.New("telegram client is not configured")
	}

	updateConfig := tgbotapi.UpdateConfig{
		Offset:  0,   // Start with the first update
		Limit:   100, // Fetch up to 100 updates at a time
		Timeout: 5,   // Long polling timeout
	}

	shouldExit := false
	decisions := make([]ModerationAction, 0)

	var (
		moderationMessage  TelegramReviewMessage
		mostRecentDecision *tgbotapi.CallbackQuery
	)

	for !shouldExit {
		select {
		case <-ctx.Done():
			// Exit if context is canceled
			return output, stopPipeline, false, "", -1, ctx.Err()
		default:
			if shouldExit {
				break
			}

			// Fetch updates from Telegram
			updates, err := client.GetUpdates(updateConfig)
			if err != nil {
				return output, stopPipeline, true, "", -1, err
			}

			for _, update := range updates {
				if update.CallbackQuery != nil {
					callbackData := update.CallbackQuery.Data

					if _, exists := acknowledgedCallbacks.get(update.CallbackQuery.ID); exists {
						continue
					}

					parts := strings.Split(callbackData, ":")

					if len(parts) == 2 {
						buttonDecision := GetModerationAction(parts[0]) // "a", "d", "r"
						buttonIndex := parts[1]                         // "0", "1", "2", ...

						buttonIndexInt, err := strconv.Atoi(buttonIndex)
						if err != nil {
							fmt.Println("telegram index error converting button index to int:", err)
							continue
						}

						// Check if received processing ID matches the current one
						if buttonIndexInt == data.GetInputIndex() && buttonDecision != ModerationActionUnknown {
							// Get the message caption from the CallbackQuery
							messageText := update.CallbackQuery.Message.Text // Regular text message
							if messageText == "" {
								messageText = update.CallbackQuery.Message.Caption // Fallback to caption if text is empty
							}

							moderationMessage, err = ParseTelegramReviewMessage(messageText)
							if err != nil {
								fmt.Println("Error parsing Telegram message:", err)
								continue
							}

							if moderationMessage.ProcessingID != processingID.String() ||
								moderationMessage.BlockSlug != blockConfig.BlockSlug ||
								moderationMessage.Index != data.GetInputIndex() {
								continue
							}

							acknowledgedCallbacks.set(update.CallbackQuery.ID, true)

							mostRecentDecision = update.CallbackQuery
							decisions = append(decisions, buttonDecision)
						}
					}
				}
				// Set the offset to the last processed update's ID + 1
				updateConfig.Offset = update.UpdateID + 1
			}

			// If all updates have been iterated, exit the loop
			if len(updates) < updateConfig.Limit {
				shouldExit = true
			}
		}
	}

	// Get most recent decision
	if len(decisions) > 0 {
		moderationDecision.Action = decisions[len(decisions)-1]
		if mostRecentDecision != nil {
			// Remove keyboard from the message
			edit := tgbotapi.NewEditMessageReplyMarkup(
				mostRecentDecision.Message.Chat.ID,
				mostRecentDecision.Message.MessageID,
				tgbotapi.InlineKeyboardMarkup{
					InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{},
				},
			)

			if _, err := client.Request(edit); err == nil {
				// Acknowledge the callback (removes the loading indicator)
				callback := tgbotapi.NewCallback(mostRecentDecision.ID, "Processing...")
				client.Request(callback)
			}
		}

		if moderationDecision.Action == ModerationActionApprove ||
			moderationDecision.Action == ShortenedActionApprove {
			stopPipeline = false
		}
		if moderationDecision.Action == ModerationActionRegenerate {
			if moderationMessage.ProcessingID == processingID.String() {
				return output, true, false, moderationMessage.RegenerateBlockSlug, data.GetInputIndex(), nil
			}
		}
	}

	moderationDecisionBytes, err := json.Marshal(moderationDecision)
	if err != nil {
		return output, stopPipeline, false, "", -1, err
	}
	output = append(output, bytes.NewBuffer(moderationDecisionBytes))

	// Retry if the decision is unknown
	if blockConfig.RetryIfUnknown && moderationDecision.Action == ModerationActionUnknown {
		return output, false, true, "", -1, nil
	}

	return output, stopPipeline, false, "", -1, err
}

type BlockFetchModerationFromTelegramConfig struct {
	BlockSlug             string        `yaml:"block_slug" json:"block_slug"`
	StopPipelineIfDecline bool          `yaml:"stop_pipeline_if_decline" json:"stop_pipeline_if_decline"`
	RetryIfUnknown        bool          `yaml:"retry_if_unknown" json:"-"`
	RetryCount            int           `yaml:"retry_count" json:"-"`
	RetryInterval         time.Duration `yaml:"retry_interval" json:"-"`
}

type ModerationDecision struct {
	ProcessingID uuid.UUID        `json:"processing_id"`
	Action       ModerationAction `json:"action"`
}

func NewModerationDecision(processingID uuid.UUID) *ModerationDecision {
	return &ModerationDecision{
		ProcessingID: processingID,
		Action:       ModerationActionUnknown,
	}
}

type BlockFetchModerationFromTelegram struct {
	generics.ConfigurableBlock[BlockFetchModerationFromTelegramConfig] `json:"-" yaml:"-"`
	BlockParent
}

var _ interfaces.Block = (*BlockFetchModerationFromTelegram)(nil)

func (b *BlockFetchModerationFromTelegram) GetBlockConfig(_config config.Config) *BlockFetchModerationFromTelegramConfig {
	blockConfig := _config.Blocks[b.GetId()].Config

	defaultBlockConfig := &BlockFetchModerationFromTelegramConfig{}
	helpers.MapToYAMLStruct(blockConfig, defaultBlockConfig)

	return defaultBlockConfig
}

func NewBlockFetchModerationFromTelegram() *BlockFetchModerationFromTelegram {
	block := &BlockFetchModerationFromTelegram{
		BlockParent: BlockParent{
			Id:          "fetch_moderation_tg",
			Name:        "Fetch Moderation from Telegram",
			Description: "Fetch Moderation Action from Telegram",
			Version:     "1",
			SchemaString: `{
				"type": "object",
				"properties": {
					"input": {
						"type": "object",
						"description": "This block has no input parameters",
						"properties": {
							"block_slug": {
								"description": "Slug of a Block which has Sent moderation Request",
								"type": "string"
							},
							"stop_pipeline_if_decline": {
								"description": "Stop the pipeline if Moderation is Declined",
								"type": "boolean",
								"default": true
							}
						},
						"required": ["block_slug"]
					},
					"output": {
						"description": "Moderation action output",
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

	block.SetProcessor(NewProcessorFetchModerationFromTelegram())

	return block
}
