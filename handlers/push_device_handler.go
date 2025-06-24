package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/teamzidi/example-go-fcm/fcm"
)

// DevicePushPayload は /pubsub/push/device エンドポイントでPub/Subメッセージの
// Base64デコードされた data フィールドが示す実際の業務ペイロード構造体です。
type DevicePushPayload struct {
	Title      string            `json:"title"`
	Body       string            `json:"body"`
	Token      string            `json:"token"`
	CustomData map[string]string `json:"custom_data,omitempty"`
}

// PushDeviceHandler は特定の単一デバイストークンへのPush通知を処理します。
type PushDeviceHandler struct {
	fcmClient fcmClient
}

func NewPushDeviceHandler(fc *fcm.Client) *PushDeviceHandler {
	return &PushDeviceHandler{
		fcmClient: fc,
	}
}

// ServeHTTP はHTTPリクエストを処理します。
func (h *PushDeviceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("PushDeviceHandler: Invalid request method: %s", r.Method)
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	decodedData, err := decodeData(r.Body)
	if err != nil {
		log.Printf("PushDeviceHandler: decoding data: %v", err)
		w.WriteHeader(http.StatusNoContent) // New: Ack with 204
		return
	}

	messageID, err := h.send(decodedData)
	if err != nil {
		if IsRetryable(err) {
			http.Error(w, "Failed to send notification via FCM (retryable)", http.StatusInternalServerError) // Nack
		} else {
			w.WriteHeader(http.StatusNoContent) // Ack
		}

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "processed", // "processed" indicates successful delivery or a non-retryable FCM error.
		"message_id": messageID,
	}); err != nil {
		log.Printf("PushDeviceHandler: Error encoding success response: %v\n", err)
	}
}

func (h *PushDeviceHandler) send(decodedData []byte) (string, error) {
	var payload DevicePushPayload
	if err := json.Unmarshal(decodedData, &payload); err != nil {
		return "", fmt.Errorf("unmarshalling actual payload: %v. Decoded data was: %s", err, string(decodedData))
	}

	if payload.Title == "" {
		return "", fmt.Errorf("title is required in payload")
	}

	if payload.Body == "" {
		return "", fmt.Errorf("body is required in payload")
	}

	if payload.Token == "" {
		return "", fmt.Errorf("token is required in payload")
	}

	log.Printf("sending notification to device token '%s'. Title: '%s', Data: %v\n",
		payload.Token, payload.Title, payload.CustomData)

	// FCM送信
	messageID, err := h.fcmClient.SendToToken(context.Background(), payload.Token, payload.Title, payload.Body, payload.CustomData)
	if err != nil {
		return "", fmt.Errorf("sending FCM message to token %s: %v", payload.Token, err) // Log error
	}

	log.Printf("PushDeviceHandler: Successfully sent message ID %s to token %s", messageID, payload.Token)

	return messageID, nil
}
