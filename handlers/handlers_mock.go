//go:build mock

package handlers

import (
	"context"
	"strings"
)

type fcmClient interface {
	SendToToken(ctx context.Context, token, title, body string, customData map[string]string) (string, error)
	SendToTopic(ctx context.Context, topic, title, body string, customData map[string]string) (string, error)
}

func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()
	return strings.Contains(msg, "retryable")
}
