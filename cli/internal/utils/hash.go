package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

// calculateSHA256 calculates the SHA256 hash of the given data and returns it as a hex string
// If data is nil, it returns the hash of an empty string (required for AWS Signature V4).
func CalculateSHA256(data []byte) string {
	if data == nil {
		data = []byte{}
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
