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
	mu      sync.RWMutex
	loggers map[string]*LoggerWrapper
}

// LoggerWrapper holds multiple writers for easier extraction
type LoggerWrapper struct {
	writers []io.Writer
}

func (lw *LoggerWrapper) Write(p []byte) (n int, err error) {
	for _, writer := range lw.writers {
		n, err = writer.Write(p)
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (lw *LoggerWrapper) AddWriter(writer io.Writer) {
	lw.writers = append(lw.writers, writer)
}

func (lw *LoggerWrapper) GetBuffer() *bytes.Buffer {
	for _, writer := range lw.writers {
		if buf, ok := writer.(*syncWriter); ok {
			if buffer, ok := buf.writer.(*bytes.Buffer); ok {
				return buffer
			}
		}
	}
	return nil
}

// Custom writer that synchronizes access to an underlying writer
type syncWriter struct {
	mu     sync.Mutex // Mutex to synchronize writes
	writer io.Writer  // The underlying writer (e.g., bytes.Buffer, file, etc.)
}

func (s *syncWriter) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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
func GetLoggerForEntity(entityType string, entityId interface{}) (echo.Logger, *bytes.Buffer) {
	logLevel := log.DEBUG
	entityKey := fmt.Sprintf("%s:%s", entityType, entityId)

	entityLoggerStore.mu.RLock()
	existingLogger, found := entityLoggerStore.loggers[entityKey]
	entityLoggerStore.mu.RUnlock()

	if found {
		e := echo.New()
		e.Logger.SetLevel(logLevel)
		e.Logger.SetOutput(existingLogger)
		e.Logger.SetPrefix(entityKey)

		return e.Logger, existingLogger.GetBuffer()
	}

	newBuffer := &bytes.Buffer{}
	syncBuffer := &syncWriter{
		writer: newBuffer,
	}
	syncStdout := &syncWriter{
		writer: os.Stdout,
	}
	logWriter := &LoggerWrapper{
		writers: []io.Writer{syncBuffer, syncStdout},
	}

	entityLoggerStore.mu.Lock()
	entityLoggerStore.loggers[entityKey] = logWriter
	entityLoggerStore.mu.Unlock()

	e := echo.New()
	e.Logger.SetLevel(logLevel)
	e.Logger.SetOutput(logWriter)
	e.Logger.SetPrefix(entityKey)

	return e.Logger, newBuffer
}
