package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect" // 追加
	"sort"    // 追加
	"testing"

	"firebase.google.com/go/v4/messaging" // messagingパッケージをインポート
	"github.com/teamzidi/example-go-fcm/fcm"   // fcmパッケージをインポート
	"github.com/teamzidi/example-go-fcm/store"
)

// MockFCMClient は fcm.FCMClientInterface のモック実装です。
type MockFCMClient struct {
	SendToMultipleTokensFunc func(ctx context.Context, tokens []string, title string, body string) (*messaging.BatchResponse, error)
	SendToTokenFunc          func(ctx context.Context, token string, title string, body string) (string, error)
}

func (m *MockFCMClient) SendToMultipleTokens(ctx context.Context, tokens []string, title string, body string) (*messaging.BatchResponse, error) {
	if m.SendToMultipleTokensFunc != nil {
		return m.SendToMultipleTokensFunc(ctx, tokens, title, body)
	}
	return nil, errors.New("SendToMultipleTokensFunc not implemented in mock")
}

func (m *MockFCMClient) SendToToken(ctx context.Context, token string, title string, body string) (string, error) {
	if m.SendToTokenFunc != nil {
		return m.SendToTokenFunc(ctx, token, title, body)
	}
	return "", errors.New("SendToTokenFunc not implemented in mock")
}

