package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

var (
	logger     echo.Logger
	onceLogger sync.Once

	entityLoggerStore EntityLoggerStore = EntityLoggerStore{
		loggers: make(map[string]*LoggerWrapper),
	}
)

type EntityLoggerStore struct {
	sync.RWMutex
	loggers map[string]*LoggerWrapper
}

// SafeBuffer is a thread-safe wrapper around bytes.Buffer
type SafeBuffer struct {
	sync.RWMutex
	buffer bytes.Buffer
}

func (sb *SafeBuffer) Write(p []byte) (n int, err error) {
	sb.Lock()
	defer sb.Unlock()
	return sb.buffer.Write(p)
}

func (sb *SafeBuffer) Bytes() []byte {
	sb.RLock()
	defer sb.RUnlock()
	return sb.buffer.Bytes()
}

func (sb *SafeBuffer) String() string {
	sb.RLock()
	defer sb.RUnlock()
	return sb.buffer.String()
}

func (sb *SafeBuffer) Reset() {
	sb.Lock()
	defer sb.Unlock()
	sb.buffer.Reset()
}

// LoggerWrapper holds multiple writers for easier extraction
type LoggerWrapper struct {
	sync.Mutex
	writers []io.Writer
}

func (lw *LoggerWrapper) Write(p []byte) (n int, err error) {
	lw.Lock()
	defer lw.Unlock()

	for _, writer := range lw.writers {
		n, err = writer.Write(p)
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (lw *LoggerWrapper) AddWriter(writer io.Writer) {
	lw.Lock()
	defer lw.Unlock()

	lw.writers = append(lw.writers, writer)
}

func (lw *LoggerWrapper) GetBuffer() *SafeBuffer {
	lw.Lock()
	defer lw.Unlock()

	for _, writer := range lw.writers {
		if buf, ok := writer.(*syncWriter); ok {
			if buffer, ok := buf.writer.(*SafeBuffer); ok {
				return buffer
			}
		}
	}
	return nil
}

type syncWriter struct {
	sync.Mutex
	writer io.Writer // The underlying writer (e.g., bytes.Buffer, file, etc.)
}

func (s *syncWriter) Write(p []byte) (n int, err error) {
	s.Lock()
	defer s.Unlock()

	return s.writer.Write(p)
}

func GetLogger() echo.Logger {
	onceLogger.Do(func() {
		config := GetConfig()
		e := echo.New()
		e.Logger.SetLevel(log.WARN)

		switch strings.ToUpper(config.Log.Level) {
		case "DEBUG":
			e.Logger.SetLevel(log.DEBUG)
		case "INFO":
			e.Logger.SetLevel(log.INFO)
		case "WARN":
			e.Logger.SetLevel(log.WARN)
		case "ERROR":
			e.Logger.SetLevel(log.ERROR)
		case "OFF":
			e.Logger.SetLevel(log.OFF)
		}

		logger = e.Logger
	})
	return logger
}

// TODO: Cleanup that mapping since it might take some memory
func GetLoggerForEntity(entityType string, entityId interface{}) (echo.Logger, *SafeBuffer) {
	logLevel := log.DEBUG
	entityKey := fmt.Sprintf("%s:%s", entityType, entityId)

	entityLoggerStore.RLock()
	existingLogger, found := entityLoggerStore.loggers[entityKey]
	entityLoggerStore.RUnlock()

	if found {
		e := echo.New()
		e.Logger.SetLevel(logLevel)
		e.Logger.SetOutput(existingLogger)
		e.Logger.SetPrefix(entityKey)

		return e.Logger, existingLogger.GetBuffer()
	}

	//&bytes.Buffer{}
	newBuffer := &SafeBuffer{
		buffer: bytes.Buffer{},
	}
	syncBuffer := &syncWriter{
		writer: newBuffer,
	}
	syncStdout := &syncWriter{
		writer: os.Stdout,
	}
	logWriter := &LoggerWrapper{
		writers: []io.Writer{syncBuffer, syncStdout},
	}

	entityLoggerStore.Lock()
	entityLoggerStore.loggers[entityKey] = logWriter
	entityLoggerStore.Unlock()

	e := echo.New()
	e.Logger.SetLevel(logLevel)
	e.Logger.SetOutput(logWriter)
	e.Logger.SetPrefix(entityKey)

	return e.Logger, newBuffer
}
