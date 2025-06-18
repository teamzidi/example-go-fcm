package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	// "reflect" // DeepEqual を使用しない場合は不要
)

// Helper function to create a PubSubPushRequest for topic push tests
func newTopicPushPubSubRequest(payload TopicPushPayload) PubSubPushRequest {
	payloadBytes, _ := json.Marshal(payload)
	return PubSubPushRequest{
		Message: PubSubInternalMessage{
			Data:        base64.StdEncoding.EncodeToString(payloadBytes),
			MessageID:   "test-message-id-topic",
			PublishTime: "test-publish-time-topic",
		},
		Subscription: "test-subscription-topic",
	}
}

func TestPushTopicHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name                string
		actualPayload       TopicPushPayload // Base64デコード後のペイロード
		emptyData           bool
		invalidBase64       bool
		setupMock           func(mockFCM *fcmHandlerClient)
		expectedStatus      int
	}{
		{
			name: "Successful push to topic with custom data",
			actualPayload: TopicPushPayload{
				Title: "Topic Test Title",
				Body:  "Topic Test Body",
				Topic: "test_topic",
				CustomData: map[string]string{"type": "topic_push", "id": "abc"},
			},
			setupMock: func(mockFCM *fcmHandlerClient) {
				mockFCM.MockSendToTopic = func(ctx context.Context, topic string, title string, body string, customData map[string]string) (string, error) {
					if topic != "test_topic" {
						t.Errorf("Mock: Topic mismatch. Got %s", topic)
					}
					// ... (他のアサーションは省略) ...
					return "mock_message_id_topic_custom", nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "FCM client returns error on topic push",
			actualPayload: TopicPushPayload{
				Title: "Topic Error Title",
				Body:  "Topic Error Body",
				Topic: "error_topic",
			},
			setupMock: func(mockFCM *fcmHandlerClient) {
				mockFCM.MockSendToTopic = func(ctx context.Context, topic string, title string, body string, customData map[string]string) (string, error) {
					return "", errors.New("FCM send to topic failed")
				}
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:        "Missing title in actual payload",
			actualPayload: TopicPushPayload{Body: "Body only", Topic: "t"},
			setupMock:   func(mockFCM *fcmHandlerClient) { /* No FCM call */ },
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Missing topic in actual payload",
			actualPayload: TopicPushPayload{Title: "Title", Body: "Body", Topic: ""},
			setupMock:   func(mockFCM *fcmHandlerClient) { /* No FCM call */ },
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty Message.Data",
			emptyData:      true,
			setupMock:      func(mockFCM *fcmHandlerClient) { /* No FCM call */ },
			expectedStatus: http.StatusOK, // Ackされる
		},
		{
			name:           "Invalid Base64 in Message.Data",
			invalidBase64:  true,
			actualPayload:  TopicPushPayload{Title: "T", Body: "B", Topic: "t"},
			setupMock:      func(mockFCM *fcmHandlerClient) { /* No FCM call */ },
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFCMClient, err := NewFcmHandlerClient(context.Background())
			if err != nil {
				t.Fatalf("Failed to create mock FCMClient: %v", err)
			}

			originalMockSendToTopic := mockFCMClient.MockSendToTopic
			if tt.setupMock != nil {
				tt.setupMock(mockFCMClient)
			}

			handler := NewPushTopicHandler(mockFCMClient)

			var reqBodyBytes []byte
			if tt.emptyData {
				pubSubReq := PubSubPushRequest{Message: PubSubInternalMessage{Data: ""}}
				reqBodyBytes, _ = json.Marshal(pubSubReq)
			} else if tt.invalidBase64 {
				pubSubReq := PubSubPushRequest{Message: PubSubInternalMessage{Data: "not-base64"}}
				reqBodyBytes, _ = json.Marshal(pubSubReq)
			} else {
				pubSubReq := newTopicPushPubSubRequest(tt.actualPayload)
				reqBodyBytes, _ = json.Marshal(pubSubReq)
			}

			req := httptest.NewRequest(http.MethodPost, "/pubsub/push/topic", bytes.NewBuffer(reqBodyBytes))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Status = %v, want %v. Body: %s", rr.Code, tt.expectedStatus, rr.Body.String())
			}

			mockFCMClient.MockSendToTopic = originalMockSendToTopic
		})
	}
}

func TestPushTopicHandler_ServeHTTP_InvalidMethod(t *testing.T) {
	mockFCMClient, _ := NewFcmHandlerClient(context.Background())
	handler := NewPushTopicHandler(mockFCMClient)
	req := httptest.NewRequest(http.MethodGet, "/pubsub/push/topic", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %v for GET, got %v", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestPushTopicHandler_ServeHTTP_InvalidPubSubJSON(t *testing.T) {
	mockFCMClient, _ := NewFcmHandlerClient(context.Background())
	handler := NewPushTopicHandler(mockFCMClient)
	req := httptest.NewRequest(http.MethodPost, "/pubsub/push/topic", strings.NewReader("this is not json"))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %v for invalid Pub/Sub JSON, got %v. Body: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}
