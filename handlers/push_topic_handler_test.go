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
	"reflect" // reflect を追加

	"firebase.google.com/go/v4/messaging"
	"github.com/teamzidi/example-go-fcm/fcm"
	"github.com/teamzidi/example-go-fcm/store"
)

// MockFCMClientForTopicTests は fcm.FCMClientInterface のモック実装です。
type MockFCMClientForTopicTests struct {
	SendFunc                 func(ctx context.Context, message *messaging.Message) (string, error)
	SendToTokenFunc          func(ctx context.Context, token string, title string, body string) (string, error)
	SendToMultipleTokensFunc func(ctx context.Context, tokens []string, title string, body string) (*messaging.BatchResponse, error)
}

func (m *MockFCMClientForTopicTests) Send(ctx context.Context, message *messaging.Message) (string, error) {
	if m.SendFunc != nil {
		return m.SendFunc(ctx, message)
	}
	return "", errors.New("SendFunc not implemented")
}

func (m *MockFCMClientForTopicTests) SendToToken(ctx context.Context, token string, title string, body string) (string, error) {
	// Not used in topic tests directly, but needed for interface
	return "", errors.New("SendToTokenFunc not implemented for topic mock")
}

func (m *MockFCMClientForTopicTests) SendToMultipleTokens(ctx context.Context, tokens []string, title string, body string) (*messaging.BatchResponse, error) {
	// Not used in topic tests directly, but needed for interface
	return nil, errors.New("SendToMultipleTokensFunc not implemented for topic mock")
}


func TestPushTopicHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name                string
		requestBody         PushTopicRequest
		mockFCMSendResponse string // messageID
		mockFCMSendError    error
		expectedStatus      int
		expectedFCMCallCount int
		expectedTopicSent   string
		expectedTitleSent   string
		expectedBodySent    string
		expectedDataSent    map[string]string
	}{
		{
			name: "Successful push to topic",
			requestBody: PushTopicRequest{
				Title: "Topic Test Title",
				Body:  "Topic Test Body",
				Topic: "test_topic",
				CustomData: map[string]string{"key": "value"},
			},
			mockFCMSendResponse: "mock_message_id_1",
			expectedStatus:      http.StatusOK,
			expectedFCMCallCount:1,
			expectedTopicSent:   "test_topic",
			expectedTitleSent:   "Topic Test Title",
			expectedBodySent:    "Topic Test Body",
			expectedDataSent:    map[string]string{"key": "value"},
		},
		{
			name: "Successful push to topic no custom data",
			requestBody: PushTopicRequest{
				Title: "Topic Test Title No Data",
				Body:  "Topic Test Body No Data",
				Topic: "test_topic_no_data",
				// CustomData is nil or empty
			},
			mockFCMSendResponse: "mock_message_id_2",
			expectedStatus:      http.StatusOK,
			expectedFCMCallCount:1,
			expectedTopicSent:   "test_topic_no_data",
			expectedTitleSent:   "Topic Test Title No Data",
			expectedBodySent:    "Topic Test Body No Data",
			expectedDataSent:    nil, // or map[string]string{}
		},
		{
			name: "FCM client returns error on topic push",
			requestBody: PushTopicRequest{
				Title: "Topic Error Title",
				Body:  "Topic Error Body",
				Topic: "error_topic",
			},
			mockFCMSendError:     errors.New("FCM send to topic failed"),
			expectedStatus:       http.StatusServiceUnavailable,
			expectedFCMCallCount: 1,
			expectedTopicSent:   "error_topic",
			expectedTitleSent:   "Topic Error Title",
			expectedBodySent:    "Topic Error Body",
		},
		{
			name:                 "Missing title",
			requestBody:          PushTopicRequest{Body: "Body only", Topic: "t"},
			expectedStatus:       http.StatusBadRequest,
			expectedFCMCallCount: 0,
		},
		{
			name:                 "Missing body",
			requestBody:          PushTopicRequest{Title: "Title only", Topic: "t"},
			expectedStatus:       http.StatusBadRequest,
			expectedFCMCallCount: 0,
		},
		{
			name:                 "Missing topic",
			requestBody:          PushTopicRequest{Title: "Title", Body: "Body"},
			expectedStatus:       http.StatusBadRequest,
			expectedFCMCallCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deviceStore := store.NewDeviceStore() // Not used by this handler but part of constructor
			mockClient := &MockFCMClientForTopicTests{}
			var fcmCallCount int
			var sentMessage *messaging.Message

			mockClient.SendFunc = func(ctx context.Context, msg *messaging.Message) (string, error) {
				fcmCallCount++
				sentMessage = msg
				return tt.mockFCMSendResponse, tt.mockFCMSendError
			}

			handler := NewPushTopicHandler(mockClient, deviceStore)

			reqBodyBytes, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/pubsub/push/topic", bytes.NewBuffer(reqBodyBytes))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Status = %v, want %v. Body: %s", rr.Code, tt.expectedStatus, rr.Body.String())
			}
			if fcmCallCount != tt.expectedFCMCallCount {
				t.Errorf("FCM CallCount = %d, want %d", fcmCallCount, tt.expectedFCMCallCount)
			}

			if tt.expectedFCMCallCount > 0 && sentMessage != nil {
				if sentMessage.Topic != tt.expectedTopicSent {
					t.Errorf("TopicSent = %q, want %q", sentMessage.Topic, tt.expectedTopicSent)
				}
				if sentMessage.Notification == nil {
					t.Fatalf("sentMessage.Notification is nil, want Title=%q, Body=%q", tt.expectedTitleSent, tt.expectedBodySent)
				}
				if sentMessage.Notification.Title != tt.expectedTitleSent {
					t.Errorf("TitleSent = %q, want %q", sentMessage.Notification.Title, tt.expectedTitleSent)
				}
				if sentMessage.Notification.Body != tt.expectedBodySent {
					t.Errorf("BodySent = %q, want %q", sentMessage.Notification.Body, tt.expectedBodySent)
				}

				// Use reflect.DeepEqual for comparing maps
				if !reflect.DeepEqual(sentMessage.Data, tt.expectedDataSent) {
					// Handle case where tt.expectedDataSent is nil but sentMessage.Data is empty map (or vice versa)
					if (len(sentMessage.Data) == 0 && tt.expectedDataSent == nil) || (sentMessage.Data == nil && len(tt.expectedDataSent) == 0) {
						// This is acceptable
					} else {
						t.Errorf("DataSent = %v, want %v", sentMessage.Data, tt.expectedDataSent)
					}
				}
			}
		})
	}
}

// Test for invalid HTTP method
func TestPushTopicHandler_ServeHTTP_InvalidMethod(t *testing.T) {
	deviceStore := store.NewDeviceStore()
	mockClient := &MockFCMClientForTopicTests{}
	handler := NewPushTopicHandler(mockClient, deviceStore)

	req := httptest.NewRequest(http.MethodGet, "/pubsub/push/topic", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %v for GET, got %v", http.StatusMethodNotAllowed, rr.Code)
	}
}

// Test for invalid JSON body
func TestPushTopicHandler_ServeHTTP_InvalidJSON(t *testing.T) {
	deviceStore := store.NewDeviceStore()
	mockClient := &MockFCMClientForTopicTests{}
	handler := NewPushTopicHandler(mockClient, deviceStore)

	req := httptest.NewRequest(http.MethodPost, "/pubsub/push/topic", strings.NewReader("this is not json"))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %v for invalid JSON, got %v", http.StatusBadRequest, rr.Code)
	}
}
