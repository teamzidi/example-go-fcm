package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
)

// PubSubInternalMessage はPub/SubからのPushリクエストのメッセージ部分の内部構造体です。
// PubSubPushRequest の Message フィールドとして使用されます。
type PubSubInternalMessage struct {
	Data        string `json:"data"` // Base64エンコードされた実際の業務ペイロード
	MessageID   string `json:"messageId"`
	PublishTime string `json:"publishTime"`
	// Attributes map[string]string `json:"attributes,omitempty"` // 必要であれば属性も
}

// PubSubPushRequest はPub/SubからのPushリクエスト全体の構造体です。
// これがHTTPリクエストボディの最上位のJSONオブジェクトに対応します。
type PubSubPushRequest struct {
	Message      PubSubInternalMessage `json:"message"`
	Subscription string                `json:"subscription"`
}

func decodeData(body io.Reader) ([]byte, error) {
	var pubSubReq PubSubPushRequest
	if err := json.NewDecoder(body).Decode(&pubSubReq); err != nil {
		return nil, fmt.Errorf("decoding Pub/Sub envelope: %v", err)
	}

	log.Printf("PushDeviceHandler: Received Pub/Sub message ID %s from subscription %s published at %s",
		pubSubReq.Message.MessageID, pubSubReq.Subscription, pubSubReq.Message.PublishTime)

	if pubSubReq.Message.Data == "" {
		return nil, fmt.Errorf("Pub/Sub message data is empty")
	}

	decodedData, err := base64.StdEncoding.DecodeString(pubSubReq.Message.Data)
	if err != nil {
		return nil, fmt.Errorf("decoding base64 data: %w", err)
	}

	return decodedData, nil
}
