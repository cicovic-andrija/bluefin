package utils

import (
	"strconv"
	"strings"
)

func ConvertAndCheckID(strid string, max int) int {
	id, err := strconv.Atoi(strid)
	if err != nil || id < 1 || id > max {
		return 0
	}
	return id
}

func IsSpecialTag(tag string) bool {
	return strings.HasPrefix(tag, "_")
}
