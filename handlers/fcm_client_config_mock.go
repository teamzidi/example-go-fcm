//go:build test_fcm_mock

package handlers

import (
	"context"
	"errors"
	"log"
)

// fcmHandlerClient は、このハンドラパッケージ内でFCMクライアントを参照するための型です。
// test_fcm_mock ビルド時は、このモック実装が使用されます。
type fcmHandlerClient struct {
	// モックの挙動を制御するための関数フィールド
	MockSendToToken func(ctx context.Context, token string, title string, body string, customData map[string]string) (string, error)
	MockSendToTopic func(ctx context.Context, topic string, title string, body string, customData map[string]string) (string, error)
}

// NewFcmHandlerClient は、FCMクライアントのインスタンスを生成します。
// test_fcm_mock ビルド時は、モック版のFCMクライアントを生成します。
func NewFcmHandlerClient(ctx context.Context) (*fcmHandlerClient, error) {
	log.Println("Using Mock FCMClient via handlers/fcm_client_config_mock.go (test_fcm_mock build tag is active)")
	return &fcmHandlerClient{
		MockSendToToken: func(ctx context.Context, token string, title string, body string, customData map[string]string) (string, error) {
			log.Printf("Mock fcmHandlerClient.SendToToken called with: Token=%s, Title=%s, Body=%s, Data=%v\n", token, title, body, customData)
			if token == "" {
				return "", errors.New("mock: token cannot be empty")
			}
			return "mock-message-id-for-" + token, nil
		},
		MockSendToTopic: func(ctx context.Context, topic string, title string, body string, customData map[string]string) (string, error) {
			log.Printf("Mock fcmHandlerClient.SendToTopic called with: Topic=%s, Title=%s, Body=%s, Data=%v\n", topic, title, body, customData)
			if topic == "" {
				return "", errors.New("mock: topic cannot be empty")
			}
			return "mock-message-id-for-topic-" + topic, nil
		},
	}, nil
}

// SendToToken はモック版のSendToTokenメソッドです。
// fcm.FCMClient (本物) と同じメソッドシグネチャを持ちます。
func (m *fcmHandlerClient) SendToToken(ctx context.Context, token string, title string, body string, customData map[string]string) (string, error) {
	if m.MockSendToToken != nil {
		return m.MockSendToToken(ctx, token, title, body, customData)
	}
	return "", errors.New("MockSendToToken function not configured in mock fcmHandlerClient")
}

// SendToTopic はモック版のSendToTopicメソッドです。
// fcm.FCMClient (本物) と同じメソッドシグネチャを持ちます。
func (m *fcmHandlerClient) SendToTopic(ctx context.Context, topic string, title string, body string, customData map[string]string) (string, error) {
	if m.MockSendToTopic != nil {
		return m.MockSendToTopic(ctx, topic, title, body, customData)
	}
	return "", errors.New("MockSendToTopic function not configured in mock fcmHandlerClient")
}
