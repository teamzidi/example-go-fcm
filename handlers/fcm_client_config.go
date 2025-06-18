//go:build !test_fcm_mock

package handlers

import (
	"context"
	realfcm "github.com/teamzidi/example-go-fcm/fcm"
)

// fcmHandlerClient は、このハンドラパッケージ内でFCMクライアントを参照するための型です。
// 通常ビルド時は、実際のfcm.Clientへのエイリアスとなります。
type fcmHandlerClient = realfcm.Client

// NewFcmHandlerClient は、FCMクライアントのインスタンスを生成します。
// 通常ビルド時は、実際のfcm.Clientを生成します。
func NewFcmHandlerClient(ctx context.Context) (*fcmHandlerClient, error) {
	client, err := realfcm.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return client, nil
}
