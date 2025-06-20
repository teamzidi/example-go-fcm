package handlers

import (
	"context"
	"errors"
	"log"
)

func (h *PushDeviceHandler) WithMock(mock any) *PushDeviceHandler {
	c, ok := mock.(fcmClient)
	if !ok {
		panic("mock must implement fcmClient interface")
	}

	h.fcmClient = c

	return h
}

func (h *PushTopicHandler) WithMock(mock any) *PushTopicHandler {
	c, ok := mock.(fcmClient)
	if !ok {
		panic("mock must implement fcmClient interface")
	}

	h.fcmClient = c

	return h
}

type MockFCMClient struct {
	MockSendToToken func(ctx context.Context, token, title, body string, customData map[string]string) (string, error)
	MockSendToTopic func(ctx context.Context, topic, title, body string, customData map[string]string) (string, error)
}

func (m *MockFCMClient) SendToToken(ctx context.Context, token string, title string, body string, customData map[string]string) (string, error) {
	if m.MockSendToToken != nil {
		return m.MockSendToToken(ctx, token, title, body, customData)
	}

	log.Printf("Mock fcmHandlerClient.SendToToken called with: Token=%s, Title=%s, Body=%s, Data=%v\n", token, title, body, customData)
	if token == "" {
		return "", errors.New("mock: token cannot be empty")
	}

	return "mock-message-id-for-" + token, nil
}

func (m *MockFCMClient) SendToTopic(ctx context.Context, topic string, title string, body string, customData map[string]string) (string, error) {
	if m.MockSendToTopic != nil {
		return m.MockSendToTopic(ctx, topic, title, body, customData)
	}

	log.Printf("Mock fcmHandlerClient.SendToTopic called with: Topic=%s, Title=%s, Body=%s, Data=%v\n", topic, title, body, customData)
	if topic == "" {
		return "", errors.New("mock: topic cannot be empty")
	}
	return "mock-message-id-for-topic-" + topic, nil
}
