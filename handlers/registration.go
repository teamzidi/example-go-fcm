package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/teamzidi/example-go-fcm/store"
)

// RegistrationHandler はデバイストークンの登録を処理します。
type RegistrationHandler struct {
	deviceStore *store.DeviceStore
}

// NewRegistrationHandler は新しいRegistrationHandlerのインスタンスを作成します。
func NewRegistrationHandler(ds *store.DeviceStore) *RegistrationHandler {
	return &RegistrationHandler{deviceStore: ds}
}

// RegisterRequest はデバイストークン登録リクエストの構造体です。
type RegisterRequest struct {
	Token string `json:"token"`
}

const maxTokenLength = 4096 // デバイストークンの最大長

// ServeHTTP はHTTPリクエストを処理します。
// POST /register でデバイストークンを受け付けます。
func (h *RegistrationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: unable to decode JSON", http.StatusBadRequest)
		return
	}

	// トークンのバリデーション
	trimmedToken := strings.TrimSpace(req.Token) // 前後の空白を除去

	if trimmedToken == "" {
		http.Error(w, "Token is required", http.StatusBadRequest)
		return
	}

	if len(trimmedToken) > maxTokenLength {
		http.Error(w, "Token exceeds maximum length", http.StatusBadRequest)
		return
	}

	// storeのAddTokenを呼び出し、結果に応じてレスポンスを分岐
	added := h.deviceStore.AddToken(trimmedToken)

	if added {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "Device token registered successfully"})
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict) // 既に存在する場合は 409 Conflict
		json.NewEncoder(w).Encode(map[string]string{"message": "Device token already exists"})
	}
}
