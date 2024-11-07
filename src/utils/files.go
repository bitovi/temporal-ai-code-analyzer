package utils

import (
	"strings"
)

func IsHiddenFile(path string) bool {
	parts := strings.Split(path, "/")

	for _, part := range parts {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}

func IsConfigFile(path string) bool {
	if strings.HasSuffix(path, ".json") || strings.HasSuffix(path, "yaml") || strings.HasSuffix(path, "yml") || strings.HasSuffix(path, "toml") {
		return true
	}
	return false
}
