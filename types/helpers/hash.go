package helpers

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashInput hashes the input string using SHA-256
func HashInput(input string) string {
	hash := sha256.New()
	hash.Write([]byte(input))
	hashBytes := hash.Sum(nil)
	return hex.EncodeToString(hashBytes) // Return the hash as a hexadecimal string
}

// VerifyHash checks if the given input matches the provided hash
func VerifyHash(input, receivedHash string) bool {
	computedHash := HashInput(input)
	return computedHash == receivedHash // Return true if they match
}
