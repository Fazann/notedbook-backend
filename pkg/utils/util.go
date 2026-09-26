package utils

import (
	"slices"
)

func IsInclude[T comparable](arr []T, val T) bool {
	return slices.Contains(arr, val)
}

func GetTempFilePath() string {
	return "./tmp/file/"
}
