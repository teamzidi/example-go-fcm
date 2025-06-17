package fcm

import (
	"context"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
)

// FCMClientInterface はFCMクライアントの操作を定義するインターフェースです。
// モックテストのために使用します。
type FCMClientInterface interface {
	SendToToken(ctx context.Context, token string, title string, body string) (string, error)
	SendToMultipleTokens(ctx context.Context, tokens []string, title string, body string) (*messaging.BatchResponse, error)
}

// FCMClient はFirebase Cloud Messagingのクライアントです。(既存の構造体)
// この構造体が FCMClientInterface を実装することを確認します。
var _ FCMClientInterface = (*FCMClient)(nil)

// FCMClient はFirebase Cloud Messagingのクライアントです。
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

// SendToToken は指定されたデバイストークンに通知メッセージを送信します。
func (c *FCMClient) SendToToken(ctx context.Context, token string, title string, body string) (string, error) {
	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Token: token,
	}

	response, err := c.msg.Send(ctx, message)
	if err != nil {
		log.Printf("Error sending message to token %s: %v\n", token, err)
		return "", err
	}
	log.Printf("Successfully sent message to token %s: %s\n", token, response)
	return response, nil
}

// SendToMultipleTokens は複数のデバイストークンに同じ通知メッセージを送信します。
// FCMは SendMulticast という一括送信APIを提供しており、それを利用します。
func (c *FCMClient) SendToMultipleTokens(ctx context.Context, tokens []string, title string, body string) (*messaging.BatchResponse, error) {
	if len(tokens) == 0 {
		log.Println("No tokens to send messages to.")
		return &messaging.BatchResponse{}, nil // 空のレスポンスを返すか、エラーを返すかは要件による
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
		for i, resp := range br.Responses {
			if !resp.Success {
				log.Printf("Failed to send message to token %s: %s\n", tokens[i], resp.Error)
			}
		}
	}
	return br, nil
}
