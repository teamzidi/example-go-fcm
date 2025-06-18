package handlers

import (
	"context"
	"encoding/base64" // Base64デコードのためにインポート
	"encoding/json"
	"log"
	"net/http"
)

// TopicPushPayload は /pubsub/push/topic エンドポイントでPub/Subメッセージの
// Base64デコードされた data フィールドが示す実際の業務ペイロード構造体です。
type TopicPushPayload struct {
	Title      string            `json:"title"`
	Body       string            `json:"body"`
	Topic      string            `json:"topic"`
	CustomData map[string]string `json:"custom_data,omitempty"`
}

// PushTopicHandler は特定のFCMトピックへのPush通知を処理します。
type PushTopicHandler struct {
	fcmClient *fcmHandlerClient
}

// NewPushTopicHandler は新しいPushTopicHandlerのインスタンスを作成します。
func NewPushTopicHandler(fc *fcmHandlerClient) *PushTopicHandler {
	return &PushTopicHandler{
		fcmClient: fc,
	}
}

// ServeHTTP はHTTPリクエストを処理します。
// Pub/SubからのPushリクエストを想定しています。
func (h *PushTopicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("PushTopicHandler: Invalid request method: %s\n", r.Method)
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// 1. Pub/Sub Pushリクエストのエンベロープをデコード
	var pubSubReq PubSubPushRequest // common_push_types.go で定義
	if err := json.NewDecoder(r.Body).Decode(&pubSubReq); err != nil {
		log.Printf("PushTopicHandler: Error decoding Pub/Sub envelope: %v\n", err)
		http.Error(w, "Invalid Pub/Sub message format", http.StatusBadRequest)
		return
	}

	log.Printf("PushTopicHandler: Received Pub/Sub message ID %s from subscription %s published at %s\n",
		pubSubReq.Message.MessageID, pubSubReq.Subscription, pubSubReq.Message.PublishTime)

	if pubSubReq.Message.Data == "" {
		log.Println("PushTopicHandler: Pub/Sub message data is empty. Acking.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "acknowledged", "reason": "empty Pub/Sub message data"})
		return
	}

	// 2. Message.Data (Base64エンコードされた文字列) をデコード
	decodedData, err := base64.StdEncoding.DecodeString(pubSubReq.Message.Data)
	if err != nil {
		log.Printf("PushTopicHandler: Error decoding base64 data: %v\n", err)
		http.Error(w, "Invalid base64 data in Pub/Sub message", http.StatusBadRequest)
		return
	}

	// 3. デコードされたJSONを実際の業務ペイロード (TopicPushPayload) にアンマーシャル
	var actualPayload TopicPushPayload
	if err := json.Unmarshal(decodedData, &actualPayload); err != nil {
		log.Printf("PushTopicHandler: Error unmarshalling actual payload: %v. Decoded data was: %s\n", err, string(decodedData))
		http.Error(w, "Invalid actual payload format in Pub/Sub message data", http.StatusBadRequest)
		return
	}

	// 4. 業務ペイロードのバリデーション
	if actualPayload.Title == "" {
		log.Println("PushTopicHandler: Title is required in actual payload")
		http.Error(w, "Title is required in actual payload", http.StatusBadRequest)
		return
	}
	if actualPayload.Body == "" {
		log.Println("PushTopicHandler: Body is required in actual payload")
		http.Error(w, "Body is required in actual payload", http.StatusBadRequest)
		return
	}
	if actualPayload.Topic == "" {
		log.Println("PushTopicHandler: Topic is required in actual payload")
		http.Error(w, "Topic is required in actual payload", http.StatusBadRequest)
		return
	}

	log.Printf("PushTopicHandler: Sending notification to topic '%s'. Title: '%s', Data: %v\n",
		actualPayload.Topic, actualPayload.Title, actualPayload.CustomData)

	// 5. FCM送信
	messageID, err := h.fcmClient.SendToTopic(context.Background(), actualPayload.Topic, actualPayload.Title, actualPayload.Body, actualPayload.CustomData)
	if err != nil {
		log.Printf("PushTopicHandler: Error sending FCM message to topic %s: %v. Returning 503.\n", actualPayload.Topic, err)
		http.Error(w, "Failed to send notification to topic via FCM", http.StatusServiceUnavailable)
		return
	}

	log.Printf("PushTopicHandler: Successfully sent message ID %s to topic %s\n", messageID, actualPayload.Topic)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "processed",
		"message_id": messageID,
		"topic":     actualPayload.Topic,
	})
}
