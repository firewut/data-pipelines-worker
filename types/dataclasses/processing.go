package dataclasses

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"data-pipelines-worker/types/interfaces"
)

type Processing struct {
	sync.Mutex

	Id     uuid.UUID
	status interfaces.ProcessingStatus

	pipeline  interfaces.Pipeline
	block     interfaces.Block
	processor interfaces.BlockProcessor
	data      interfaces.ProcessableBlockData

	err                         error
	output                      *ProcessingOutput
	registryNotificationChannel chan interfaces.Processing
	startTimestamp              int64
	endTimestamp                int64

	ctx       context.Context
	ctxCancel context.CancelFunc
	closeOnce sync.Once
}

func NewProcessing(
	id uuid.UUID,
	pipeline interfaces.Pipeline,
	block interfaces.Block,
	data interfaces.ProcessableBlockData,
) *Processing {
	ctx, ctxCancel := context.WithCancel(context.Background())
	return &Processing{
		Id:                          id,
		status:                      interfaces.ProcessingStatusPending,
		pipeline:                    pipeline,
		block:                       block,
		processor:                   block.GetProcessor(),
		data:                        data,
		registryNotificationChannel: nil,
		ctx:                         ctx,
		ctxCancel:                   ctxCancel,
		output:                      nil,
		err:                         nil,
	}
}

func (p *Processing) GetId() uuid.UUID {
	p.Lock()
	defer p.Unlock()

	return p.Id
}

func (p *Processing) GetPipeline() interfaces.Pipeline {
	p.Lock()
	defer p.Unlock()

	return p.pipeline
}

func (p *Processing) GetBlock() interfaces.Block {
	p.Lock()
	defer p.Unlock()

	return p.block
}

func (p *Processing) Shutdown(ctx context.Context) error {
	if p.GetStatus() != interfaces.ProcessingStatusCompleted {
		p.SetStatus(interfaces.ProcessingStatusFailed)
	}
	p.ctxCancel()

	return nil
}

func (p *Processing) GetStatus() interfaces.ProcessingStatus {
	p.Lock()
	defer p.Unlock()

	return p.status
}

func (p *Processing) SetStatus(status interfaces.ProcessingStatus) {
	p.Lock()
	defer p.Unlock()

	p.status = status
	switch status {
	case interfaces.ProcessingStatusRunning:
		p.startTimestamp = time.Now().Unix()
	case interfaces.ProcessingStatusCompleted,
		interfaces.ProcessingStatusFailed,
		interfaces.ProcessingStatusTransferred:
		p.endTimestamp = time.Now().Unix()
	}
}

func (p *Processing) GetData() interfaces.ProcessableBlockData {
	p.Lock()
	defer p.Unlock()

	return p.data
}

func (p *Processing) GetOutput() interfaces.ProcessingOutput {
	p.Lock()
	defer p.Unlock()

	return p.output
}

func (p *Processing) SetRegistryNotificationChannel(channel chan interfaces.Processing) {
	p.Lock()
	defer p.Unlock()

	p.registryNotificationChannel = channel
}

func (p *Processing) Start() (interfaces.ProcessingOutput, error) {
	if p.GetStatus() != interfaces.ProcessingStatusPending {
		p.sendResult(false)
		return nil, fmt.Errorf("processing with id %s is not in pending state", p.Id.String())
	}
	p.SetStatus(interfaces.ProcessingStatusRunning)

	// Call Process and pass the processing context
	result, err := p.block.Process(p.ctx, p.processor, p.data)
	if err != nil {
		p.SetStatus(interfaces.ProcessingStatusFailed)
		if err == context.Canceled {
			// Handle cancellation specifically
			p.sendResult(true)
			return nil, fmt.Errorf("processing with id %s was cancelled", p.Id.String())
		}

		p.sendResult(false)
		return nil, err
	}

	p.Lock()
	p.status = interfaces.ProcessingStatusCompleted
	p.output = NewProcessingOutput(p.data.GetSlug(), result)

	p.sendResult(false)
	p.Unlock()

	return p.output, nil
}

func (p *Processing) Stop(status interfaces.ProcessingStatus, err error) {
	p.SetStatus(status)
	p.SetError(err)
	p.sendResult(false)
}

func (p *Processing) SetError(err error) {
	p.Lock()
	defer p.Unlock()

	p.err = err
}

func (p *Processing) GetError() error {
	p.Lock()
	defer p.Unlock()

	return p.err
}

func (p *Processing) sendResult(shutdown bool) {
	p.closeOnce.Do(func() {
		if shutdown {
			return
		}

		// Send to registryNotificationChannel only if it's open
		if p.registryNotificationChannel != nil {
			select {
			case p.registryNotificationChannel <- p:
			default:
				// Channel might be closed or not ready to receive, handle accordingly
			}
		}
	})
}

func (p *Processing) GetProcessingTime() time.Duration {
	p.Lock()
	defer p.Unlock()

	if p.startTimestamp == 0 {
		return 0
	}

	if p.endTimestamp == 0 {
		return time.Duration(
			time.Now().Unix() - p.startTimestamp,
		)
	}

	return time.Duration(p.endTimestamp - p.startTimestamp)
}

type ProcessingOutput struct {
	blockSlug string
	data      *bytes.Buffer
}

func NewProcessingOutput(blockSlug string, data *bytes.Buffer) *ProcessingOutput {
	return &ProcessingOutput{
		blockSlug: blockSlug,
		data:      data,
	}
}

func (po *ProcessingOutput) GetId() string {
	return po.blockSlug
}

func (po *ProcessingOutput) GetValue() *bytes.Buffer {
	return po.data
}
