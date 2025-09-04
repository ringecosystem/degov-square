package utils

import (
	"fmt"
	"strconv"
	"time"
)

func ParseTimestamp(tsStr string) (time.Time, error) {
	if tsStr == "" {
		return time.Time{}, fmt.Errorf("timestamp string is empty")
	}

	// prefer parsing millisecond unix timestamps first
	if unixMilli, err := strconv.ParseInt(tsStr, 10, 64); err == nil {
		return time.UnixMilli(unixMilli), nil
	}

	return time.Time{}, fmt.Errorf("failed to parse timestamp in any known format: %s", tsStr)
}
