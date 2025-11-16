package auth

import (
	"crypto/md5"
	"fmt"
	"strconv"
	"strings"
)

// FirstHash performs the first MD5 hash on the password
// This hash is stored locally in the config file
func FirstHash(password string) string {
	hash := md5.Sum([]byte(password))
	return strings.ToUpper(fmt.Sprintf("%X", hash))
}

// SecondHash performs the second MD5 hash with timestamp
// Input: first hash (from FirstHash) and current timestamp
// Output: final hash to send to the API
func SecondHash(firstHash string, timestamp uint64) string {
	combined := firstHash + strconv.FormatUint(timestamp, 10)
	hash := md5.Sum([]byte(combined))
	return strings.ToUpper(fmt.Sprintf("%X", hash))
}

// HashPassword performs double MD5 hashing as required by the API
// First: MD5(password) -> uppercase hex
// Second: MD5(hash1 + timestamp) -> uppercase hex
// This is a convenience function that combines FirstHash and SecondHash
func HashPassword(password string, timestamp uint64) string {
	firstHash := FirstHash(password)
	return SecondHash(firstHash, timestamp)
}
