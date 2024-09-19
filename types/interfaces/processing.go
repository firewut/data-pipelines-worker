package interfaces

import (
	"bytes"
	"context"

	"github.com/google/uuid"
)

type ProcessingStatus int

const (
	ProcessingStatusUnknown ProcessingStatus = iota
	ProcessingStatusPending
	ProcessingStatusRunning
	ProcessingStatusCompleted
	ProcessingStatusFailed
)

type Processing interface {
	GetId() uuid.UUID
	GetPipeline() Pipeline
	GetBlock() Block
	GetData() ProcessableBlockData
	GetStatus() ProcessingStatus
	SetStatus(ProcessingStatus)
	GetOutput() ProcessingOutput

	SetRegistryNotificationChannel(chan Processing)

	Start() (ProcessingOutput, error)
	Shutdown(context.Context) error
}

type ProcessingOutput interface {
	GetId() string
	GetValue() *bytes.Buffer
}
