package interfaces

import (
	"bytes"
	"context"
	"time"

	"github.com/google/uuid"
)

type ProcessingStatus int
type ContextKeyProcessingID struct{}
type ContextKeyProcessingInstanceID struct{}

const (
	ProcessingStatusUnknown ProcessingStatus = iota
	ProcessingStatusPending
	ProcessingStatusRunning
	ProcessingStatusCompleted
	ProcessingStatusFailed
	ProcessingStatusTransferred
	ProcessingStatusStopped
	ProcessingStatusRetry
)

type Processing interface {
	GetId() uuid.UUID
	GetInstanceId() uuid.UUID
	GetPipeline() Pipeline
	GetBlock() Block
	GetData() ProcessableBlockData
	GetStatus() ProcessingStatus
	SetStatus(ProcessingStatus)
	SetError(error)
	GetError() error
	GetOutput() ProcessingOutput
	GetProcessingTime() time.Duration

	SetRegistryNotificationChannel(chan Processing)

	Start() (ProcessingOutput, bool, error)
	Shutdown(context.Context) error

	// Stop pending processing
	Stop(ProcessingStatus, error)
}

type ProcessingOutput interface {
	GetId() string
	GetStop() bool
	GetValue() *bytes.Buffer
}
