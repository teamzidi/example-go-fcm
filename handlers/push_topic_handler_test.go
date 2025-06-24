package handlers_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/teamzidi/example-go-fcm/handlers"
)

func TestPushTopicHandler_Comprehensive(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           []byte
		mockSendFunc   func(ctx context.Context, topic string, title string, body string, customData map[string]string) (string, error)
		expectedStatus int
	}{
		{
			name:   "successful FCM send to topic",
			method: http.MethodPost,
			// Uses newPushPubSubRequest from test_helpers_test.go
			// Needs TopicPushPayload from "github.com/teamzidi/example-go-fcm/handlers"
			body: newPushPubSubRequest(TopicPushPayload{Title: "Title", Body: "Body", Topic: "topic-name"}),
			mockSendFunc: func(ctx context.Context, topic, title, body string, customData map[string]string) (string, error) {
				return "fcm-topic-success-id", nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "retryable FCM error on send to topic",
			method: http.MethodPost,
			body:   newPushPubSubRequest(TopicPushPayload{Title: "Title", Body: "Body", Topic: "topic-retry"}),
			mockSendFunc: func(ctx context.Context, topic, title, body string, customData map[string]string) (string, error) {
				return "", errors.New("retryable")
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "non-retryable FCM error on send to topic",
			method: http.MethodPost,
			body:   newPushPubSubRequest(TopicPushPayload{Title: "Title", Body: "Body", Topic: "topic-nonretry"}),
			mockSendFunc: func(ctx context.Context, topic, title, body string, customData map[string]string) (string, error) {
				return "", errors.New("retryable")
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "invalid HTTP method",
			method:         http.MethodGet,
			body:           nil,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Pub/Sub envelope decoding error (malformed JSON)",
			method:         http.MethodPost,
			body:           []byte("this is not json"),
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "empty Pub/Sub message data (Data field is empty string)",
			method:         http.MethodPost,
			body:           newPushPubSubRequest([]byte("")), // Uses newPushPubSubRequestRawData from test_helpers_test.go
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "base64 decoding error for message data (Data field is invalid base64)",
			method:         http.MethodPost,
			body:           newPushPubSubRequest([]byte("!@#$ThisIsNotBase64")),
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "payload unmarshalling error (decoded data is not valid TopicPushPayload JSON)",
			method:         http.MethodPost,
			body:           newPushPubSubRequest(base64.StdEncoding.EncodeToString([]byte("this is not topic push payload json"))),
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "missing Title in payload",
			method:         http.MethodPost,
			body:           newPushPubSubRequest(TopicPushPayload{Body: "Body", Topic: "topic-no-title"}),
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "missing Body in payload",
			method:         http.MethodPost,
			body:           newPushPubSubRequest(TopicPushPayload{Title: "Title", Topic: "topic-no-body"}),
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "missing Topic in payload",
			method:         http.MethodPost,
			body:           newPushPubSubRequest(TopicPushPayload{Title: "Title", Body: "Body"}),
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "empty HTTP request body",
			method:         http.MethodPost,
			body:           []byte{},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "HTTP request body is JSON null",
			method:         http.MethodPost,
			body:           []byte("null"),
			expectedStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use handlers.MockFCMClient from handlers/mock_test.go
			mockClient := &MockFCMClient{
				MockSendToTopic: tt.mockSendFunc,
			}

			// The PushTopicHandler type is from the dot-imported "handlers" package.
			handler := new(PushTopicHandler).WithMock(mockClient)

			req := httptest.NewRequest(tt.method, "/", bytes.NewBuffer(tt.body))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code for '%s': got %v want %v. Body: %s",
					tt.name, rr.Code, tt.expectedStatus, rr.Body.String())
			}
		})
	}
}
