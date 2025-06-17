package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"reflect" // DeepEqualのため

	"firebase.google.com/go/v4/messaging"
	"github.com/teamzidi/example-go-fcm/fcm"
	"github.com/teamzidi/example-go-fcm/store"
)

func TestPushTopicHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name                string
		requestBody         PushTopicRequest
		setupMock           func(mockFCM *fcm.FCMClient)
		expectedStatus      int
		// expectedFCMCallCount int // 検証は setupMock 内で行うか、より詳細なフィールドを mockFCM に持たせる
	}{
		{
			name: "Successful push to topic",
			requestBody: PushTopicRequest{
				Title: "Topic Test Title",
				Body:  "Topic Test Body",
				Topic: "test_topic",
				CustomData: map[string]string{"key": "value"},
			},
			setupMock: func(mockFCM *fcm.FCMClient) {
				mockFCM.MockSend = func(ctx context.Context, msg *messaging.Message) (string, error) {
					if msg.Topic != "test_topic" {
						t.Errorf("Mock: Topic mismatch. Got %s, Want %s", msg.Topic, "test_topic")
					}
					if msg.Notification.Title != "Topic Test Title" {
						t.Errorf("Mock: Title mismatch. Got %s, Want %s", msg.Notification.Title, "Topic Test Title")
					}
					if !reflect.DeepEqual(msg.Data, map[string]string{"key": "value"}) {
						t.Errorf("Mock: Data mismatch. Got %v, Want %v", msg.Data, map[string]string{"key": "value"})
					}
					return "mock_message_id_topic", nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "FCM client returns error on topic push",
			requestBody: PushTopicRequest{
				Title: "Topic Error Title",
				Body:  "Topic Error Body",
				Topic: "error_topic",
			},
			setupMock: func(mockFCM *fcm.FCMClient) {
				mockFCM.MockSend = func(ctx context.Context, msg *messaging.Message) (string, error) {
					return "", errors.New("FCM send to topic failed")
				}
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:        "Missing title",
			requestBody: PushTopicRequest{Body: "Body only", Topic: "t"},
			setupMock:   func(mockFCM *fcm.FCMClient) { /* No FCM call */ },
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Missing body",
			requestBody: PushTopicRequest{Title: "Title only", Topic: "t"},
			setupMock:   func(mockFCM *fcm.FCMClient) { /* No FCM call */ },
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Missing topic",
			requestBody: PushTopicRequest{Title: "Title", Body: "Body"},
			setupMock:   func(mockFCM *fcm.FCMClient) { /* No FCM call */ },
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deviceStore := store.NewDeviceStore()
			mockFCMClient, err := fcm.NewFCMClient(context.Background())
			if err != nil {
				t.Fatalf("Failed to create mock FCMClient: %v", err)
			}

			originalMockSend := mockFCMClient.MockSend // 元のデフォルトモックを保持
			tt.setupMock(mockFCMClient)

			handler := NewPushTopicHandler(mockFCMClient, deviceStore)

			reqBodyBytes, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/pubsub/push/topic", bytes.NewBuffer(reqBodyBytes))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Status = %v, want %v. Body: %s", rr.Code, tt.expectedStatus, rr.Body.String())
			}

			mockFCMClient.MockSend = originalMockSend // モック関数を元に戻す
		})
	}
}


// Test for invalid HTTP method (内容は変更なし)
func TestPushTopicHandler_ServeHTTP_InvalidMethod(t *testing.T) {
	deviceStore := store.NewDeviceStore()
	mockFCMClient, _ := fcm.NewFCMClient(context.Background())
	handler := NewPushTopicHandler(mockFCMClient, deviceStore)

	req := httptest.NewRequest(http.MethodGet, "/pubsub/push/topic", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %v for GET, got %v", http.StatusMethodNotAllowed, rr.Code)
	}
}

// Test for invalid JSON body (内容は変更なし)
func TestPushTopicHandler_ServeHTTP_InvalidJSON(t *testing.T) {
	deviceStore := store.NewDeviceStore()
	mockFCMClient, _ := fcm.NewFCMClient(context.Background())
	handler := NewPushTopicHandler(mockFCMClient, deviceStore)

	req := httptest.NewRequest(http.MethodPost, "/pubsub/push/topic", strings.NewReader("this is not json"))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %v for invalid JSON, got %v", http.StatusBadRequest, rr.Code)
	}
}
