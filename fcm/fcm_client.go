package fcm

import (
	"context"
	"errors"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
)

// Client はFirebase Cloud Messagingのクライアントです。(旧 FCMClient)
type Client struct {
	msg *messaging.Client
}

// NewClient は新しいClientのインスタンスを作成します。(旧 NewFCMClient)
// 環境変数 GOOGLE_APPLICATION_CREDENTIALS が設定されている必要があります。
func NewClient(ctx context.Context) (*Client, error) {
	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		log.Printf("Error initializing Firebase app: %v\n", err)
		return nil, err
	}
	msgClient, err := app.Messaging(ctx)
	if err != nil {
		log.Printf("Error getting Messaging client: %v\n", err)
		return nil, err
	}
	return &Client{msg: msgClient}, nil
}

// SendToToken は指定された単一のデバイストークンに通知とデータペイロードを送信します。
func (c *Client) SendToToken(ctx context.Context, token string, title string, body string, customData map[string]string) (string, error) {
	if token == "" {
		log.Println("Client: Token is empty in SendToToken.")
		return "", errors.New("FCM token cannot be empty")
	}
	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data:  customData,
		Token: token,
	}

	response, err := c.msg.Send(ctx, message)
	if err != nil {
		log.Printf("Client: Error sending message to token %s: %v\n", token, err)
		return "", err
	}
	log.Printf("Client: Successfully sent message to token %s: %s\n", token, response)
	return response, nil
}

// SendToTopic は指定されたFCMトピックに通知とデータペイロードを送信します。
func (c *Client) SendToTopic(ctx context.Context, topic string, title string, body string, customData map[string]string) (string, error) {
	if topic == "" {
		log.Println("Client: Topic is empty in SendToTopic.")
		return "", errors.New("FCM topic cannot be empty")
	}
	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data:  customData,
		Topic: topic,
	}

	response, err := c.msg.Send(ctx, message)
	if err != nil {
		log.Printf("Client: Error sending message to topic %s: %v\n", topic, err)
		return "", err
	}
	log.Printf("Client: Successfully sent message to topic %s: %s\n", topic, response)
	return response, nil
}
