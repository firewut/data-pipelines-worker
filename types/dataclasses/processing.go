package dataclasses

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"data-pipelines-worker/types/config"
	"data-pipelines-worker/types/interfaces"
)

type Processing struct {
	sync.Mutex

	Id         uuid.UUID
	instanceId uuid.UUID
	status     interfaces.ProcessingStatus

	pipeline  interfaces.Pipeline
	block     interfaces.Block
	processor interfaces.BlockProcessor
	data      interfaces.ProcessableBlockData

	output                      *ProcessingOutput
	stop                        bool
	err                         error
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
	instanceId := uuid.New()

	ctx, ctxCancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, interfaces.ContextKeyProcessingID{}, id)
	ctx = context.WithValue(ctx, interfaces.ContextKeyProcessingInstanceID{}, instanceId)

	return &Processing{
		Id:                          id,
		instanceId:                  instanceId,
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

func (p *Processing) GetInstanceId() uuid.UUID {
	p.Lock()
	defer p.Unlock()

	return p.instanceId
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

func (p *Processing) Start() (interfaces.ProcessingOutput, bool, error) {
	logger := config.GetLogger()

	if p.GetStatus() != interfaces.ProcessingStatusPending {
		p.sendResult(false)
		return nil, false, fmt.Errorf("processing with id %s is not in pending state", p.GetId().String())
	}
	p.SetStatus(interfaces.ProcessingStatusRunning)

	retryCount := p.processor.GetRetryCount(p.block)
	retryInterval := p.processor.GetRetryInterval(p.block)

	var processResult *bytes.Buffer
	var stop, retry bool
	var err error

	for attempt := 0; attempt <= retryCount; attempt++ {
		processResult, stop, retry, err = p.block.Process(p.ctx, p.processor, p.data)

		if err == nil && !retry {
			break
		}

		if err == context.Canceled {
			p.SetStatus(interfaces.ProcessingStatusFailed)
			p.sendResult(true)
			return nil, false, fmt.Errorf("processing with id %s was cancelled", p.GetId().String())
		}

		// If retry is required and we haven't exhausted retry attempts
		if retry && attempt < retryCount {
			p.SetStatus(interfaces.ProcessingStatusRetry)

			logger.Warnf(
				"processing with id %s requires retry, attempt %d of %d",
				p.GetId().String(),
				attempt+1,
				retryCount,
			)

			time.Sleep(retryInterval)
			continue
		}

		// If we reach here and retry is still required, mark the process as failed
		if attempt == retryCount && retry {
			p.SetStatus(interfaces.ProcessingStatusRetryFailed)
			p.sendResult(false)
			return nil, false, fmt.Errorf("processing with id %s failed after exhausting all %d retry attempts", p.GetId().String(), retryCount)
		}

		// If unrecoverable error, mark as failed
		if err != nil {
			p.SetStatus(interfaces.ProcessingStatusFailed)
			p.sendResult(false)
			return nil, false, fmt.Errorf("processing with id %s failed after %d attempts: %w", p.GetId().String(), attempt+1, err)
		}
	}

	// Processing completed without errors or retries
	p.Lock()

	// Create a ProcessingOutput using the result from the block process
	p.output = NewProcessingOutput(p.data.GetSlug(), stop, processResult)
	p.stop = stop
	p.status = interfaces.ProcessingStatusCompleted
	if stop {
		p.status = interfaces.ProcessingStatusStopped
	}

	p.sendResult(false)
	p.Unlock()

	return p.output, stop, nil
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
	stop      bool
	data      *bytes.Buffer
}

func NewProcessingOutput(blockSlug string, stop bool, data *bytes.Buffer) *ProcessingOutput {
	return &ProcessingOutput{
		blockSlug: blockSlug,
		stop:      stop,
		data:      data,
	}
}

func (po *ProcessingOutput) GetId() string {
	return po.blockSlug
}

func (po *ProcessingOutput) GetStop() bool {
	return po.stop
}

func (po *ProcessingOutput) GetValue() *bytes.Buffer {
	return po.data
}
