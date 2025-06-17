package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"

	"github.com/teamzidi/example-go-fcm/fcm"
	"github.com/teamzidi/example-go-fcm/store"
)

// PubSubPushMessage はPub/SubからのPushリクエストのメッセージ部分の構造体です。
type PubSubPushMessage struct {
	Data        string `json:"data"` // Base64エンコードされたペイロード
	MessageID   string `json:"messageId"`
	PublishTime string `json:"publishTime"`
}

// PubSubPushRequest はPub/SubからのPushリクエスト全体の構造体です。
type PubSubPushRequest struct {
	Message      PubSubPushMessage `json:"message"`
	Subscription string            `json:"subscription"`
}

// ActualMessagePayload は message.data をデコードした後の実際のペイロードです。
type ActualMessagePayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// PushHandler はPub/SubからのPush通知を処理します。
type PushHandler struct {
	fcmClient   fcm.FCMClientInterface // モックしやすいようにインターフェース型に変更
	deviceStore *store.DeviceStore
}

// NewPushHandler は新しいPushHandlerのインスタンスを作成します。
// fcm.FCMClientInterface を引数に取るように変更
func NewPushHandler(fc fcm.FCMClientInterface, ds *store.DeviceStore) *PushHandler {
	return &PushHandler{
		fcmClient:   fc,
		deviceStore: ds,
	}
}

// ServeHTTP はHTTPリクエストを処理します。
// Pub/SubからのPush通知はPOSTリクエストでこのエンドポイントに送信されます。
func (h *PushHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("PushHandler: Invalid request method: %s\n", r.Method)
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req PubSubPushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("PushHandler: Error decoding Pub/Sub push request: %v\n", err)
		http.Error(w, "Invalid request body format", http.StatusBadRequest)
		return
	}

	log.Printf("PushHandler: Received message ID %s from subscription %s published at %s\n", req.Message.MessageID, req.Subscription, req.Message.PublishTime)

	if req.Message.Data == "" {
		log.Println("PushHandler: Message data is empty. Acking by returning 200.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "acknowledged", "reason": "empty message data"})
		return
	}

	decodedData, err := base64.StdEncoding.DecodeString(req.Message.Data)
	if err != nil {
		log.Printf("PushHandler: Error decoding message data (base64): %v\n", err)
		http.Error(w, "Invalid message data encoding", http.StatusBadRequest)
		return
	}

	var payload ActualMessagePayload
	if err := json.Unmarshal(decodedData, &payload); err != nil {
		log.Printf("PushHandler: Error unmarshalling actual message payload: %v. Payload was: %s\n", err, string(decodedData))
		http.Error(w, "Invalid actual message payload format", http.StatusBadRequest)
		return
	}

	if payload.Title == "" || payload.Body == "" {
		log.Println("PushHandler: Actual message title or body is empty. Acking by returning 200.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "acknowledged", "reason": "empty title or body"})
		return
	}

	tokens := h.deviceStore.GetTokens()
	if len(tokens) == 0 {
		log.Println("PushHandler: No registered devices to send notification. Acking by returning 200.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "acknowledged", "reason": "no registered devices"})
		return
	}

	log.Printf("PushHandler: Sending notification to %d devices. Title: '%s', Body: '%s'\n", len(tokens), payload.Title, payload.Body)

	br, err := h.fcmClient.SendToMultipleTokens(context.Background(), tokens, payload.Title, payload.Body)
	if err != nil {
		// FCMへの送信に失敗した場合、エラーをログに出力し、503 Service Unavailable を返す
		log.Printf("PushHandler: Error sending FCM messages: %v. Returning 503 to trigger Pub/Sub retry.\n", err)
		// br が nil の可能性もあるので注意
		// var successCount, failureCount int // Errorの場合brがnilの可能性があるので、ここでは使わない
		// if br != nil {
		// 	successCount = br.SuccessCount
		// 	failureCount = br.FailureCount
		// }
		http.Error(w, "Failed to send notifications via FCM", http.StatusServiceUnavailable)
		// エラーレスポンスボディに詳細を含めることも検討できるが、Pub/Subの再試行にはステータスコードが重要
		// json.NewEncoder(w).Encode(map[string]interface{}{"status": "error", "reason": "FCM send failure", "success_count": successCount, "failure_count": failureCount, "error_details": err.Error()})
		return
	}

	log.Printf("PushHandler: Successfully processed message ID %s. FCM BatchResponse: SuccessCount: %d, FailureCount: %d\n", req.Message.MessageID, br.SuccessCount, br.FailureCount)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "processed", "fcm_success_count": br.SuccessCount, "fcm_failure_count": br.FailureCount})
}
