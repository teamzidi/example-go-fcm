package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/teamzidi/example-go-fcm/fcm"
	// "github.com/teamzidi/example-go-fcm/store" // storeパッケージはもう使わない
)

// PushDeviceRequest は /pubsub/push/device エンドポイントのリクエストボディ構造体です。
type PushDeviceRequest struct {
	Title      string            `json:"title"`
	Body       string            `json:"body"`
	Token      string            `json:"token"`
	CustomData map[string]string `json:"custom_data,omitempty"`
}

// PushDeviceHandler は特定の単一デバイストークンへのPush通知を処理します。
type PushDeviceHandler struct {
	fcmClient *fcmHandlerClient
	// deviceStore *store.DeviceStore // 削除
}

// NewPushDeviceHandler は新しいPushDeviceHandlerのインスタンスを作成します。
func NewPushDeviceHandler(fc *fcmHandlerClient /* ds *store.DeviceStore // 削除 */) *PushDeviceHandler {
	return &PushDeviceHandler{
		fcmClient: fc,
		// deviceStore: ds, // 削除
	}
}

// ServeHTTP はHTTPリクエストを処理します。
func (h *PushDeviceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("PushDeviceHandler: Invalid request method: %s\n", r.Method)
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req PushDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("PushDeviceHandler: Error decoding request body: %v\n", err)
		http.Error(w, "Invalid request body: unable to decode JSON", http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		log.Println("PushDeviceHandler: Title is required")
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}
	if req.Body == "" {
		log.Println("PushDeviceHandler: Body is required")
		http.Error(w, "Body is required", http.StatusBadRequest)
		return
	}
	if req.Token == "" {
		log.Println("PushDeviceHandler: Token is required")
		http.Error(w, "Token is required", http.StatusBadRequest)
		return
	}

	log.Printf("PushDeviceHandler: Sending notification to device token '%s'. Title: '%s', Data: %v\n", req.Token, req.Title, req.CustomData)

	messageID, err := h.fcmClient.SendToToken(context.Background(), req.Token, req.Title, req.Body, req.CustomData)
	if err != nil {
		log.Printf("PushDeviceHandler: Error sending FCM message to token %s: %v. Returning 503.\n", req.Token, err)
		http.Error(w, "Failed to send notification via FCM", http.StatusServiceUnavailable)
		return
	}

	log.Printf("PushDeviceHandler: Successfully sent message ID %s to token %s\n", messageID, req.Token)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "processed",
		"message_id": messageID,
		"token":      req.Token,
	})
}
