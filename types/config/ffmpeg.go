package config

import (
	"embed"
	"fmt"
	"os"
	"runtime"
	"sync"
)

//go:embed assets/ffmpeg/ffmpeg-darwin assets/ffmpeg/ffmpeg-linux
var ffmpegFiles embed.FS

var (
	ffmpegPath    string
	ffmpegInitErr error
	ffmpegOnce    sync.Once
)

func GetFFmpegBinary() (string, error) {
	ffmpegOnce.Do(func() {
		var ffmpegFile string

		switch runtime.GOOS {
		case "darwin":
			ffmpegFile = "assets/ffmpeg/ffmpeg-darwin"
		case "linux":
			ffmpegFile = "assets/ffmpeg/ffmpeg-linux"
		// case "windows":
		// 	ffmpegFile = "bin/ffmpeg-windows.exe"
		default:
			ffmpegInitErr = fmt.Errorf("unsupported platform: %s", runtime.GOOS)
			return
		}

		// Extract FFmpeg binary to a temporary file
		tempFile, err := os.CreateTemp("", "ffmpeg-*")
		if err != nil {
			ffmpegInitErr = err
			return
		}
		defer tempFile.Close()

		ffmpegData, err := ffmpegFiles.ReadFile(ffmpegFile)
		if err != nil {
			ffmpegInitErr = err
			return
		}

		_, err = tempFile.Write(ffmpegData)
		if err != nil {
			ffmpegInitErr = err
			return
		}

		// Make the temporary file executable
		err = os.Chmod(tempFile.Name(), 0755)
		if err != nil {
			ffmpegInitErr = err
			return
		}

		ffmpegPath = tempFile.Name()
	})

	return ffmpegPath, ffmpegInitErr
}
