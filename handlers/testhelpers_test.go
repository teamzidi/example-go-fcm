package handlers_test

import (
	"encoding/base64"
	"encoding/json"

	"github.com/teamzidi/example-go-fcm/handlers"
)

// newPushPubSubRequest encodes a payload into the Pub/Sub message structure.
func newPushPubSubRequest(payload any) []byte {
	var payloadBytes []byte

	if b, ok := payload.([]byte); ok {
		payloadBytes = b
	} else {
		if b, err := json.Marshal(payload); err == nil {
			payloadBytes = b
		} else {
			panic("Failed to marshal test payload: " + err.Error())
		}
	}

	req := handlers.PubSubPushRequest{
		Message: handlers.PubSubInternalMessage{
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
