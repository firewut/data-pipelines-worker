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

	pipeline   interfaces.Pipeline
	block      interfaces.Block
	processor  interfaces.BlockProcessor
	blockData  interfaces.ProcessableBlockData
	inputIndex int

	output                      *ProcessingOutput
	registryNotificationChannel chan interfaces.Processing
	channelClosed               bool
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
	blockData interfaces.ProcessableBlockData,
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
		blockData:                   blockData,
		inputIndex:                  blockData.GetInputIndex(),
		channelClosed:               true,
		registryNotificationChannel: nil,
		ctx:                         ctx,
		ctxCancel:                   ctxCancel,
		output: NewProcessingOutput(
			blockData.GetSlug(),
			false,
			bytes.NewBuffer([]byte{}),
			nil,
		),
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
	p.setChannelClosed()
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

	return p.blockData
}

func (p *Processing) GetOutput() interfaces.ProcessingOutput {
	p.Lock()
	defer p.Unlock()

	return p.output
}

func (p *Processing) SetRegistryNotificationChannel(channel chan interfaces.Processing) {
	p.Lock()
	defer p.Unlock()

	p.channelClosed = false
	p.registryNotificationChannel = channel
}

func (p *Processing) Start() interfaces.ProcessingOutput {
	logger := config.GetLogger()

	processingOutput := p.GetOutput()

	// Check initial status and fail if it's not pending
	if p.GetStatus() != interfaces.ProcessingStatusPending {
		p.sendResult(false)
		processingOutput.SetError(
			fmt.Errorf(
				"processing with id %s is not in pending state",
				p.GetId().String(),
			),
		)

		return processingOutput
	}
	p.SetStatus(interfaces.ProcessingStatusRunning)

	retryCount := p.processor.GetRetryCount(p.block)
	retryInterval := p.processor.GetRetryInterval(p.block)

	var output *bytes.Buffer
	var stop, retry bool
	var err error

	// Retry loop
	for attempt := 0; attempt <= retryCount; attempt++ {
		output, stop, retry, err = p.block.Process(p.ctx, p.processor, p.blockData)
		processingOutput.SetValue(output)
		processingOutput.SetError(err)
		processingOutput.SetRetry(retry)
		processingOutput.SetRetryAttempt(attempt)

		if err == nil && !retry {
			break
		}

		if err == context.Canceled {
			p.SetError(err)
			p.SetStatus(interfaces.ProcessingStatusFailed)
			p.sendResult(true)

			processingOutput.SetError(
				fmt.Errorf(
					"processing with id %s was cancelled",
					p.GetId().String(),
				),
			)
			return processingOutput
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

			processingOutput.SetError(
				fmt.Errorf(
					"processing with id %s failed after exhausting all %d retry attempts",
					p.GetId().String(),
					retryCount,
				),
			)
			return processingOutput
		}

		// If unrecoverable error, mark as failed
		if err != nil {
			retryMsg := ""
			if attempt > 0 {
				retryMsg = fmt.Sprintf(" after %d attempt(s)", attempt+1)
			}
			_err := fmt.Errorf("processing with id %s failed%s: %w", p.GetId().String(), retryMsg, err)
			processingOutput.SetError(_err)

			p.SetStatus(interfaces.ProcessingStatusFailed)
			p.sendResult(false)
			return processingOutput
		}

	}

	// Processing completed without errors or retries
	p.Lock()

	processingOutput.SetStop(stop)
	p.status = interfaces.ProcessingStatusCompleted
	if stop {
		p.status = interfaces.ProcessingStatusStopped
	}

	p.Unlock()

	p.sendResult(false)

	return processingOutput
}

func (p *Processing) Stop(status interfaces.ProcessingStatus, err error) {
	p.SetStatus(status)
	p.SetError(err)
	p.sendResult(false)
}

func (p *Processing) SetError(err error) {
	p.Lock()
	defer p.Unlock()

	p.output.SetError(err)
}

func (p *Processing) GetError() error {
	p.Lock()
	defer p.Unlock()

	return p.output.GetError()
}

func (p *Processing) setChannelClosed() {
	p.Lock()
	defer p.Unlock()

	p.channelClosed = true
}

func (p *Processing) isChannelClosed() bool {
	p.Lock()
	defer p.Unlock()
	return p.channelClosed
}

func (p *Processing) sendResult(shutdown bool) {
	logger := config.GetLogger()

	p.closeOnce.Do(func() {
		if shutdown || p.isChannelClosed() {
			return
		}

		if p.registryNotificationChannel != nil {
			// Attempt to send without blocking
			select {
			case p.registryNotificationChannel <- p:
				// Sent successfully
			default:
				logger.Warn("Notification channel is full, dropping processing notification")
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
	sync.Mutex

	blockSlug    string
	stop         bool
	retry        bool
	retryAttempt int
	data         *bytes.Buffer
	err          error
}

func NewProcessingOutput(blockSlug string, stop bool, data *bytes.Buffer, err error) *ProcessingOutput {
	return &ProcessingOutput{
		blockSlug: blockSlug,
		stop:      stop,
		data:      data,
		err:       err,
	}
}

func (po *ProcessingOutput) GetId() string {
	po.Lock()
	defer po.Unlock()

	return po.blockSlug
}

func (po *ProcessingOutput) GetStop() bool {
	po.Lock()
	defer po.Unlock()

	return po.stop
}

func (po *ProcessingOutput) GetValue() *bytes.Buffer {
	po.Lock()
	defer po.Unlock()

	return po.data
}

func (po *ProcessingOutput) GetError() error {
	po.Lock()
	defer po.Unlock()

	return po.err
}

func (po *ProcessingOutput) GetRetry() bool {
	po.Lock()
	defer po.Unlock()

	return po.retry
}

func (po *ProcessingOutput) GetRetryAttempt() int {
	po.Lock()
	defer po.Unlock()

	return po.retryAttempt
}

func (po *ProcessingOutput) SetId(blockSlug string) {
	po.Lock()
	defer po.Unlock()

	po.blockSlug = blockSlug
}

func (po *ProcessingOutput) SetStop(stop bool) {
	po.Lock()
	defer po.Unlock()

	po.stop = stop
}

func (po *ProcessingOutput) SetValue(data *bytes.Buffer) {
	po.Lock()
	defer po.Unlock()

	po.data = data
}

func (po *ProcessingOutput) SetError(err error) {
	po.Lock()
	defer po.Unlock()

	po.err = err
}

func (po *ProcessingOutput) SetRetry(retry bool) {
	po.Lock()
	defer po.Unlock()

	po.retry = retry
}

func (po *ProcessingOutput) SetRetryAttempt(retryAttempt int) {
	po.Lock()
	defer po.Unlock()

	po.retryAttempt = retryAttempt
}
