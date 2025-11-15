package auth

import (
	"crypto/md5"
	"fmt"
	"strconv"
	"strings"
)

// HashPassword performs double MD5 hashing as required by the API
// First: MD5(password) -> uppercase hex
// Second: MD5(hash1 + timestamp) -> uppercase hex
func HashPassword(password string, timestamp uint64) string {
	// First MD5 hash
	hash1 := md5.Sum([]byte(password))
	hash1Str := strings.ToUpper(fmt.Sprintf("%X", hash1))

	// Second MD5 hash with timestamp
	combined := hash1Str + strconv.FormatUint(timestamp, 10)
	hash2 := md5.Sum([]byte(combined))
	return strings.ToUpper(fmt.Sprintf("%X", hash2))
}
