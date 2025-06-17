# Go FCM Backend Service Example

これは、Go言語で実装されたFirebase Cloud Messaging (FCM) のバックエンドサービスのサンプルです。
Cloud Runでの動作を想定しています。

## 機能

- デバイス登録API: クライアント（モバイルアプリなど）からデバイストークンを受け取り保存します。
- Pub/Sub連携 (Push型): Google Cloud Pub/Sub からのPush通知を受け取り、登録されているデバイスにプッシュ通知を送信します。

## ディレクトリ構成

- `main.go`: アプリケーションのエントリーポイント。HTTPサーバー、ルーティングなど。
- `handlers/`: HTTPリクエストハンドラ。
  - `registration.go`: デバイストークン登録処理。
  - `push_handler.go`: Pub/SubからのPush通知受信・処理。
- `fcm/`: FCM関連処理。
  - `fcm.go`: FCMクライアントの初期化、メッセージ送信。
- `store/`: デバイストークンストレージ。
  - `devicestore.go`: インメモリでのデバイストークン管理。
- `Dockerfile`: アプリケーションのコンテナイメージをビルドするためのファイル。

## APIエンドポイント

- `POST /register`: デバイストークンを登録します。
  - リクエストボディ (JSON):
    ```json
    {
      "token": "YOUR_DEVICE_TOKEN"
    }
    ```
  - レスポンス:
    - 成功 (新規登録) (201 Created):
      ```json
      {
        "message": "Device token registered successfully"
      }
      ```
    - 成功 (既に登録済み) (409 Conflict):
      ```json
      {
        "message": "Device token already exists"
      }
      ```
    - エラー (バリデーションエラー: トークンが空、長すぎる等) (400 Bad Request):
      プレーンテキストでエラーメッセージ。
- `POST /pubsub/push`: Pub/SubからのPush通知受信用エンドポイント。直接呼び出すのではなく、Pub/SubサブスクリプションのPush先として設定します。
  - 内部でFCMへの送信に失敗し、再試行を促す場合は HTTP 503 Service Unavailable を返すことがあります。
- `GET /health`: ヘルスチェック用エンドポイント。
  - 成功レスポンス (200 OK):
    ```
    OK
    ```

## Pub/Sub設定

このサービスはPub/Subの**Pushサブスクリプション**を使用します。
サブスクリプションは、このサービスをデプロイし、公開URLが確定した後に、手動または `gcloud` コマンド等で作成する必要があります。

### Pushサブスクリプションの作成例

```bash
# トピックがまだ存在しない場合は作成
gcloud pubsub topics create your-topic-name --project=your-gcp-project-id

# Cloud RunサービスのURL (例: https://your-service-name-xxxxxx-an.a.run.app)
SERVICE_URL="your-cloud-run-service-url"
PUSH_ENDPOINT="${SERVICE_URL}/pubsub/push"

# Pub/SubサービスアカウントにCloud Run Invokerロールを付与するための情報を取得
PROJECT_NUMBER=$(gcloud projects describe your-gcp-project-id --format='value(projectNumber)')
PUBSUB_SERVICE_ACCOUNT="service-${PROJECT_NUMBER}@gcp-sa-pubsub.iam.gserviceaccount.com"

# Cloud RunサービスにPUBSUB_SERVICE_ACCOUNTからの呼び出しを許可 (roles/run.invoker)
gcloud run services add-iam-policy-binding your-service-name \
  --member="serviceAccount:${PUBSUB_SERVICE_ACCOUNT}" \
  --role="roles/run.invoker" \
  --region=your-region \
  --project=your-gcp-project-id

# Pushサブスクリプションを作成
gcloud pubsub subscriptions create your-subscription-name \
  --topic your-topic-name \
  --push-endpoint="${PUSH_ENDPOINT}" \
  --push-auth-service-account="${PUBSUB_SERVICE_ACCOUNT}" \
  --ack-deadline=60 \ # 必要に応じて調整 (デフォルト10秒)
  --project=your-gcp-project-id
```

### Pub/Subメッセージ形式

Pub/Subトピックに発行するメッセージのペイロードは以下のJSON形式を期待します。

```json
{
  "title": "Notification Title",
  "body": "Notification Body Text"
}
```
このJSONがPub/Sub Pushリクエストの `message.data` フィールドにBase64エンコードされて格納されます。

## セットアップと実行

