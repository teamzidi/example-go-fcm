package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"firebase.google.com/go/v4/messaging" // messaging.Message を使うため
	"github.com/teamzidi/example-go-fcm/fcm"
	"github.com/teamzidi/example-go-fcm/store"
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
	fcmClient   *fcm.FCMClient
	deviceStore *store.DeviceStore
}

// NewPushTopicHandler は新しいPushTopicHandlerのインスタンスを作成します。
func NewPushTopicHandler(fc *fcm.FCMClient, ds *store.DeviceStore) *PushTopicHandler {
	return &PushTopicHandler{
		fcmClient:   fc,
		deviceStore: ds,
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

	// バリデーション
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
	// トピック名のバリデーション (例: `/topics/` プレフィックスはFCMが自動処理することが多いので、ここでは必須としない)
	// 正規表現: [a-zA-Z0-9-_.~%]+
	// ここでは簡単な空チェックのみ。より厳密なチェックも可能。


	log.Printf("PushTopicHandler: Sending notification to topic '%s'. Title: '%s'\n", req.Topic, req.Title)

	// FCMメッセージの作成
	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: req.Title,
			Body:  req.Body,
		},
		Data:  req.CustomData, // custom_data を FCM の data payload に設定
		Topic: req.Topic,
	}

	// fcmClientインターフェースに汎用的なSendメソッドがあると仮定する。
	// もし SendToTopicのような特化メソッドがインターフェースにあればそれを使う。
	// ここでは、FCMClientInterface に Send(ctx, message) があると想定し、
	// その Send が内部で message.Topic を見て適切に処理すると期待する。
	// そのためには、fcm.FCMClient の SendToToken を Send にリネームまたはラップする必要があるかもしれない。
	// → FCMClientInterface には SendToToken と SendToMultipleTokens がある。
	//   トピック送信のためには、messaging.Message を受け取る Send メソッドをインターフェースに追加するか、
	//   SendToTopicのようなメソッドを新設する必要がある。
	//   今回は、fcm.goのFCMClientに Send(ctx, *messaging.Message)string, error を追加し、
	//   それをインターフェースにも追加する、という変更を後続のfcm/fcm.goの調整ステップで行うこととする。
	//   ここでは、そのメソッドが利用可能であると仮定して進める。
	//   ※ fcm.FCMClient.Send(ctx, message) を呼び出す形を想定。

	// 仮に、FCMClientInterface に Send(ctx, *messaging.Message) (string, error) が追加されたと想定
	// messageID, err := h.fcmClient.Send(context.Background(), message) // このような呼び出しをしたい

	// 現在のインターフェース *fcm.FCMClient には汎用的な Send(*messaging.Message) がない。
	// SendToToken はトークン専用、SendToMultipleTokensもトークン専用。
	// トピック送信のためには fcm.go と *fcm.FCMClient の変更が必要。
	// このサブタスクでは、その変更が後ほど行われることを前提として、ロジックの骨子を記述する。
	// **実際にはこのままではコンパイルエラーになるため、次のfcm.go調整ステップで解決する。**
	// ここでは仮の成功として進める。

	// --- ここから仮実装 ---
	// 本来は fcmClient.Send(ctx, message) を呼びたい
	// log.Printf("PushTopicHandler: (仮) FCM Send to topic %s would be called here.\n", req.Topic)
	// --- ここまで仮実装 ---

	// FCMClientInterface.Send を呼び出す (次のステップでインターフェースと実装を更新)
	messageID, err := h.fcmClient.Send(context.Background(), message) // *fcm.FCMClient に Send メソッドを追加する必要あり
	if err != nil {
		log.Printf("PushTopicHandler: Error sending FCM message to topic %s: %v. Returning 503.\n", req.Topic, err)
		http.Error(w, "Failed to send notification to topic via FCM", http.StatusServiceUnavailable)
		return
	}


	log.Printf("PushTopicHandler: Successfully sent message ID %s to topic %s\n", messageID, req.Topic)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "processed",
		"message_id": messageID,
		"topic":     req.Topic,
	})
}
