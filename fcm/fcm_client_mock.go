//go:build test_fcm_mock

package fcm

import (
	"context"
	"errors" // エラー作成のため
	"log"    // ログ出力のため

	"firebase.google.com/go/v4/messaging"
)

// FCMClient はFirebase Cloud Messagingクライアントのモック実装です。
// テスト時に使用されます。
type FCMClient struct {
	// モックの挙動を制御するための関数フィールド
	MockSend                 func(ctx context.Context, message *messaging.Message) (string, error)
	MockSendToToken          func(ctx context.Context, token string, title string, body string) (string, error)
	MockSendToMultipleTokens func(ctx context.Context, tokens []string, title string, body string) (*messaging.BatchResponse, error)

	// 呼び出し記録など、テストで検証したい情報を保持するフィールド (オプション)
	// 例: SendCalledWith *messaging.Message
	// 例: SendToMultipleTokensCalledWithTokens []string
}

// NewFCMClient はモック版FCMClientの新しいインスタンスを作成します。
// テスト時には、この関数が呼び出され、モッククライアントが返されます。
func NewFCMClient(ctx context.Context) (*FCMClient, error) {
	log.Println("Using Mock FCMClient (test_fcm_mock build tag is active)")
	// モッククライアントは通常、エラーなく初期化される想定
	return &FCMClient{
		// 必要に応じてデフォルトのモック関数をここで設定することもできる
		MockSend: func(ctx context.Context, message *messaging.Message) (string, error) {
			log.Printf("MockFCMClient.Send called with: Topic=%s, Token=%s\n", message.Topic, message.Token)
			return "mock-message-id", nil // デフォルトの成功レスポンス
		},
		MockSendToToken: func(ctx context.Context, token string, title string, body string) (string, error) {
			log.Printf("MockFCMClient.SendToToken called with: Token=%s, Title=%s\n", token, title)
			return "mock-message-id-single", nil
		},
		MockSendToMultipleTokens: func(ctx context.Context, tokens []string, title string, body string) (*messaging.BatchResponse, error) {
			log.Printf("MockFCMClient.SendToMultipleTokens called with: Tokens=%v, Title=%s\n", tokens, title)
			return &messaging.BatchResponse{SuccessCount: len(tokens), FailureCount: 0}, nil
		},
	}, nil
}

// Send はモック版のSendメソッドです。
func (c *FCMClient) Send(ctx context.Context, message *messaging.Message) (string, error) {
	if c.MockSend != nil {
		return c.MockSend(ctx, message)
	}
	// デフォルトの挙動やエラーを返すなど
	return "", errors.New("MockSend function not configured")
}

// SendToToken はモック版のSendToTokenメソッドです。
func (c *FCMClient) SendToToken(ctx context.Context, token string, title string, body string) (string, error) {
	if c.MockSendToToken != nil {
		return c.MockSendToToken(ctx, token, title, body)
	}
	return "", errors.New("MockSendToToken function not configured")
}

// SendToMultipleTokens はモック版のSendToMultipleTokensメソッドです。
func (c *FCMClient) SendToMultipleTokens(ctx context.Context, tokens []string, title string, body string) (*messaging.BatchResponse, error) {
	if c.MockSendToMultipleTokens != nil {
		return c.MockSendToMultipleTokens(ctx, tokens, title, body)
	}
	return nil, errors.New("MockSendToMultipleTokens function not configured")
}
