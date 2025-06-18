package handlers

import (
	"bytes"
	"context"
	"encoding/base64" // Base64エンコードのためにインポート
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	// "reflect" // DeepEqual を使用しない場合は不要
)

// 注意: このテストファイルが正しく動作するためには、
// go test -tags=test_fcm_mock ./... のようにビルドタグを指定して実行し、
// handlers/fcm_client_config_mock.go がビルドされるようにする必要がある。

// Helper function to create a PubSubPushRequest for device push tests
func newDevicePushPubSubRequest(payload DevicePushPayload) PubSubPushRequest {
	payloadBytes, _ := json.Marshal(payload)
	return PubSubPushRequest{
		Message: PubSubInternalMessage{
			Data:        base64.StdEncoding.EncodeToString(payloadBytes),
			MessageID:   "test-message-id",
			PublishTime: "test-publish-time",
		},
		Subscription: "test-subscription",
	}
}

func TestPushDeviceHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name                string
		actualPayload       DevicePushPayload // Base64デコード後のペイロード
		emptyData           bool              // Message.Data を空にするか
		invalidBase64       bool              // Message.Data を不正なBase64にするか
		setupMock           func(mockFCM *fcmHandlerClient)
		expectedStatus      int
	}{
		{
			name: "Successful push to single device with custom data",
			actualPayload: DevicePushPayload{
				Title:  "Device Test Title",
				Body:   "Device Test Body",
				Token:  "dev_token1",
				CustomData: map[string]string{"type": "device_push"},
			},
			setupMock: func(mockFCM *fcmHandlerClient) {
				mockFCM.MockSendToToken = func(ctx context.Context, token string, title string, body string, customData map[string]string) (string, error) {
					if token != "dev_token1" {
						t.Errorf("Mock: Token mismatch. Got %s", token)
					}
					// ... (他のアサーションは省略)
					return "mock-message-id-single-device", nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "FCM client returns error on single device push",
			actualPayload: DevicePushPayload{
				Title:  "Device Error Title",
				Body:   "Device Error Body",
				Token:  "dev_token_err",
			},
			setupMock: func(mockFCM *fcmHandlerClient) {
				mockFCM.MockSendToToken = func(ctx context.Context, token string, title string, body string, customData map[string]string) (string, error) {
					return "", errors.New("FCM send to device failed")
				}
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:        "Missing title in actual payload",
			actualPayload: DevicePushPayload{Body: "Body only", Token: "t1"}, // Title が空
			setupMock:   func(mockFCM *fcmHandlerClient) { /* No FCM call expected */ },
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Missing token in actual payload",
			actualPayload: DevicePushPayload{Title: "Title", Body: "Body", Token: ""}, // Token が空
			setupMock:   func(mockFCM *fcmHandlerClient) { /* No FCM call expected */ },
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
			actualPayload:  DevicePushPayload{Title: "T", Body: "B", Token: "t"}, // この中身は使われない
			setupMock:      func(mockFCM *fcmHandlerClient) { /* No FCM call */ },
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// deviceStore はもう使わない
			mockFCMClient, err := NewFcmHandlerClient(context.Background())
			if err != nil {
				t.Fatalf("Failed to create mock FCMClient: %v", err)
			}

			originalMockSendToToken := mockFCMClient.MockSendToToken
			if tt.setupMock != nil {
				tt.setupMock(mockFCMClient)
			}

			handler := NewPushDeviceHandler(mockFCMClient)

			var reqBodyBytes []byte
			if tt.emptyData {
				pubSubReq := PubSubPushRequest{Message: PubSubInternalMessage{Data: ""}}
				reqBodyBytes, _ = json.Marshal(pubSubReq)
			} else if tt.invalidBase64 {
				pubSubReq := PubSubPushRequest{Message: PubSubInternalMessage{Data: "this-is-not-base64"}}
				reqBodyBytes, _ = json.Marshal(pubSubReq)
			} else {
				pubSubReq := newDevicePushPubSubRequest(tt.actualPayload)
				reqBodyBytes, _ = json.Marshal(pubSubReq)
			}

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

// InvalidMethod と InvalidJSON のテストは、リクエストボディの最上位構造が変わったため、
// そのままだと PubSubPushRequest のデコードエラーになるか、あるいは別のエラーになる。
// PubSubPushRequest の形式で送る必要がある。
// ただし、これらのテストの主眼はメソッドチェックやトップレベルJSONの形式なので、
// PubSubPushRequestでラップした上で、中身の actualPayload は適当でよい。

func TestPushDeviceHandler_ServeHTTP_InvalidMethod(t *testing.T) {
	mockFCMClient, _ := NewFcmHandlerClient(context.Background())
	handler := NewPushDeviceHandler(mockFCMClient)

	// ボディはなんでもよいが、nilを渡す
	req := httptest.NewRequest(http.MethodGet, "/pubsub/push/device", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %v for GET, got %v", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestPushDeviceHandler_ServeHTTP_InvalidPubSubJSON(t *testing.T) {
	mockFCMClient, _ := NewFcmHandlerClient(context.Background())
	handler := NewPushDeviceHandler(mockFCMClient)

	// PubSubPushRequest として不正なJSON
	req := httptest.NewRequest(http.MethodPost, "/pubsub/push/device", strings.NewReader("this is not json"))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		// エラーメッセージも確認した方が良い: "Invalid Pub/Sub message format"
		t.Errorf("Expected status %v for invalid Pub/Sub JSON, got %v. Body: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}
