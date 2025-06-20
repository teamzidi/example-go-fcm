package handlers_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/teamzidi/example-go-fcm/handlers"
)

func TestPushTopicHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		body           []byte
		mockFunc       func(ctx context.Context, Topic string, title string, body string, customData map[string]string) (string, error)
		expectedStatus int
	}{
		{
			name: "success",
			body: newPushPubSubRequest(TopicPushPayload{
				Title: "Topic Test Title",
				Body:  "Topic Test Body",
				Topic: "dev_Topic1",
			}),
			mockFunc:       nil,
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing Topic",
			body: newPushPubSubRequest(TopicPushPayload{
				Title: "Topic Error Title",
				Body:  "Topic Error Body",
			}),
			mockFunc:       nil,
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing title",
			body: newPushPubSubRequest(TopicPushPayload{
				Body:  "Body",
				Topic: "Topic",
			}),
			mockFunc:       nil,
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing body",
			body: newPushPubSubRequest(TopicPushPayload{
				Title: "Topic Test Title",
				Topic: "dev_Topic1",
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
			mock := &MockFCMClient{MockSendToTopic: tt.mockFunc}
			handler := new(PushTopicHandler).WithMock(mock)

			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(tt.body))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)
			if rr.Code != tt.expectedStatus {
				t.Errorf("Status = %v, want %v. Body: %s", rr.Code, tt.expectedStatus, rr.Body.String())
			}
		})
	}
}

func TestPushTopicHandler_ServeHTTP_InvalidMethod(t *testing.T) {
	mock := new(MockFCMClient)
	handler := new(PushTopicHandler).WithMock(mock)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %v for GET, got %v", http.StatusMethodNotAllowed, rr.Code)
	}
}
