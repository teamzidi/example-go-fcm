package handlers_test

import (
	"encoding/base64"
	"encoding/json"
	"errors"

	// Import handlers to access PubSubPushRequest, PubSubInternalMessage, etc.
	// These types are used by the helper functions.
	// The dot import should be avoided if specific types can be prefixed.
	// However, the test files were using dot import for handlers.
	// For helpers, it might be cleaner to use `handlers.PubSubPushRequest`.
	// Let's try to use specific imports if the types are exported from `handlers`.
	// Assuming PubSubPushRequest and PubSubInternalMessage are exported by `handlers`.
	"github.com/teamzidi/example-go-fcm/handlers"
)

var (
	// These errors are used to simulate FCM responses for testing.
	// The real `fcm.IsRetryableError` function's behavior with these errors is key.
	errFCMRetryable    = errors.New("fcm: simulated retryable error for test")
	errFCMNonRetryable = errors.New("fcm: simulated non-retryable error for test")
)

// newPushPubSubRequest encodes a payload into the Pub/Sub message structure.
func newPushPubSubRequest(payload interface{}) []byte {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		panic("Failed to marshal test payload: " + err.Error())
	}
	req := handlers.PubSubPushRequest{ // Prefixed with handlers.
		Message: handlers.PubSubInternalMessage{ // Prefixed with handlers.
			Data:        base64.StdEncoding.EncodeToString(payloadBytes),
			MessageID:   "test-message-id",
			PublishTime: "test-publish-time",
		},
		Subscription: "test-subscription",
	}
	requestBytes, err := json.Marshal(&req)
	if err != nil {
		panic("Failed to marshal PubSubPushRequest: " + err.Error())
	}
	return requestBytes
}

// newPushPubSubRequestRawData creates a Pub/Sub message with raw, unencoded data string for Message.Data.
func newPushPubSubRequestRawData(rawData string) []byte {
	req := handlers.PubSubPushRequest{ // Prefixed with handlers.
		Message: handlers.PubSubInternalMessage{ // Prefixed with handlers.
			Data:        rawData, // Data is already string, might be invalid base64 or empty
			MessageID:   "test-raw-data-message-id",
			PublishTime: "test-publish-time",
		},
		Subscription: "test-subscription",
	}
	requestBytes, err := json.Marshal(&req)
	if err != nil {
		panic("Failed to marshal PubSubPushRequest with raw data: " + err.Error())
	}
	return requestBytes
}
