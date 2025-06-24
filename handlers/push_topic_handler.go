package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/teamzidi/example-go-fcm/fcm"
)

// TopicPushPayload は /pubsub/push/Topic エンドポイントでPub/Subメッセージの
// Base64デコードされた data フィールドが示す実際の業務ペイロード構造体です。
type TopicPushPayload struct {
	Title      string            `json:"title"`
	Body       string            `json:"body"`
	Topic      string            `json:"topic"`
	CustomData map[string]string `json:"custom_data,omitempty"`
}

// PushTopicHandler は特定の単一デバイストークンへのPush通知を処理します。
type PushTopicHandler struct {
	fcmClient fcmClient
}

func NewPushTopicHandler(fc *fcm.Client) *PushTopicHandler {
	return &PushTopicHandler{
		fcmClient: fc,
	}
}

// ServeHTTP はHTTPリクエストを処理します。
func (h *PushTopicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
		"status":     "processed",
		"message_id": messageID,
	}); err != nil {
		log.Printf("PushDeviceHandler: Error encoding success response: %v\n", err)
	}
}

func (h *PushTopicHandler) send(decodedData []byte) (string, error) {
	var payload TopicPushPayload
	if err := json.Unmarshal(decodedData, &payload); err != nil {
		return "", fmt.Errorf("unmarshalling payload: %v. Decoded data was: %s", err, string(decodedData))
	}

	if payload.Title == "" {
		return "", fmt.Errorf("title is required in payload")
	}

	if payload.Body == "" {
		return "", fmt.Errorf("body is required in payload")
	}

	if payload.Topic == "" {
		return "", fmt.Errorf("topic is required in payload")
	}

	log.Printf("PushTopicHandler: Sending notification to Topic: topic=%q title=%q data=%v",
		payload.Topic, payload.Title, payload.CustomData)

	// FCM送信
	messageID, err := h.fcmClient.SendToTopic(context.Background(), payload.Topic, payload.Title, payload.Body, payload.CustomData)
	if err != nil {
		return "", fmt.Errorf("sending FCM message to topic %s: %v", payload.Topic, err)
	}

	log.Printf("PushTopicHandler: Successfully sent message ID %s to topic %s", messageID, payload.Topic)

	return "", nil
}
