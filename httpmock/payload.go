package httpmock

import (
	"strings"
)

const (
	CHANNEL_CLEAR     = "clear"
	CHANNEL_MOCK      = "mock"
	PAYLOAD_SEPARATOR = "|"
)

func createMockPayload(mockString []string) string {
	return CHANNEL_MOCK + SEPARATOR + strings.Join(mockString, SEPARATOR)
}

func createClearPayload() string {
	return CHANNEL_CLEAR + SEPARATOR
}