func TestPushHandler_ServeHTTP(t *testing.T) {
	validPayload := ActualMessagePayload{Title: "Test Title", Body: "Test Body"}
	validPayloadBytes, _ := json.Marshal(validPayload)
	validBase64Payload := base64.StdEncoding.EncodeToString(validPayloadBytes)

	tests := []struct {
		name                 string
		method               string
		requestBody          PubSubPushRequest
		setupStore           func(ds *store.DeviceStore)
		mockFCMResponse      *messaging.BatchResponse
		mockFCMError         error
		expectedStatus       int
		expectedFCMCallCount int // SendToMultipleTokens が呼ばれた回数
		expectedTokensSentTo []string
		expectedTitleSent    string
		expectedBodySent     string
		expectedJSONResponse map[string]interface{} // 成功時のJSONレスポンスボディの期待値
	}{
		{
			name:   "Successful push notification",
			method: http.MethodPost,
			requestBody: PubSubPushRequest{
				Message:      PubSubPushMessage{Data: validBase64Payload, MessageID: "1"},
				Subscription: "sub1",
			},
			setupStore: func(ds *store.DeviceStore) {
				ds.AddToken("token1")
				ds.AddToken("token2")
			},
			mockFCMResponse:      &messaging.BatchResponse{SuccessCount: 2, FailureCount: 0},
			mockFCMError:         nil,
			expectedStatus:       http.StatusOK,
			expectedFCMCallCount: 1,
			expectedTokensSentTo: []string{"token1", "token2"},
			expectedTitleSent:    "Test Title",
			expectedBodySent:     "Test Body",
			expectedJSONResponse: map[string]interface{}{"status": "processed", "fcm_success_count": float64(2), "fcm_failure_count": float64(0)},
		},
		{
			name:   "FCM client returns error",
			method: http.MethodPost,
			requestBody: PubSubPushRequest{
				Message:      PubSubPushMessage{Data: validBase64Payload, MessageID: "2"},
				Subscription: "sub1",
			},
			setupStore: func(ds *store.DeviceStore) {
				ds.AddToken("token3")
			},
			mockFCMResponse:      nil,
			mockFCMError:         errors.New("FCM internal error"),
			expectedStatus:       http.StatusServiceUnavailable,
			expectedFCMCallCount: 1,
			expectedTokensSentTo: []string{"token3"},
			expectedTitleSent:    "Test Title",
			expectedBodySent:     "Test Body",
		},
		{
			name:   "No registered devices",
			method: http.MethodPost,
			requestBody: PubSubPushRequest{
				Message:      PubSubPushMessage{Data: validBase64Payload, MessageID: "3"},
				Subscription: "sub1",
			},
			setupStore:           func(ds *store.DeviceStore) {}, // No tokens
			expectedStatus:       http.StatusOK,                 // Acked
			expectedFCMCallCount: 0,                             // FCM should not be called
			expectedJSONResponse: map[string]interface{}{"status": "acknowledged", "reason": "no registered devices"},
		},
		{
			name:                 "Empty message data",
			method:               http.MethodPost,
			requestBody:          PubSubPushRequest{Message: PubSubPushMessage{Data: "", MessageID: "4"}},
			expectedStatus:       http.StatusOK, // Acked
			expectedFCMCallCount: 0,
			expectedJSONResponse: map[string]interface{}{"status": "acknowledged", "reason": "empty message data"},
		},
		{
			name:                 "Invalid base64 data",
			method:               http.MethodPost,
			requestBody:          PubSubPushRequest{Message: PubSubPushMessage{Data: "not-base64", MessageID: "5"}},
			expectedStatus:       http.StatusBadRequest,
			expectedFCMCallCount: 0,
		},
		{
			name:   "Invalid actual payload format",
			method: http.MethodPost,
			requestBody: PubSubPushRequest{
				Message: PubSubPushMessage{Data: base64.StdEncoding.EncodeToString([]byte("{\"bad\":\"json\"}")), MessageID: "6"},
			},
			expectedStatus:       http.StatusBadRequest,
			expectedFCMCallCount: 0,
		},
		{
			name:   "Empty title in actual payload",
			method: http.MethodPost,
			requestBody: PubSubPushRequest{
				Message: PubSubPushMessage{Data: base64.StdEncoding.EncodeToString([]byte("{\"body\":\"some body\"}")), MessageID: "7"},
			},
			expectedStatus:       http.StatusOK, // Acked
			expectedFCMCallCount: 0,
			expectedJSONResponse: map[string]interface{}{"status": "acknowledged", "reason": "empty title or body"},
		},
		{
			name:                 "Invalid HTTP method",
			method:               http.MethodGet,
			requestBody:          PubSubPushRequest{},
			expectedStatus:       http.StatusMethodNotAllowed,
			expectedFCMCallCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deviceStore := store.NewDeviceStore()
			if tt.setupStore != nil {
				tt.setupStore(deviceStore)
			}

			mockClient := &MockFCMClient{}
			var fcmCallCount int
			var tokensSentTo []string
			var titleSent, bodySent string

			mockClient.SendToMultipleTokensFunc = func(ctx context.Context, tokens []string, title string, body string) (*messaging.BatchResponse, error) {
				fcmCallCount++
				tokensSentTo = tokens // capture actual tokens passed
				titleSent = title
				bodySent = body
				return tt.mockFCMResponse, tt.mockFCMError
			}

			handler := NewPushHandler(mockClient, deviceStore)

			reqBodyBytes, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(tt.method, "/pubsub/push", bytes.NewBuffer(reqBodyBytes))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("ServeHTTP status = %v, want %v. Response body: %s", rr.Code, tt.expectedStatus, rr.Body.String())
			}

			if fcmCallCount != tt.expectedFCMCallCount {
				t.Errorf("Expected FCM SendToMultipleTokens to be called %d times, but called %d times", tt.expectedFCMCallCount, fcmCallCount)
			}

			if tt.expectedFCMCallCount > 0 {
				// Sort slices before comparison for consistency if order doesn't matter
				// For this test, the order of tokens from GetTokens() can be non-deterministic
				// if the underlying map iterates differently. So, sort both.
				sort.Strings(tokensSentTo)
				sort.Strings(tt.expectedTokensSentTo)

				if !reflect.DeepEqual(tokensSentTo, tt.expectedTokensSentTo) {
					t.Errorf("Expected tokens sent to FCM %v, got %v", tt.expectedTokensSentTo, tokensSentTo)
				}
				if titleSent != tt.expectedTitleSent {
					t.Errorf("Expected title sent to FCM %q, got %q", tt.expectedTitleSent, titleSent)
				}
				if bodySent != tt.expectedBodySent {
					t.Errorf("Expected body sent to FCM %q, got %q", tt.expectedBodySent, bodySent)
				}
			}

			if tt.expectedJSONResponse != nil {
				var actualResp map[string]interface{}
				if err := json.Unmarshal(rr.Body.Bytes(), &actualResp); err != nil {
					t.Fatalf("Failed to unmarshal response body: %v. Body: %s", err, rr.Body.String())
				}
				if !reflect.DeepEqual(actualResp, tt.expectedJSONResponse) {
					t.Errorf("Expected JSON response %v, got %v", tt.expectedJSONResponse, actualResp)
				}
			}
		})
	}
}

func TestPushHandler_ServeHTTP_InvalidBody(t *testing.T) {
	deviceStore := store.NewDeviceStore()
	mockClient := &MockFCMClient{}
	handler := NewPushHandler(mockClient, deviceStore)

	req := httptest.NewRequest(http.MethodPost, "/pubsub/push", bytes.NewBufferString("not a valid json"))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %v for invalid body, got %v", http.StatusBadRequest, rr.Code)
	}
}
