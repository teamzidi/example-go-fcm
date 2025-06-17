package fcm

import (
	"context"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
)

// FCMClientInterface はFCMクライアントの操作を定義するインターフェースです。
type FCMClientInterface interface {
	Send(ctx context.Context, message *messaging.Message) (string, error) // 新しい汎用Sendメソッド
	SendToToken(ctx context.Context, token string, title string, body string) (string, error)
	SendToMultipleTokens(ctx context.Context, tokens []string, title string, body string) (*messaging.BatchResponse, error)
}

// FCMClient はFirebase Cloud Messagingのクライアントです。
type FCMClient struct {
	app *firebase.App
	msg *messaging.Client
}

// FCMClient が FCMClientInterface を実装していることをコンパイル時に確認
var _ FCMClientInterface = (*FCMClient)(nil)

// NewFCMClient は新しいFCMClientのインスタンスを作成します。
// 環境変数 GOOGLE_APPLICATION_CREDENTIALS が設定されている必要があります。
func NewFCMClient(ctx context.Context) (*FCMClient, error) {
	app, err := firebase.NewApp(ctx, nil) // オプションなしで初期化
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
// message.Token, message.Topic, message.Condition のいずれかが設定されている必要があります。
func (c *FCMClient) Send(ctx context.Context, message *messaging.Message) (string, error) {
	response, err := c.msg.Send(ctx, message)
	if err != nil {
		// エラーログは呼び出し側で出すことも検討 (ここではFCMクライアントの責務としてログを出す)
		log.Printf("Error sending message via FCM Send: %v\n", err)
		return "", err
	}
	log.Printf("Successfully sent message via FCM Send: %s\n", response)
	return response, nil
}

// SendToToken は指定されたデバイストークンに通知メッセージを送信します。
// (このメソッドは、より汎用的な Send メソッドのラッパーとしても実装可能だが、今回は既存のまま)
func (c *FCMClient) SendToToken(ctx context.Context, token string, title string, body string) (string, error) {
	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Token: token,
	}

	response, err := c.msg.Send(ctx, message) // 内部的には汎用Sendと同じ
	if err != nil {
		log.Printf("Error sending message to token %s: %v\n", token, err)
		return "", err
	}
	log.Printf("Successfully sent message to token %s: %s\n", token, response)
	return response, nil
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
		// 個々のエラーのログ出力は呼び出し側で行うか、ここで詳細に行うか検討。
		// ここでは基本的なログのみ。
		log.Printf("Some messages failed in multicast send. Check BatchResponse.Responses for details.")
	}
	return br, nil
}
