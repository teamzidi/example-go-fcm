package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/teamzidi/example-go-fcm/fcm"
	"github.com/teamzidi/example-go-fcm/store" // store は直接使わないが、New関数のシグネチャを合わせるため残す
)

// PushDeviceRequest は /push/device エンドポイントのリクエストボディ構造体です。
type PushDeviceRequest struct {
	Title      string            `json:"title"`
	Body       string            `json:"body"`
	Tokens     []string          `json:"tokens"`
	CustomData map[string]string `json:"custom_data,omitempty"`
}

// PushDeviceHandler は特定のデバイストークン群へのPush通知を処理します。
type PushDeviceHandler struct {
	fcmClient   *fcm.FCMClient
	deviceStore *store.DeviceStore
}

// NewPushDeviceHandler は新しいPushDeviceHandlerのインスタンスを作成します。
func NewPushDeviceHandler(fc *fcm.FCMClient, ds *store.DeviceStore) *PushDeviceHandler {
	return &PushDeviceHandler{
		fcmClient:   fc,
		deviceStore: ds,
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

	// バリデーション
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
	if len(req.Tokens) == 0 {
		log.Println("PushDeviceHandler: Tokens list is required and cannot be empty")
		http.Error(w, "Tokens list is required and cannot be empty", http.StatusBadRequest)
		return
	}
	// FCMのSendMulticastは最大500トークンまでなので、必要に応じて分割処理を検討
	// ここではバリデーションのみ
	if len(req.Tokens) > 500 {
		log.Println("PushDeviceHandler: Number of tokens exceeds maximum (500)")
		http.Error(w, "Number of tokens exceeds maximum (500)", http.StatusBadRequest)
		return
	}


	log.Printf("PushDeviceHandler: Sending notification to %d devices. Title: '%s'\n", len(req.Tokens), req.Title)

	// FCMメッセージの作成 (SendToMultipleTokensは title, body を直接取るので、Notificationオブジェクトは不要)
	// もし custom_data を FCM の data payload に含めたい場合は、
	// fcmClient.SendToMultipleTokens を変更するか、より汎用的な SendMulticastMessage を使うインターフェースに変更する必要がある。
	// 今回は custom_data を FCM メッセージに含める部分は実装しないでおく。
	// (FCMClientInterface とその実装も custom_data をサポートするように変更が必要になるためスコープを絞る)

	br, err := h.fcmClient.SendToMultipleTokens(context.Background(), req.Tokens, req.Title, req.Body)
	if err != nil {
		log.Printf("PushDeviceHandler: Error sending FCM messages: %v. Returning 503.\n", err)
		http.Error(w, "Failed to send notifications via FCM", http.StatusServiceUnavailable)
		return
	}

	log.Printf("PushDeviceHandler: Successfully processed request. FCM BatchResponse: SuccessCount: %d, FailureCount: %d\n", br.SuccessCount, br.FailureCount)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":            "processed",
		"fcm_success_count": br.SuccessCount,
		"fcm_failure_count": br.FailureCount,
	})
}
