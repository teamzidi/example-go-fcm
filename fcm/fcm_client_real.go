//go:build !test_fcm_mock

package fcm

import (
	"context"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
)

// FCMClient はFirebase Cloud Messagingのクライアントです。
// これは本番用の実装です。
type FCMClient struct {
	app *firebase.App
	msg *messaging.Client
}

// NewFCMClient は新しいFCMClientのインスタンスを作成します。
// 環境変数 GOOGLE_APPLICATION_CREDENTIALS が設定されている必要があります。
func NewFCMClient(ctx context.Context) (*FCMClient, error) {
	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		log.Printf("Error initializing Firebase app: %v\n", err)
		return nil, err
	}

	msgClient, err := app.Messaging(ctx)
	if err != nil {
		log.Printf("Error getting Messaging client: %v\n", err)
		return nil, err
	}

	return &FCMClient{
		app: app,
		msg: msgClient,
	}, nil
}

// Send は指定された messaging.Message を送信します。
func (c *FCMClient) Send(ctx context.Context, message *messaging.Message) (string, error) {
	response, err := c.msg.Send(ctx, message)
	if err != nil {
		log.Printf("Error sending message via FCM Send: %v\n", err)
		return "", err
	}
	log.Printf("Successfully sent message via FCM Send: %s\n", response)
	return response, nil
}

// SendToToken は指定されたデバイストークンに通知メッセージを送信します。
func (c *FCMClient) SendToToken(ctx context.Context, token string, title string, body string) (string, error) {
	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Token: token,
	}
	return c.Send(ctx, message) // 汎用 Send を使用
}

// SendToMultipleTokens は複数のデバイストークンに同じ通知メッセージを送信します。
func (c *FCMClient) SendToMultipleTokens(ctx context.Context, tokens []string, title string, body string) (*messaging.BatchResponse, error) {
	if len(tokens) == 0 {
		log.Println("No tokens to send messages to in SendToMultipleTokens.")
		return &messaging.BatchResponse{}, nil
	}

	message := &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Tokens: tokens,
	}

	br, err := c.msg.SendMulticast(ctx, message)
	if err != nil {
		log.Printf("Error sending multicast message: %v\n", err)
		return nil, err
	}

	log.Printf("Successfully sent multicast message. SuccessCount: %d, FailureCount: %d\n", br.SuccessCount, br.FailureCount)
	if br.FailureCount > 0 {
		log.Printf("Some messages failed in multicast send. Check BatchResponse.Responses for details.")
	}
	return br, nil
}
