package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/teamzidi/example-go-fcm/handlers"
	// "github.com/teamzidi/example-go-fcm/pubsub" // ← 不要になるのでコメントアウトまたは削除
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 環境変数から設定を読み込む
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // デフォルトポート
	}

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		log.Println("WARNING: GOOGLE_CLOUD_PROJECT environment variable not set. This might be an issue for FCM client initialization if not running on Cloud Run.")
	}

	// Pullサブスクライバ用の環境変数はアプリケーションロジックからは不要になる
	// pubsubSubscriptionID := os.Getenv("PUBSUB_SUBSCRIPTION_ID")
	// if pubsubSubscriptionID == "" {
	// 	log.Fatal("PUBSUB_SUBSCRIPTION_ID environment variable is required.")
	// }

	// デバイストークンストアの初期化

	// FCMクライアントの初期化
	fcmClient, err := handlers.newFcmHandlerClient(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize FCM client: %v", err)
	}
	log.Println("FCM client initialized.")

	// Pub/Sub Pushハンドラの初期化

	// Pullサブスクライバの初期化と起動処理は削除
	// subscriber, err := pubsub.NewSubscriber(ctx, projectID, pubsubSubscriptionID, fcmClient, deviceStore)
	// if err != nil {
	// 	log.Fatalf("Failed to initialize Pub/Sub subscriber: %v", err)
	// }
	// if subscriber == nil {
	// 	log.Printf("Pub/Sub subscriber was not fully initialized...")
	// } else {
	// 	go subscriber.StartReceiving(ctx)
	// 	log.Printf("Pub/Sub subscriber started for subscription %s.", pubsubSubscriptionID)
	// 	defer subscriber.Close()
	// }


	// HTTPルーターの設定
	mux := http.NewServeMux()
	// デバイストークン登録APIハンドラ (既存)
	// Pub/Sub Push受信用ハンドラ (デバイス指定)
	pushDeviceHandler := handlers.NewPushDeviceHandler(fcmClient)
	mux.Handle("/pubsub/push/device", pushDeviceHandler)
	// Pub/Sub Push受信用ハンドラ (トピック指定)
	pushTopicHandler := handlers.NewPushTopicHandler(fcmClient)
	mux.Handle("/pubsub/push/topic", pushTopicHandler)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	log.Printf("Starting server on port %s\n", port)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
