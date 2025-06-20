package handlers_test

import (
	"bytes"
	"context"
	"encoding/base64" // Base64エンコードのためにインポート
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/teamzidi/example-go-fcm/handlers"
)

// Helper function to create a PubSubPushRequest for device push tests
func newPushPubSubRequest(payload any) []byte {
	payloadBytes, _ := json.Marshal(payload)
	req := PubSubPushRequest{
		Message: PubSubInternalMessage{
			Data:        base64.StdEncoding.EncodeToString(payloadBytes),
			MessageID:   "test-message-id",
			PublishTime: "test-publish-time",
		},
		Subscription: "test-subscription",
	}

	bytes, err := json.Marshal(&req)
	if err != nil {
		panic("Failed to marshal PubSubPushRequest: " + err.Error())
	}

	return bytes
}

func TestPushDeviceHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		body           []byte
		mockFunc       func(ctx context.Context, token string, title string, body string, customData map[string]string) (string, error)
		expectedStatus int
	}{
		{
			name: "success",
			body: newPushPubSubRequest(DevicePushPayload{
				Title: "Device Test Title",
				Body:  "Device Test Body",
				Token: "dev_token1",
			}),
			mockFunc:       nil,
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing token",
			body: newPushPubSubRequest(DevicePushPayload{
				Title: "Device Error Title",
				Body:  "Device Error Body",
			}),
			mockFunc:       nil,
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing title",
			body: newPushPubSubRequest(DevicePushPayload{
				Body:  "Body",
				Token: "token",
			}),
			mockFunc:       nil,
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing body",
			body: newPushPubSubRequest(DevicePushPayload{
				Title: "Device Test Title",
				Token: "dev_token1",
			}),
			mockFunc:       nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "empty request body",
			body:           nil,
			mockFunc:       nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "empty data",
			body:           newPushPubSubRequest(nil),
			mockFunc:       nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid request body",
			body:           []byte("hello, world!"),
			mockFunc:       nil,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockFCMClient{MockSendToToken: tt.mockFunc}
			handler := new(PushDeviceHandler).WithMock(mock)

			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(tt.body))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)
			if rr.Code != tt.expectedStatus {
				t.Errorf("Status = %v, want %v. Body: %s", rr.Code, tt.expectedStatus, rr.Body.String())
			}
		})
	}
}

func TestPushDeviceHandler_ServeHTTP_InvalidMethod(t *testing.T) {
	mock := new(MockFCMClient)
	handler := new(PushDeviceHandler).WithMock(mock)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %v for GET, got %v", http.StatusMethodNotAllowed, rr.Code)
	}
}
