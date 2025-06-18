package handlers // モック定義も同じパッケージなので _test サフィックスなし

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"reflect"

	// "github.com/teamzidi/example-go-fcm/fcm" // fcm パッケージは直接インポートしない
	"github.com/teamzidi/example-go-fcm/store"
	// "firebase.google.com/go/v4/messaging" // messaging.Message はモックの引数型としては不要になった
)

// 注意: このテストファイルが正しく動作するためには、
// go test -tags=test_fcm_mock ./... のようにビルドタグを指定して実行し、
// handlers/fcm_client_config_mock.go がビルドされるようにする必要がある。

func TestPushTopicHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name                string
		requestBody         PushTopicRequest
		setupMock           func(mockFCM *fcmHandlerClient) // 型を *fcmHandlerClient に変更
		expectedStatus      int
	}{
		{
			name: "Successful push to topic with custom data",
			requestBody: PushTopicRequest{
				Title: "Topic Test Title",
				Body:  "Topic Test Body",
				Topic: "test_topic",
				CustomData: map[string]string{"type": "topic_push", "id": "abc"},
			},
			setupMock: func(mockFCM *fcmHandlerClient) { // 型を *fcmHandlerClient に変更
				mockFCM.MockSendToTopic = func(ctx context.Context, topic string, title string, body string, customData map[string]string) (string, error) {
					if topic != "test_topic" {
						t.Errorf("Mock: Topic mismatch. Got %s", topic)
					}
					if title != "Topic Test Title" {
						t.Errorf("Mock: Title mismatch. Got %s", title)
					}
					expectedData := map[string]string{"type": "topic_push", "id": "abc"}
					if !reflect.DeepEqual(customData, expectedData) {
						t.Errorf("Mock: CustomData mismatch. Got %v, want %v", customData, expectedData)
					}
					return "mock_message_id_topic_custom", nil
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
			setupMock: func(mockFCM *fcmHandlerClient) { // 型を *fcmHandlerClient に変更
				mockFCM.MockSendToTopic = func(ctx context.Context, topic string, title string, body string, customData map[string]string) (string, error) {
					return "", errors.New("FCM send to topic failed")
				}
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:        "Missing title",
			requestBody: PushTopicRequest{Body: "Body only", Topic: "t"},
			setupMock:   func(mockFCM *fcmHandlerClient) { /* No FCM call */ },
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Missing body",
			requestBody: PushTopicRequest{Title: "Title only", Topic: "t"},
			setupMock:   func(mockFCM *fcmHandlerClient) { /* No FCM call */ },
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Missing topic",
			requestBody: PushTopicRequest{Title: "Title", Body: "Body"},
			setupMock:   func(mockFCM *fcmHandlerClient) { /* No FCM call */ },
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deviceStore := store.NewDeviceStore()
			// handlers パッケージ内の newFcmHandlerClient を呼び出す (ビルドタグで実体が切り替わる)
			mockFCMClient, err := newFcmHandlerClient(context.Background())
			if err != nil {
				t.Fatalf("Failed to create mock FCMClient: %v", err)
			}

			originalMockSendToTopic := mockFCMClient.MockSendToTopic
			if tt.setupMock != nil {
				tt.setupMock(mockFCMClient)
			}

			// NewPushTopicHandler は *fcmHandlerClient を受け取るように修正済みのはず
			handler := NewPushTopicHandler(mockFCMClient, deviceStore)

			reqBodyBytes, _ := json.Marshal(tt.requestBody)
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
	deviceStore := store.NewDeviceStore()
	mockFCMClient, _ := newFcmHandlerClient(context.Background()) // ここも変更
	handler := NewPushTopicHandler(mockFCMClient, deviceStore)
	req := httptest.NewRequest(http.MethodGet, "/pubsub/push/topic", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %v for GET, got %v", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestPushTopicHandler_ServeHTTP_InvalidJSON(t *testing.T) {
	deviceStore := store.NewDeviceStore()
	mockFCMClient, _ := newFcmHandlerClient(context.Background()) // ここも変更
	handler := NewPushTopicHandler(mockFCMClient, deviceStore)
	req := httptest.NewRequest(http.MethodPost, "/pubsub/push/topic", strings.NewReader("this is not json"))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %v for invalid JSON, got %v", http.StatusBadRequest, rr.Code)
	}
}
