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
	// firebase messaging はモックのシグネチャで直接は使わない
	// "firebase.google.com/go/v4/messaging"
)

// 注意: このテストファイルが正しく動作するためには、
// go test -tags=test_fcm_mock ./... のようにビルドタグを指定して実行し、
// handlers/fcm_client_config_mock.go がビルドされるようにする必要がある。

func TestPushDeviceHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name                string
		requestBody         PushDeviceRequest
		setupMock           func(mockFCM *fcmHandlerClient) // 型を *fcmHandlerClient に変更
		expectedStatus      int
	}{
		{
			name: "Successful push to single device with custom data",
			requestBody: PushDeviceRequest{
				Title:  "Device Test Title",
				Body:   "Device Test Body",
				Token:  "dev_token1",
				CustomData: map[string]string{"type": "device_push"},
			},
			setupMock: func(mockFCM *fcmHandlerClient) { // 型を *fcmHandlerClient に変更
				mockFCM.MockSendToToken = func(ctx context.Context, token string, title string, body string, customData map[string]string) (string, error) {
					if token != "dev_token1" {
						t.Errorf("Mock: Token mismatch. Got %s", token)
					}
					if title != "Device Test Title" {
						t.Errorf("Mock: Title mismatch. Got %s", title)
					}
					if !reflect.DeepEqual(customData, map[string]string{"type": "device_push"}) {
						t.Errorf("Mock: CustomData mismatch. Got %v", customData)
					}
					return "mock-message-id-single-device", nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "FCM client returns error on single device push",
			requestBody: PushDeviceRequest{
				Title:  "Device Error Title",
				Body:   "Device Error Body",
				Token:  "dev_token_err",
			},
			setupMock: func(mockFCM *fcmHandlerClient) { // 型を *fcmHandlerClient に変更
				mockFCM.MockSendToToken = func(ctx context.Context, token string, title string, body string, customData map[string]string) (string, error) {
					return "", errors.New("FCM send to device failed")
				}
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:        "Missing title",
			requestBody: PushDeviceRequest{Body: "Body only", Token: "t1"},
			setupMock:   func(mockFCM *fcmHandlerClient) { /* No FCM call expected */ },
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Missing body",
			requestBody: PushDeviceRequest{Title: "Title only", Token: "t1"},
			setupMock:   func(mockFCM *fcmHandlerClient) { /* No FCM call expected */ },
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Missing token",
			requestBody: PushDeviceRequest{Title: "Title", Body: "Body", Token: ""},
			setupMock:   func(mockFCM *fcmHandlerClient) { /* No FCM call expected */ },
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

			originalMockSendToToken := mockFCMClient.MockSendToToken
			if tt.setupMock != nil {
				tt.setupMock(mockFCMClient)
			}

			// NewPushDeviceHandler は *fcmHandlerClient を受け取るように修正済みのはず
			handler := NewPushDeviceHandler(mockFCMClient, deviceStore)

			reqBodyBytes, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/pubsub/push/device", bytes.NewBuffer(reqBodyBytes))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Status = %v, want %v. Body: %s", rr.Code, tt.expectedStatus, rr.Body.String())
			}

			mockFCMClient.MockSendToToken = originalMockSendToToken
		})
	}
}

func TestPushDeviceHandler_ServeHTTP_InvalidMethod(t *testing.T) {
	deviceStore := store.NewDeviceStore()
	mockFCMClient, _ := newFcmHandlerClient(context.Background()) // ここも変更
	handler := NewPushDeviceHandler(mockFCMClient, deviceStore)
	req := httptest.NewRequest(http.MethodGet, "/pubsub/push/device", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %v for GET, got %v", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestPushDeviceHandler_ServeHTTP_InvalidJSON(t *testing.T) {
	deviceStore := store.NewDeviceStore()
	mockFCMClient, _ := newFcmHandlerClient(context.Background()) // ここも変更
	handler := NewPushDeviceHandler(mockFCMClient, deviceStore)
	req := httptest.NewRequest(http.MethodPost, "/pubsub/push/device", strings.NewReader("this is not json"))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %v for invalid JSON, got %v", http.StatusBadRequest, rr.Code)
	}
}
