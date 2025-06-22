package fcm

import (
	"context"
	"errors"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
)

func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	return messaging.IsInternal(err) || messaging.IsUnavailable(err) || messaging.IsQuotaExceeded(err)
}

// Client はFirebase Cloud Messagingのクライアントです。(旧 FCMClient)
type Client struct {
	msg *messaging.Client
}

// NewClient は新しいClientのインスタンスを作成します。(旧 NewFCMClient)
// 環境変数 GOOGLE_APPLICATION_CREDENTIALS が設定されている必要があります。
func NewClient(ctx context.Context) (*Client, error) {
	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		return nil, err
	}

	msgClient, err := app.Messaging(ctx)
	if err != nil {
		return nil, err
	}

	return &Client{msg: msgClient}, nil
}

// SendToToken は指定された単一のデバイストークンに通知とデータペイロードを送信します。
func (c *Client) SendToToken(ctx context.Context, token string, title string, body string, customData map[string]string) (string, error) {
	if token == "" {
		return "", errors.New("token cannot be empty")
	}

	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data:  customData,
		Token: token,
	}

	response, err := c.msg.Send(ctx, message)
	if err != nil {
		return "", fmt.Errorf("sending message to token %s: %w", token, err)
	}

	return response, nil
}

// SendToTopic は指定されたFCMトピックに通知とデータペイロードを送信します。
func (c *Client) SendToTopic(ctx context.Context, topic string, title string, body string, customData map[string]string) (string, error) {
	if topic == "" {
		return "", fmt.Errorf("topic cannot be empty")
	}

	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data:  customData,
		Topic: topic,
	}

	response, err := c.msg.Send(ctx, message)
	if err != nil {
		return "", fmt.Errorf("sending message to topic %s: %w", topic, err)
	}

	return response, nil
}
