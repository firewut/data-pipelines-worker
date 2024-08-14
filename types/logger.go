package types

import (
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

var (
	logger echo.Logger
	once   sync.Once
)

func GetLogger() echo.Logger {
	once.Do(func() {
		config := NewConfig()
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
