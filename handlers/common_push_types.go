package handlers

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

// 注意: Base64デコード後の実際の業務ペイロードを表す構造体
// (例: DevicePushPayload, TopicPushPayload) は、
// それぞれのハンドラファイル (push_device_handler.go, push_topic_handler.go) 内で
// 次のステップで定義（またはリネーム）します。
// このファイルはPub/Sub Pushリクエストのエンベロープ構造のみを定義します。
