package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

var (
	mutex   sync.Mutex
	workers []string
)

func init() {
	go populateWorkers()
}

func populateWorkers() {
	for {
		mutex.Lock()
		workers = append(workers, time.Now().Format(time.RFC3339))
		mutex.Unlock()
		time.Sleep(1 * time.Second)
	}
}

func WorkersDiscoverHandler(c echo.Context) error {
	mutex.Lock()
	defer mutex.Unlock()
	return c.JSON(http.StatusOK, workers)
}
