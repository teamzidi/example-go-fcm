package handlers

import (
	"context"
	"encoding/base64" // Base64デコードのためにインポート
	"encoding/json"
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
		log.Printf("PushTopicHandler: Invalid request method: %s\n", r.Method)
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed) // Reverted to original behavior
		return
	}

	var pubSubReq PubSubPushRequest
	if err := json.NewDecoder(r.Body).Decode(&pubSubReq); err != nil {
		log.Printf("PushTopicHandler: Error decoding Pub/Sub envelope: %v\n", err)
		w.WriteHeader(http.StatusNoContent) // New: Ack with 204
		return // bye
	}

	log.Printf("PushTopicHandler: Received Pub/Sub message ID %s from subscription %s published at %s\n",
		pubSubReq.Message.MessageID, pubSubReq.Subscription, pubSubReq.Message.PublishTime)

	if pubSubReq.Message.Data == "" {
		log.Println("PushTopicHandler: Pub/Sub message data is empty. Acking.")
		w.WriteHeader(http.StatusNoContent) // New: Ack with 204
		return // bye
	}

	decodedData, err := base64.StdEncoding.DecodeString(pubSubReq.Message.Data)
	if err != nil {
		log.Printf("PushTopicHandler: Error decoding base64 data: %v\n", err)
		w.WriteHeader(http.StatusNoContent) // New: Ack with 204
		return // bye
	}

	var payload TopicPushPayload
	if err := json.Unmarshal(decodedData, &payload); err != nil {
		log.Printf("PushTopicHandler: Error unmarshalling actual payload: %v. Decoded data was: %s\n", err, string(decodedData))
		w.WriteHeader(http.StatusNoContent) // New: Ack with 204
		return // bye
	}

	if payload.Title == "" {
		log.Println("PushTopicHandler: Title is required in payload")
		w.WriteHeader(http.StatusNoContent) // New: Ack with 204
		return // bye
	}

	if payload.Body == "" {
		log.Println("PushTopicHandler: Body is required in payload")
		w.WriteHeader(http.StatusNoContent) // New: Ack with 204
		return // bye
	}

	if payload.Topic == "" {
		log.Println("PushTopicHandler: Topic is required in payload")
		w.WriteHeader(http.StatusNoContent) // New: Ack with 204
		return // bye
	}

	log.Printf("PushTopicHandler: Sending notification to Topic: topic=%q title=%q data=%v",
		payload.Topic, payload.Title, payload.CustomData)

	// FCM送信
	messageID, err := h.fcmClient.SendToTopic(context.Background(), payload.Topic, payload.Title, payload.Body, payload.CustomData)
	if err != nil {
		log.Printf("PushTopicHandler: Error sending FCM message to topic %s: %v.\n", payload.Topic, err) // Log error
		if fcm.IsRetryableError(err) {
			http.Error(w, "Failed to send notification via FCM (retryable)", http.StatusInternalServerError) // 500
		} else {
			// Non-retryable errors are treated as "processed" from the perspective of the pub/sub queue,
			// so we return a 204 No Content to acknowledge the message without encouraging retries of this specific message.
			log.Printf("PushTopicHandler: Non-retryable error for topic %s. Acknowledging message with 204 No Content.\n", payload.Topic)
			w.WriteHeader(http.StatusNoContent) // 204
		}
		return
	}

	log.Printf("PushTopicHandler: Successfully sent message ID %s to topic %s\n", messageID, payload.Topic)

	w.Header().Set("Content-Type", "application/json")
	// On success, status is OK.
	w.WriteHeader(http.StatusOK) // 200

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "processed", // "processed" indicates successful delivery or a non-retryable FCM error.
		"message_id": messageID,
	}); err != nil {
		// This error is about writing the HTTP response, not FCM itself.
		log.Printf("PushTopicHandler: Error encoding success response: %v\n", err)
		// The header might have already been written, so we can't easily change the status code here.
		// The client will likely experience a broken response.
	}
}
