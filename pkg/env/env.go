package env

import (
	"os"
	"strings"
)

// IsLocal 环境监测
func IsLocal() bool {
	host, _ := os.Hostname()
	if host == "DESKTOP-2VT9AH2" ||
		strings.Contains(host, ".local") ||
		strings.Contains(host, "vanilla") ||
		strings.Contains(host, "deMini") ||
		strings.Contains(host, "MacBook") ||
		strings.Contains(host, "DESKTOP-") ||
		strings.Contains(host, "MBP") {
		return true
	} else {
		return false
	}
}
