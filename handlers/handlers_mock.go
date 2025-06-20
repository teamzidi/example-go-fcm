//go:build mock

package handlers

import "context"

type fcmClient interface {
	SendToToken(ctx context.Context, token, title, body string, customData map[string]string) (string, error)
	SendToTopic(ctx context.Context, topic, title, body string, customData map[string]string) (string, error)
}
