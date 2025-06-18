package handlers

import (
	"context"
	"encoding/base64" // Base64デコードのためにインポート
	"encoding/json"
	"log"
	"net/http"
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
	fcmClient *fcmHandlerClient
}

// NewPushDeviceHandler は新しいPushDeviceHandlerのインスタンスを作成します。
func NewPushDeviceHandler(fc *fcmHandlerClient) *PushDeviceHandler {
	return &PushDeviceHandler{
		fcmClient: fc,
	}
}

// ServeHTTP はHTTPリクエストを処理します。
// Pub/SubからのPushリクエストを想定しています。
func (h *PushDeviceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("PushDeviceHandler: Invalid request method: %s\n", r.Method)
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// 1. Pub/Sub Pushリクエストのエンベロープをデコード
	var pubSubReq PubSubPushRequest // common_push_types.go で定義
	if err := json.NewDecoder(r.Body).Decode(&pubSubReq); err != nil {
		log.Printf("PushDeviceHandler: Error decoding Pub/Sub envelope: %v\n", err)
		http.Error(w, "Invalid Pub/Sub message format", http.StatusBadRequest)
		return
	}

	log.Printf("PushDeviceHandler: Received Pub/Sub message ID %s from subscription %s published at %s\n",
		pubSubReq.Message.MessageID, pubSubReq.Subscription, pubSubReq.Message.PublishTime)

	if pubSubReq.Message.Data == "" {
		log.Println("PushDeviceHandler: Pub/Sub message data is empty. Acking.")
		// 何も処理せず200 OKを返すことでメッセージがAckされる
		w.Header().Set("Content-Type", "application/json") // レスポンスタイプを設定
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "acknowledged", "reason": "empty Pub/Sub message data"})
		return
	}

	// 2. Message.Data (Base64エンコードされた文字列) をデコード
	decodedData, err := base64.StdEncoding.DecodeString(pubSubReq.Message.Data)
	if err != nil {
		log.Printf("PushDeviceHandler: Error decoding base64 data: %v\n", err)
		http.Error(w, "Invalid base64 data in Pub/Sub message", http.StatusBadRequest)
		return
	}

	// 3. デコードされたJSONを実際の業務ペイロード (DevicePushPayload) にアンマーシャル
	var actualPayload DevicePushPayload
	if err := json.Unmarshal(decodedData, &actualPayload); err != nil {
		log.Printf("PushDeviceHandler: Error unmarshalling actual payload: %v. Decoded data was: %s\n", err, string(decodedData))
		http.Error(w, "Invalid actual payload format in Pub/Sub message data", http.StatusBadRequest)
		return
	}

	// 4. 業務ペイロードのバリデーション
	if actualPayload.Title == "" {
		log.Println("PushDeviceHandler: Title is required in actual payload")
		http.Error(w, "Title is required in actual payload", http.StatusBadRequest)
		return
	}
	if actualPayload.Body == "" {
		log.Println("PushDeviceHandler: Body is required in actual payload")
		http.Error(w, "Body is required in actual payload", http.StatusBadRequest)
		return
	}
	if actualPayload.Token == "" {
		log.Println("PushDeviceHandler: Token is required in actual payload")
		http.Error(w, "Token is required in actual payload", http.StatusBadRequest)
		return
	}

	log.Printf("PushDeviceHandler: Sending notification to device token '%s'. Title: '%s', Data: %v\n",
		actualPayload.Token, actualPayload.Title, actualPayload.CustomData)

	// 5. FCM送信
	messageID, err := h.fcmClient.SendToToken(context.Background(), actualPayload.Token, actualPayload.Title, actualPayload.Body, actualPayload.CustomData)
	if err != nil {
		log.Printf("PushDeviceHandler: Error sending FCM message to token %s: %v. Returning 503.\n", actualPayload.Token, err)
		http.Error(w, "Failed to send notification via FCM", http.StatusServiceUnavailable)
		return
	}

	log.Printf("PushDeviceHandler: Successfully sent message ID %s to token %s\n", messageID, actualPayload.Token)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "processed",
		"message_id": messageID,
		"token":      actualPayload.Token,
	})
}
