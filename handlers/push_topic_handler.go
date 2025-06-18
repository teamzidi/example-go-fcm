package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
)

// PushTopicRequest は /push/topic エンドポイントのリクエストボディ構造体です。
type PushTopicRequest struct {
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
func (h *PushTopicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("PushTopicHandler: Invalid request method: %s\n", r.Method)
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req PushTopicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("PushTopicHandler: Error decoding request body: %v\n", err)
		http.Error(w, "Invalid request body: unable to decode JSON", http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		log.Println("PushTopicHandler: Title is required")
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}
	if req.Body == "" {
		log.Println("PushTopicHandler: Body is required")
		http.Error(w, "Body is required", http.StatusBadRequest)
		return
	}
	if req.Topic == "" {
		log.Println("PushTopicHandler: Topic is required")
		http.Error(w, "Topic is required", http.StatusBadRequest)
		return
	}

	log.Printf("PushTopicHandler: Sending notification to topic '%s'. Title: '%s', Data: %v\n", req.Topic, req.Title, req.CustomData)

	messageID, err := h.fcmClient.SendToTopic(context.Background(), req.Topic, req.Title, req.Body, req.CustomData)
	if err != nil {
		log.Printf("PushTopicHandler: Error sending FCM message to topic %s: %v. Returning 503.\n", req.Topic, err)
		http.Error(w, "Failed to send notification to topic via FCM", http.StatusServiceUnavailable)
		return
	}

	log.Printf("PushTopicHandler: Successfully sent message ID %s to topic %s\n", messageID, req.Topic)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "processed",
		"message_id": messageID,
		"topic":      req.Topic,
	})
}
