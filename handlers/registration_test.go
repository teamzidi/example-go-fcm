package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/teamzidi/example-go-fcm/store"
)

func TestRegistrationHandler_ServeHTTP(t *testing.T) {
	// deviceStore := store.NewDeviceStore() // Test毎にNewStoreするので、ここでは不要
	// handler := NewRegistrationHandler(deviceStore) // Test毎にNewHandlerするので、ここでは不要

	tests := []struct {
		name             string
		method           string
		body             interface{}
		expectedStatus   int
		expectedResponse map[string]string
		setupStore       func(ds *store.DeviceStore) // 各テスト前のストアの状態設定
		checkStore       func(t *testing.T, ds *store.DeviceStore) // テスト後のストアの状態確認
	}{
		{
			name:           "Successful registration - new token",
			method:         http.MethodPost,
			body:           RegisterRequest{Token: "new_token_123"},
			expectedStatus: http.StatusCreated,
			expectedResponse: map[string]string{"message": "Device token registered successfully"},
			checkStore: func(t *testing.T, ds *store.DeviceStore) {
				tokens := ds.GetTokens()
				if len(tokens) != 1 || tokens[0] != "new_token_123" {
					t.Errorf("Expected token 'new_token_123' to be in store, got %v", tokens)
				}
			},
		},
		{
			name:   "Token already exists",
			method: http.MethodPost,
			body:   RegisterRequest{Token: "existing_token_456"},
			setupStore: func(ds *store.DeviceStore) {
				ds.AddToken("existing_token_456") // 事前にトークンを登録
			},
			expectedStatus: http.StatusConflict,
			expectedResponse: map[string]string{"message": "Device token already exists"},
			checkStore: func(t *testing.T, ds *store.DeviceStore) {
				tokens := ds.GetTokens()
				if len(tokens) != 1 || tokens[0] != "existing_token_456" {
					// 状態が変わっていないことを確認
					t.Errorf("Expected only token 'existing_token_456' to be in store, got %v", tokens)
				}
			},
		},
		{
			name:           "Token is empty string",
			method:         http.MethodPost,
			body:           RegisterRequest{Token: ""},
			expectedStatus: http.StatusBadRequest,
			// expectedResponse is not checked for plain text error
		},
		{
			name:           "Token is only whitespace",
			method:         http.MethodPost,
			body:           RegisterRequest{Token: "   "},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Token exceeds maximum length",
			method:         http.MethodPost,
			body:           RegisterRequest{Token: strings.Repeat("a", maxTokenLength+1)},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid HTTP method (GET)",
			method:         http.MethodGet,
			body:           RegisterRequest{Token: "any_token"}, // body for GET is not typical but for test consistency
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Invalid request body (not JSON)",
			method:         http.MethodPost,
			body:           "this is not json", // 文字列を直接渡す
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid request body (JSON, but wrong structure - results in empty token)",
			method:         http.MethodPost,
			body:           map[string]string{"wrong_field": "some_value"},
			expectedStatus: http.StatusBadRequest, // Token is required になる (req.Token will be empty)
		},
		{
			name:           "Successful registration - token with leading/trailing spaces",
			method:         http.MethodPost,
			body:           RegisterRequest{Token: "  spaced_token  "},
			expectedStatus: http.StatusCreated,
			expectedResponse: map[string]string{"message": "Device token registered successfully"},
			checkStore: func(t *testing.T, ds *store.DeviceStore) {
				tokens := ds.GetTokens()
				if len(tokens) != 1 || tokens[0] != "spaced_token" { // Trimmed token
					t.Errorf("Expected token 'spaced_token' to be in store, got %v", tokens)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 各テストの前にストアをリセット（または新しいストアを作成）
			currentDeviceStore := store.NewDeviceStore()
			currentHandler := NewRegistrationHandler(currentDeviceStore)
			if tt.setupStore != nil {
				tt.setupStore(currentDeviceStore)
			}

			var reqBodyReader *bytes.Buffer
			if tt.body != nil {
				var reqBodyBytes []byte
				var err error
				if strBody, ok := tt.body.(string); ok {
					reqBodyBytes = []byte(strBody)
				} else {
					reqBodyBytes, err = json.Marshal(tt.body)
					if err != nil {
						t.Fatalf("Failed to marshal request body: %v", err)
					}
				}
				reqBodyReader = bytes.NewBuffer(reqBodyBytes)
			} else {
				reqBodyReader = bytes.NewBuffer([]byte{}) // Empty body for GET or other cases
			}


			req := httptest.NewRequest(tt.method, "/register", reqBodyReader)
			rr := httptest.NewRecorder()

			currentHandler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("ServeHTTP status = %v, want %v. Response body: %s", rr.Code, tt.expectedStatus, rr.Body.String())
			}

			if tt.expectedResponse != nil {
				var respBody map[string]string
				if err := json.Unmarshal(rr.Body.Bytes(), &respBody); err != nil {
					// 期待するレスポンスが定義されている場合はJSONであることを期待する
					// ただし、エラーケースではプレーンテキストの場合もあるので、厳密には期待ステータスコードで分岐が必要
					// ここでは、expectedResponse がある場合は JSON デコード可能であることを前提とする
					t.Fatalf("Failed to unmarshal response body: %v. Body: %s", err, rr.Body.String())

				}
				if !reflect.DeepEqual(respBody, tt.expectedResponse) {
					t.Errorf("ServeHTTP response body = %v, want %v", respBody, tt.expectedResponse)
				}
			}

			// ストアの状態をチェック
			if tt.checkStore != nil {
				tt.checkStore(t, currentDeviceStore)
			}
		})
	}
}

// reflect パッケージをインポートリストに追加
import "reflect"
