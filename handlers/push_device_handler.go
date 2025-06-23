package handlers

import (
	"context"
	"encoding/base64" // Base64デコードのためにインポート
	"encoding/json"
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
		log.Printf("PushDeviceHandler: Invalid request method: %s\n", r.Method)
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed) // Reverted to original behavior
		return
	}

	var pubSubReq PubSubPushRequest
	if err := json.NewDecoder(r.Body).Decode(&pubSubReq); err != nil {
		log.Printf("PushDeviceHandler: Error decoding Pub/Sub envelope: %v\n", err)
		w.WriteHeader(http.StatusNoContent) // New: Ack with 204
		return // bye
	}

	log.Printf("PushDeviceHandler: Received Pub/Sub message ID %s from subscription %s published at %s\n",
		pubSubReq.Message.MessageID, pubSubReq.Subscription, pubSubReq.Message.PublishTime)

	if pubSubReq.Message.Data == "" {
		log.Println("PushDeviceHandler: Pub/Sub message data is empty. Acking.")
		w.WriteHeader(http.StatusNoContent) // New: Ack with 204
		return // bye
	}

	decodedData, err := base64.StdEncoding.DecodeString(pubSubReq.Message.Data)
	if err != nil {
		log.Printf("PushDeviceHandler: Error decoding base64 data: %v\n", err)
		w.WriteHeader(http.StatusNoContent) // New: Ack with 204
		return // bye
	}

	var payload DevicePushPayload
	if err := json.Unmarshal(decodedData, &payload); err != nil {
		log.Printf("PushDeviceHandler: Error unmarshalling actual payload: %v. Decoded data was: %s\n", err, string(decodedData))
		w.WriteHeader(http.StatusNoContent) // New: Ack with 204
		return // bye
	}

	if payload.Title == "" {
		log.Println("PushDeviceHandler: Title is required in payload")
		w.WriteHeader(http.StatusNoContent) // New: Ack with 204
		return // bye
	}

	if payload.Body == "" {
		log.Println("PushDeviceHandler: Body is required in payload")
		w.WriteHeader(http.StatusNoContent) // New: Ack with 204
		return // bye
	}

	if payload.Token == "" {
		log.Println("PushDeviceHandler: Token is required in payload")
		w.WriteHeader(http.StatusNoContent) // New: Ack with 204
		return // bye
	}

	log.Printf("PushDeviceHandler: Sending notification to device token '%s'. Title: '%s', Data: %v\n",
		payload.Token, payload.Title, payload.CustomData)

	// FCM送信
	messageID, err := h.fcmClient.SendToToken(context.Background(), payload.Token, payload.Title, payload.Body, payload.CustomData)
	if err != nil {
		log.Printf("PushDeviceHandler: Error sending FCM message to token %s: %v.\n", payload.Token, err) // Log error
		if fcm.IsRetryableError(err) {
			http.Error(w, "Failed to send notification via FCM (retryable)", http.StatusInternalServerError) // 500
		} else {
			// Non-retryable errors are treated as "processed" from the perspective of the pub/sub queue,
			// so we return a 204 No Content to acknowledge the message without encouraging retries of this specific message.
			// The client application might need to handle this case (e.g. by removing an invalid token).
			log.Printf("PushDeviceHandler: Non-retryable error for token %s. Acknowledging message with 204 No Content.\n", payload.Token)
			w.WriteHeader(http.StatusNoContent) // 204
		}
		return
	}

	log.Printf("PushDeviceHandler: Successfully sent message ID %s to token %s\n", messageID, payload.Token)

	w.Header().Set("Content-Type", "application/json")
	// On success, status is OK.
	w.WriteHeader(http.StatusOK) // 200

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "processed", // "processed" indicates successful delivery or a non-retryable FCM error.
		"message_id": messageID,
	}); err != nil {
		// This error is about writing the HTTP response, not FCM itself.
		log.Printf("PushDeviceHandler: Error encoding success response: %v\n", err)
		// The header might have already been written, so we can't easily change the status code here.
		// The client will likely experience a broken response.
	}
}
