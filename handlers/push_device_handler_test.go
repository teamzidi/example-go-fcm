package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	// "reflect" // reflect.DeepEqual を使う場合は必要

	"firebase.google.com/go/v4/messaging"
	"github.com/teamzidi/example-go-fcm/fcm" // fcm パッケージをインポート
	"github.com/teamzidi/example-go-fcm/store"
)

func TestPushDeviceHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name                 string
		requestBody          PushDeviceRequest
		setupMock            func(mockFCM *fcm.FCMClient) // モックの設定用関数
		expectedStatus       int
		// 検証用フィールド (テストケース内で直接検証するため、構造体からは削除してもよい)
		// expectedFCMCallCount int
		// expectedTokensSent   []string
		// expectedTitleSent    string
		// expectedBodySent     string
	}{
		{
			name: "Successful push to devices",
			requestBody: PushDeviceRequest{
				Title:  "Device Test Title",
				Body:   "Device Test Body",
				Tokens: []string{"dev_token1", "dev_token2"},
			},
			setupMock: func(mockFCM *fcm.FCMClient) {
				mockFCM.MockSendToMultipleTokens = func(ctx context.Context, tokens []string, title string, body string) (*messaging.BatchResponse, error) {
					// 引数検証 (テスト内で直接行う方が柔軟性が高い場合もある)
					if title != "Device Test Title" {
						t.Errorf("Mock: Title mismatch. Got %s, Want %s", title, "Device Test Title")
					}
					if body != "Device Test Body" {
						t.Errorf("Mock: Body mismatch. Got %s, Want %s", body, "Device Test Body")
					}
					// reflect.DeepEqual(tokens, []string{"dev_token1", "dev_token2"}) // 必要なら
					return &messaging.BatchResponse{SuccessCount: len(tokens), FailureCount: 0}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "FCM client returns error on device push",
			requestBody: PushDeviceRequest{
				Title:  "Device Error Title",
				Body:   "Device Error Body",
				Tokens: []string{"dev_token_err"},
			},
			setupMock: func(mockFCM *fcm.FCMClient) {
				mockFCM.MockSendToMultipleTokens = func(ctx context.Context, tokens []string, title string, body string) (*messaging.BatchResponse, error) {
					return nil, errors.New("FCM send to devices failed")
				}
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:        "Missing title",
			requestBody: PushDeviceRequest{Body: "Body only", Tokens: []string{"t1"}},
			setupMock:   func(mockFCM *fcm.FCMClient) { /* No FCM call expected */ },
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Missing body",
			requestBody: PushDeviceRequest{Title: "Title only", Tokens: []string{"t1"}},
			setupMock:   func(mockFCM *fcm.FCMClient) { /* No FCM call expected */ },
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Missing tokens",
			requestBody: PushDeviceRequest{Title: "Title", Body: "Body", Tokens: []string{}},
			setupMock:   func(mockFCM *fcm.FCMClient) { /* No FCM call expected */ },
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Too many tokens",
			requestBody: PushDeviceRequest{Title: "Title", Body: "Body", Tokens: make([]string, 501)},
			setupMock:   func(mockFCM *fcm.FCMClient) { /* No FCM call expected */ },
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deviceStore := store.NewDeviceStore()
			// テスト実行時は test_fcm_mock タグによりモック版の NewFCMClient が呼ばれる
			mockFCMClient, err := fcm.NewFCMClient(context.Background())
			if err != nil {
				t.Fatalf("Failed to create mock FCMClient: %v", err)
			}

			// モックの挙動を設定
			originalMockSendToMultipleTokens := mockFCMClient.MockSendToMultipleTokens // 元のデフォルトモックを保持
			tt.setupMock(mockFCMClient)

			handler := NewPushDeviceHandler(mockFCMClient, deviceStore)

			reqBodyBytes, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/pubsub/push/device", bytes.NewBuffer(reqBodyBytes))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Status = %v, want %v. Body: %s", rr.Code, tt.expectedStatus, rr.Body.String())
			}

			// モック関数を元に戻す (他のテストケースに影響を与えないため)
			mockFCMClient.MockSendToMultipleTokens = originalMockSendToMultipleTokens
		})
	}
}

// Test for invalid HTTP method (内容は変更なし)
func TestPushDeviceHandler_ServeHTTP_InvalidMethod(t *testing.T) {
	deviceStore := store.NewDeviceStore()
	// モッククライアントは呼ばれないはずなので、デフォルトのままでよい
	mockFCMClient, _ := fcm.NewFCMClient(context.Background())
	handler := NewPushDeviceHandler(mockFCMClient, deviceStore)

	req := httptest.NewRequest(http.MethodGet, "/pubsub/push/device", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %v for GET, got %v", http.StatusMethodNotAllowed, rr.Code)
	}
}

// Test for invalid JSON body (内容は変更なし)
func TestPushDeviceHandler_ServeHTTP_InvalidJSON(t *testing.T) {
	deviceStore := store.NewDeviceStore()
	mockFCMClient, _ := fcm.NewFCMClient(context.Background())
	handler := NewPushDeviceHandler(mockFCMClient, deviceStore)

	req := httptest.NewRequest(http.MethodPost, "/pubsub/push/device", strings.NewReader("this is not json"))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %v for invalid JSON, got %v", http.StatusBadRequest, rr.Code)
	}
}
