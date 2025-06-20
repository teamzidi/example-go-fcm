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
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var pubSubReq PubSubPushRequest
	if err := json.NewDecoder(r.Body).Decode(&pubSubReq); err != nil {
		log.Printf("PushDeviceHandler: Error decoding Pub/Sub envelope: %v\n", err)
		return // bye
	}

	log.Printf("PushDeviceHandler: Received Pub/Sub message ID %s from subscription %s published at %s\n",
		pubSubReq.Message.MessageID, pubSubReq.Subscription, pubSubReq.Message.PublishTime)

	if pubSubReq.Message.Data == "" {
		log.Println("PushDeviceHandler: Pub/Sub message data is empty. Acking.")
		return // bye
	}

	decodedData, err := base64.StdEncoding.DecodeString(pubSubReq.Message.Data)
	if err != nil {
		log.Printf("PushDeviceHandler: Error decoding base64 data: %v\n", err)
		return // bye
	}

	var payload DevicePushPayload
	if err := json.Unmarshal(decodedData, &payload); err != nil {
		log.Printf("PushDeviceHandler: Error unmarshalling actual payload: %v. Decoded data was: %s\n", err, string(decodedData))
		return // bye
	}

	if payload.Title == "" {
		log.Println("PushDeviceHandler: Title is required in payload")
		return // bye
	}

	if payload.Body == "" {
		log.Println("PushDeviceHandler: Body is required in payload")
		return // bye
	}

	if payload.Token == "" {
		log.Println("PushDeviceHandler: Token is required in payload")
		return // bye
	}

	log.Printf("PushDeviceHandler: Sending notification to device token '%s'. Title: '%s', Data: %v\n",
		payload.Token, payload.Title, payload.CustomData)

	// FCM送信
	messageID, err := h.fcmClient.SendToToken(context.Background(), payload.Token, payload.Title, payload.Body, payload.CustomData)
	if err != nil {
		log.Printf("PushDeviceHandler: Error sending FCM message to token %s: %v. Returning 503.\n", payload.Token, err)
		http.Error(w, "Failed to send notification via FCM", http.StatusServiceUnavailable)
		return
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "processed",
		"message_id": messageID,
	}); err != nil {
		log.Printf("PushDeviceHandler: Error encoding response: %v\n", err)
	}

	log.Printf("PushDeviceHandler: Successfully sent message ID %s to token %s\n", messageID, payload.Token)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
