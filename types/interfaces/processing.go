package interfaces

import (
	"bytes"
	"context"
	"time"

	"github.com/google/uuid"
)

type ProcessingStatus int

const (
	ProcessingStatusUnknown ProcessingStatus = iota
	ProcessingStatusPending
	ProcessingStatusRunning
	ProcessingStatusCompleted
	ProcessingStatusFailed
	ProcessingStatusTransferred
)

type Processing interface {
	GetId() uuid.UUID
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

	Start() (ProcessingOutput, error)
	Shutdown(context.Context) error

	// Stop unstarted processing
	Stop(ProcessingStatus, error)
}

type ProcessingOutput interface {
	GetId() string
	GetValue() *bytes.Buffer
}