### 必要なもの

- Go (バージョン 1.24 以降推奨)
- Docker
- Google Cloud SDK (gcloud CLI)

### 環境変数

アプリケーションの実行には以下の環境変数が必要です。Cloud Runにデプロイする際に設定してください。

- `GOOGLE_CLOUD_PROJECT`: Google CloudプロジェクトID。FCMクライアントの初期化に利用されます。
- `PORT`: (オプション) HTTPサーバーがリッスンするポート。デフォルトは `8080`。
- `GOOGLE_APPLICATION_CREDENTIALS`: (ローカル実行時やサービスアカウントキーを直接使用する場合) Firebase Admin SDK が使用するサービスアカウントキーのJSONファイルへのパス。Cloud Run環境では通常、サービスに紐づくサービスアカウントに適切なロール（Firebase Admin SDKに必要な権限、例: Firebase Admin）を付与すれば不要です。

**以下の環境変数はPull型サブスクリプションで利用していましたが、Push型への変更によりアプリケーションコードでは直接参照しなくなりました。**
- `PUBSUB_SUBSCRIPTION_ID`
- `PUBSUB_TOPIC_ID`

### ローカルでの実行 (開発用)

1. リポジトリをクローンします。
2. 必要な環境変数を設定します。
   ```bash
   export GOOGLE_CLOUD_PROJECT="your-gcp-project-id"
   # export GOOGLE_APPLICATION_CREDENTIALS="/path/to/your/service-account-key.json" # 必要に応じて
   ```
3. サーバーを起動します。
   ```bash
   go run main.go
   ```
4. **ローカルでのPush通知テスト**:
   Pub/SubからのPush通知をローカルで受信するには、ローカル環境を外部公開するためのトンネリングツール（例: [ngrok](https://ngrok.com/)）が必要です。ngrokで取得した公開URL（例: `https://xxxx.ngrok.io/pubsub/push`）をPub/SubのPushエンドポイントとして設定します。

### Dockerイメージのビルド

```bash
docker build -t your-image-name .
```

### Cloud Runへのデプロイ (例)

1. Dockerイメージを Artifact Registry または Container Registry にプッシュします。
   ```bash
   gcloud auth configure-docker
   docker tag your-image-name gcr.io/your-gcp-project-id/your-image-name
   docker push gcr.io/your-gcp-project-id/your-image-name
   ```
2. Cloud Runにデプロイします。
   ```bash
   gcloud run deploy your-service-name \
     --image gcr.io/your-gcp-project-id/your-image-name \
     --platform managed \
     --region your-region \
     --allow-unauthenticated \ # `/register` `/health` のため。`/pubsub/push` はIAMで保護
     --set-env-vars GOOGLE_CLOUD_PROJECT="your-gcp-project-id" \
     --service-account "your-app-service-account-email" # 推奨: アプリケーション用のサービスアカウント
   ```
   アプリケーション用のサービスアカウント (`your-app-service-account-email`) には、FCM送信に必要な権限（例: Firebase Admin SDKが利用する権限、roles/firebase.adminなど）を付与してください。
   Pub/SubからのPush認証は、上記の「Pushサブスクリプションの作成例」で設定したPub/Subサービスアカウント (`service-${PROJECT_NUMBER}@gcp-sa-pubsub.iam.gserviceaccount.com`) とCloud RunサービスのIAM設定 (`roles/run.invoker`) によって行われます。

## デバイストークンのバリデーション

登録されるデバイストークンには以下の簡易的なバリデーションが適用されます。
- 空白文字のみでないこと。
- 最大長: 4096文字。

## 注意事項

- **デバイストークンの永続化**: このサンプルではデバイストークンはインメモリに保存されます。アプリケーションが再起動するとトークンは失われます。本番環境ではFirestoreやCloud SQLなどの永続ストレージに保存することを検討してください。
- **エラーハンドリング**: Pub/Subメッセージの処理失敗時のリトライ戦略（Pushサブスクリプションの再試行ポリシーやデッドレター設定）や、FCMへの送信失敗時の詳細なエラーハンドリングは、要件に応じて強化が必要です。
- **セキュリティ**: `/register` エンドポイントは現在認証なしでアクセス可能です。必要に応じて認証機構（APIキー、OAuthなど）を導入してください。`/pubsub/push` エンドポイントはIAMによって保護されています。
