package helpers

import (
	"crypto/md5"
	"fmt"
)

func CreateCallbackData(action string, firstUUID string, secondString string) string {
	// Construct the initial callback data
	// Make MD5 from secondString
	hash := md5.New()
	hash.Write([]byte(secondString))

	callbackData := fmt.Sprintf("%s:%s:%x", action, firstUUID, hash.Sum(nil))

	// Ensure the callback data is exactly 64 bytes
	if len(callbackData) < 64 {
		// Pad with spaces if it's less than 64 bytes
		callbackData = fmt.Sprintf("%-64s", callbackData)
	} else if len(callbackData) > 64 {
		// If it exceeds 64 bytes, truncate it (handle carefully)
		callbackData = callbackData[:64]
	}

	return callbackData
}
