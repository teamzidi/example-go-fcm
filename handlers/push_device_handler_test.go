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

	"firebase.google.com/go/v4/messaging"
	"github.com/teamzidi/example-go-fcm/fcm"
	"github.com/teamzidi/example-go-fcm/store"
)

// MockFCMClientForDeviceTests は fcm.FCMClientInterface のモック実装です。
// (push_handler_test.go から類似のものを再定義するか、共通化を検討)
type MockFCMClientForDeviceTests struct {
	SendFunc                 func(ctx context.Context, message *messaging.Message) (string, error)
	SendToTokenFunc          func(ctx context.Context, token string, title string, body string) (string, error)
	SendToMultipleTokensFunc func(ctx context.Context, tokens []string, title string, body string) (*messaging.BatchResponse, error)
}

func (m *MockFCMClientForDeviceTests) Send(ctx context.Context, message *messaging.Message) (string, error) {
	if m.SendFunc != nil {
		return m.SendFunc(ctx, message)
	}
	return "", errors.New("SendFunc not implemented")
}

func (m *MockFCMClientForDeviceTests) SendToToken(ctx context.Context, token string, title string, body string) (string, error) {
	if m.SendToTokenFunc != nil {
		return m.SendToTokenFunc(ctx, token, title, body)
	}
	return "", errors.New("SendToTokenFunc not implemented")
}

func (m *MockFCMClientForDeviceTests) SendToMultipleTokens(ctx context.Context, tokens []string, title string, body string) (*messaging.BatchResponse, error) {
	if m.SendToMultipleTokensFunc != nil {
		return m.SendToMultipleTokensFunc(ctx, tokens, title, body)
	}
	return nil, errors.New("SendToMultipleTokensFunc not implemented")
}


func TestPushDeviceHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name                 string
		requestBody          PushDeviceRequest
		mockFCMResponse      *messaging.BatchResponse
		mockFCMError         error
		expectedStatus       int
		expectedFCMCallCount int
		expectedTokensSent   []string
		expectedTitleSent    string
		expectedBodySent     string
	}{
		{
			name: "Successful push to devices",
			requestBody: PushDeviceRequest{
				Title:  "Device Test Title",
				Body:   "Device Test Body",
				Tokens: []string{"dev_token1", "dev_token2"},
			},
			mockFCMResponse:      &messaging.BatchResponse{SuccessCount: 2, FailureCount: 0},
			expectedStatus:       http.StatusOK,
			expectedFCMCallCount: 1,
			expectedTokensSent:   []string{"dev_token1", "dev_token2"},
			expectedTitleSent:    "Device Test Title",
			expectedBodySent:     "Device Test Body",
		},
		{
			name: "FCM client returns error on device push",
			requestBody: PushDeviceRequest{
				Title:  "Device Error Title",
				Body:   "Device Error Body",
				Tokens: []string{"dev_token_err"},
			},
			mockFCMError:         errors.New("FCM send to devices failed"),
			expectedStatus:       http.StatusServiceUnavailable,
			expectedFCMCallCount: 1,
			expectedTokensSent:   []string{"dev_token_err"},
			expectedTitleSent:    "Device Error Title",
			expectedBodySent:     "Device Error Body",
		},
		{
			name:                 "Missing title",
			requestBody:          PushDeviceRequest{Body: "Body only", Tokens: []string{"t1"}},
			expectedStatus:       http.StatusBadRequest,
			expectedFCMCallCount: 0,
		},
		{
			name:                 "Missing body",
			requestBody:          PushDeviceRequest{Title: "Title only", Tokens: []string{"t1"}},
			expectedStatus:       http.StatusBadRequest,
			expectedFCMCallCount: 0,
		},
		{
			name:                 "Missing tokens",
			requestBody:          PushDeviceRequest{Title: "Title", Body: "Body", Tokens: []string{}},
			expectedStatus:       http.StatusBadRequest,
			expectedFCMCallCount: 0,
		},
		{
			name:                 "Too many tokens",
			requestBody:          PushDeviceRequest{Title: "Title", Body: "Body", Tokens: make([]string, 501)},
			expectedStatus:       http.StatusBadRequest,
			expectedFCMCallCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deviceStore := store.NewDeviceStore() // Not directly used by this handler but part of constructor
			mockClient := &MockFCMClientForDeviceTests{}
			var fcmCallCount int
			var tokensSent []string
			var titleSent, bodySent string

			mockClient.SendToMultipleTokensFunc = func(ctx context.Context, tkns []string, ttl string, bdy string) (*messaging.BatchResponse, error) {
				fcmCallCount++
				tokensSent = tkns
				titleSent = ttl
				bodySent = bdy
				return tt.mockFCMResponse, tt.mockFCMError
			}

			handler := NewPushDeviceHandler(mockClient, deviceStore)

			reqBodyBytes, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/pubsub/push/device", bytes.NewBuffer(reqBodyBytes))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Status = %v, want %v. Body: %s", rr.Code, tt.expectedStatus, rr.Body.String())
			}

			if fcmCallCount != tt.expectedFCMCallCount {
				t.Errorf("FCM CallCount = %d, want %d", fcmCallCount, tt.expectedFCMCallCount)
			}

			if tt.expectedFCMCallCount > 0 {
				// Basic check, could be more thorough (e.g. reflect.DeepEqual on sorted slices)
				if len(tokensSent) != len(tt.expectedTokensSent) {
					t.Errorf("TokensSent length = %d, want %d. Got: %v, Want: %v", len(tokensSent), len(tt.expectedTokensSent), tokensSent, tt.expectedTokensSent)
				}
				if titleSent != tt.expectedTitleSent {
					t.Errorf("TitleSent = %q, want %q", titleSent, tt.expectedTitleSent)
				}
				if bodySent != tt.expectedBodySent {
					t.Errorf("BodySent = %q, want %q", bodySent, tt.expectedBodySent)
				}
			}
		})
	}
}

// Test for invalid HTTP method
func TestPushDeviceHandler_ServeHTTP_InvalidMethod(t *testing.T) {
	deviceStore := store.NewDeviceStore()
	mockClient := &MockFCMClientForDeviceTests{}
	handler := NewPushDeviceHandler(mockClient, deviceStore)

	req := httptest.NewRequest(http.MethodGet, "/pubsub/push/device", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %v for GET, got %v", http.StatusMethodNotAllowed, rr.Code)
	}
}

// Test for invalid JSON body
func TestPushDeviceHandler_ServeHTTP_InvalidJSON(t *testing.T) {
	deviceStore := store.NewDeviceStore()
	mockClient := &MockFCMClientForDeviceTests{}
	handler := NewPushDeviceHandler(mockClient, deviceStore)

	req := httptest.NewRequest(http.MethodPost, "/pubsub/push/device", strings.NewReader("this is not json"))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %v for invalid JSON, got %v", http.StatusBadRequest, rr.Code)
	}
}
