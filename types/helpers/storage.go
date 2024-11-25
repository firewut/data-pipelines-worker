package helpers

import (
	"bytes"
	"io"

	"github.com/gabriel-vasile/mimetype"
)

func DetectMimeTypeFromBuffer(largeBuffer bytes.Buffer) (*mimetype.MIME, error) {
	var detectorBytesLen int64 = 261

	// Create a new buffer to hold the copied data
	var copiedBuffer bytes.Buffer

	// Create a TeeReader that reads from largeBuffer and writes to copiedBuffer
	teeReader := io.TeeReader(io.LimitReader(&largeBuffer, detectorBytesLen), &copiedBuffer)

	smallBuffer := make([]byte, detectorBytesLen)
	bytesRead, err := io.ReadFull(teeReader, smallBuffer)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, err
	}

	return mimetype.Detect(smallBuffer[:bytesRead]), nil
}
