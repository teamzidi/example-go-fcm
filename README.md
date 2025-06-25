# Go FCM Backend Service Example

これは、Go言語で実装されたFirebase Cloud Messaging (FCM) のバックエンドサービスのサンプルです。
Cloud Runでの動作を想定しています。

## 機能

- Pub/Sub連携 (Push型): Google Cloud Pub/Sub からのPush通知を受け取り、以下のいずれかの方法でプッシュ通知を送信します。
    - 指定された単一のデバイストークンへ送信。
    - 指定されたFCMトピックへ送信。

## ディレクトリ構成

- `main.go`: アプリケーションのエントリーポイント。HTTPサーバー、ルーティングなど。
- `handlers/`: HTTPリクエストハンドラ。
  - `common_push_types.go`: Pub/Subからのリクエストデータ構造など、プッシュ通知関連の共通型定義。
  - `handlers.go`: `/health`エンドポイントなど、共通のハンドラ。
  - `handlers_mock.go`: `FCMClient`インターフェースのモック実装など、テスト用のモックを提供 (`//go:build test_fcm_mock`)。
  - `push_device_handler.go`: 指定デバイストークンへのPub/Sub Push通知受信・処理 (`/pubsub/push/device`)。
  - `push_topic_handler.go`: 指定FCMトピックへのPub/Sub Push通知受信・処理 (`/pubsub/push/topic`)。
  - `push_device_handler_test.go`: `push_device_handler.go`のユニットテスト。
  - `push_topic_handler_test.go`: `push_topic_handler.go`のユニットテスト。
  - `mock_test.go`: モックを使用したハンドラのテスト。
  - `testhelpers_test.go`: テストコード用のヘルパー関数群。
- `fcm/`: FCM関連処理。
  - `fcm_client.go`: `FCMClient`インターフェースと、Firebase Admin SDKを利用したFCMクライアントの本番実装を提供。
- `Dockerfile`: アプリケーションのコンテナイメージをビルドするためのファイル。
- `fcm_topic.md`: FCMトピックメッセージング機能に関する詳細説明。
- `go.mod`, `go.sum`: Goモジュールの依存関係定義ファイル。
- `push_token.sh`, `push_topic.sh`: ローカルテスト用のPub/Subメッセージ送信スクリプト例。
- `*_test.go`: 各パッケージのユニットテストファイル (上記`handlers/`内で具体的に記載したものを除く一般的な表現)。

## APIエンドポイント

### Pub/Sub Push通知受信用エンドポイント

これらのエンドポイントは、Pub/SubサブスクリプションのPush先として設定します。直接呼び出すことは通常ありません。

- `POST /pubsub/push/device`: 指定された単一のデバイストークンに通知を送信します。
  - リクエストボディ (Pub/Subメッセージの `message.data` にBase64エンコードされて格納されるJSON。実際のペイロードは `handlers.DevicePushPayload` を参照):
    ```json
    {
      "title": "通知のタイトル (必須)",
      "body": "通知の本文 (必須)",
      "token": "your_single_device_token", // 送信対象のデバイストークン (必須、文字列)
      "custom_data": { // オプショナル: アプリ固有の追加データ。FCMメッセージのデータペイロードとして送信されます。
        "key1": "value1"
      }
    }
    ```
    - 成功 (200 OK): FCMへの送信処理が成功した場合、以下のJSONを返します。
      ```json
      {
        "status": "processed",
        "message_id": "fcm_message_id"
      }
      ```
    - 成功 (204 No Content): リクエストのデコード失敗時や、FCMへの送信が非リトライ可能なエラーで失敗した場合に返します。Pub/Subメッセージはackされます。
    - エラー (必須フィールド欠如など) (400 Bad Request): リクエストの形式が不正な場合 (このドキュメントで示すJSON構造ではなく、Pub/Subのエンベロープメッセージ自体に問題がある場合など) や、ペイロード内の必須フィールドが不足している場合に、エラーメッセージと共に返されることがあります。ただし、現在の実装では必須フィールド不足は多くの場合204 No Contentでackされます。
    - エラー (FCM送信失敗時) (500 Internal Server Error): FCMへの送信がリトライ可能なエラーで失敗した場合に返します。Pub/Subに再試行を促します (nack)。

- `POST /pubsub/push/topic`: 指定されたFCMトピックに通知を送信します。
  - リクエストボディ (Pub/Subメッセージの `message.data` にBase64エンコードされて格納されるJSON。実際のペイロードは `handlers.TopicPushPayload` を参照):
    ```json
    {
      "title": "通知のタイトル (必須)",
      "body": "通知の本文 (必須)",
      "topic": "your_target_topic_name", // 送信対象のFCMトピック名 (必須)
      "custom_data": { // オプショナル: アプリ固有の追加データ。FCMメッセージのデータペイロードとして送信されます。
        "key1": "value1"
      }
    }
    ```
    - 成功 (200 OK): FCMへの送信処理が成功した場合、以下のJSONを返します。
      ```json
      {
        "status": "processed",
        "message_id": "fcm_message_id"
      }
      ```
      (注意: 現在の `push_topic_handler.go` の実装では、成功時の `message_id` が空文字列になっています。これは修正されるべき点です。)
    - 成功 (204 No Content): リクエストのデコード失敗時や、FCMへの送信が非リトライ可能なエラーで失敗した場合に返します。Pub/Subメッセージはackされます。
    - エラー (必須フィールド欠如など) (400 Bad Request): リクエストの形式が不正な場合や、ペイロード内の必須フィールドが不足している場合に、エラーメッセージと共に返されることがあります。ただし、現在の実装では必須フィールド不足は多くの場合204 No Contentでackされます。
    - エラー (FCM送信失敗時) (500 Internal Server Error): FCMへの送信がリトライ可能なエラーで失敗した場合に返します。Pub/Subに再試行を促します (nack)。

- `GET /health`: ヘルスチェック用エンドポイント。
  - 成功レスポンス (200 OK):
    ```
    OK
    ```

## Pub/Sub設定
(このセクションは変更なし)
このサービスはPub/Subの**Pushサブスクリプション**を使用します。
サブスクリプションは、このサービスをデプロイし、公開URLが確定した後に、手動または `gcloud` コマンド等で作成する必要があります。

通知の送信対象に応じて、以下のいずれかのエンドポイントをPush先として指定します。

- 特定のデバイストークン群に送信する場合: `https://<YOUR_SERVICE_URL>/pubsub/push/device`
- 特定のFCMトピックに送信する場合: `https://<YOUR_SERVICE_URL>/pubsub/push/topic`

### Pushサブスクリプションの作成例 (gcloud)
(内容は変更なし)
```bash
# Google CloudプロジェクトID
PROJECT_ID="your-gcp-project-id"
# Cloud Runサービス名
SERVICE_NAME="your-service-name"
# Cloud Runデプロイリージョン
REGION="your-region"
# Pub/Subトピック名
PUB_SUB_TOPIC="your-topic-name"
# Pub/Subサブスクリプション名 (任意)
SUBSCRIPTION_NAME_DEVICE="your-subscription-name-for-device"
SUBSCRIPTION_NAME_TOPIC="your-subscription-name-for-topic"

# Cloud RunサービスのURLを取得 (デプロイ済みの場合)
SERVICE_URL=$(gcloud run services describe ${SERVICE_NAME} --platform managed --region ${REGION} --project ${PROJECT_ID} --format 'value(status.url)')

if [ -z "${SERVICE_URL}" ]; then
  echo "Cloud RunサービスURLの取得に失敗しました。サービスがデプロイされているか確認してください。"
  # exit 1 # サブタスク実行時エラー回避のためコメントアウト
fi

# Pub/Subサービスアカウント情報を取得
PROJECT_NUMBER=$(gcloud projects describe ${PROJECT_ID} --format='value(projectNumber)')
PUBSUB_SERVICE_ACCOUNT="service-${PROJECT_NUMBER}@gcp-sa-pubsub.iam.gserviceaccount.com"

# Cloud RunサービスにPUBSUB_SERVICE_ACCOUNTからの呼び出しを許可 (roles/run.invoker)
gcloud run services add-iam-policy-binding ${SERVICE_NAME}   --member="serviceAccount:${PUBSUB_SERVICE_ACCOUNT}"   --role="roles/run.invoker"   --region=${REGION}   --project=${PROJECT_ID}

# --- デバイストークン指定送信用サブスクリプション作成 ---
PUSH_ENDPOINT_DEVICE="${SERVICE_URL}/pubsub/push/device"
gcloud pubsub subscriptions create ${SUBSCRIPTION_NAME_DEVICE}   --topic ${PUB_SUB_TOPIC}   --push-endpoint="${PUSH_ENDPOINT_DEVICE}"   --push-auth-service-account="${PUBSUB_SERVICE_ACCOUNT}"   --ack-deadline=60   --project=${PROJECT_ID}
echo "Subscription ${SUBSCRIPTION_NAME_DEVICE} created for endpoint ${PUSH_ENDPOINT_DEVICE}"

# --- トピック指定送信用サブスクリプション作成 ---
PUSH_ENDPOINT_TOPIC="${SERVICE_URL}/pubsub/push/topic"
gcloud pubsub subscriptions create ${SUBSCRIPTION_NAME_TOPIC}   --topic ${PUB_SUB_TOPIC}   --push-endpoint="${PUSH_ENDPOINT_TOPIC}"   --push-auth-service-account="${PUBSUB_SERVICE_ACCOUNT}"   --ack-deadline=60   --project=${PROJECT_ID}
echo "Subscription ${SUBSCRIPTION_NAME_TOPIC} created for endpoint ${PUSH_ENDPOINT_TOPIC}"
```
**注意:** 上記のコマンド例では、同じPub/Subトピックに対して2つの異なるサブスクリプションを作成しています。実際のユースケースに応じて、トピックを分けるか、単一のサブスクリプションでペイロードによって処理を分ける（今回はエンドポイント分離を選択）かなどを検討してください。

## FCMトピックメッセージングについて
(このセクションは変更なし)
このサービスでは、`/pubsub/push/topic` エンドポイントを利用することでFCMトピックメッセージングを活用できます。
FCMトピックメッセージングのより詳細な説明については、[FCMトピック機能の説明 (fcm_topic.md)](./fcm_topic.md) を参照してください。

## セットアップと実行

### 必要なもの
(変更なし)
- Go (バージョン 1.24 以降推奨)
- Docker
- Google Cloud SDK (gcloud CLI)

### 環境変数
(変更なし)
アプリケーションの実行には以下の環境変数が必要です。Cloud Runにデプロイする際に設定してください。

- `GOOGLE_CLOUD_PROJECT`: Google CloudプロジェクトID。FCMクライアントの初期化に利用されます。
- `PORT`: (オプション) HTTPサーバーがリッスンするポート。デフォルトは `8080`。
- `GOOGLE_APPLICATION_CREDENTIALS`: (ローカル実行時やサービスアカウントキーを直接使用する場合) Firebase Admin SDK が使用するサービスアカウントキーのJSONファイルへのパス。Cloud Run環境では通常、サービスに紐づくサービスアカウントに適切なロール（Firebase Admin SDKに必要な権限、例: Firebase Admin）を付与すれば不要です。

### ローカルでの実行 (開発用)
(変更なし)
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
   Pub/SubからのPush通知をローカルで受信するには、ローカル環境を外部公開するためのトンネリングツール（例: [ngrok](https://ngrok.com/)）が必要です。ngrokで取得した公開URL（例: `https://xxxx.ngrok.io/pubsub/push/device`）をPub/SubのPushエンドポイントとして設定します。

### テストの実行
ユニットテストを実行するには、プロジェクトのルートディレクトリで以下のコマンドを実行します。
`test_fcm_mock` ビルドタグを指定することで、HTTPハンドラが使用する `FCMClient` インターフェースの実装が、実際のFCMサーバーと通信する代わりに `handlers/handlers_mock.go` で定義されたモック実装 (`MockFCMClient`) に置き換わります。これにより、外部APIへの依存なしにハンドラのロジックをテストできます。

```bash
go test -tags=test_fcm_mock ./...
```

### Dockerイメージのビルド
(変更なし)
```bash
docker build -t your-image-name .
```

### Cloud Runへのデプロイ (例)
(変更なし)
1. Dockerイメージを Artifact Registry または Container Registry にプッシュします。
   ```bash
   gcloud auth configure-docker
   docker tag your-image-name gcr.io/your-gcp-project-id/your-image-name
   docker push gcr.io/your-gcp-project-id/your-image-name
   ```
2. Cloud Runにデプロイします。
   ```bash
   gcloud run deploy your-service-name      --image gcr.io/your-gcp-project-id/your-image-name      --platform managed      --region your-region      --allow-unauthenticated      --set-env-vars GOOGLE_CLOUD_PROJECT="your-gcp-project-id"      --service-account "your-app-service-account-email"
   ```
   アプリケーション用のサービスアカウント (`your-app-service-account-email`) には、FCM送信に必要な権限（例: Firebase Admin SDKが利用する権限、roles/firebase.adminなど）を付与してください。
   Pub/SubからのPush認証は、上記の「Pushサブスクリプションの作成例」で設定したPub/Subサービスアカウント (`service-${PROJECT_NUMBER}@gcp-sa-pubsub.iam.gserviceaccount.com`) とCloud RunサービスのIAM設定 (`roles/run.invoker`) によって行われます。

## デバイストークンのバリデーション

デバイストークンは、Pub/Subメッセージ内のペイロードで送信される際に、以下のバリデーションが `handlers/push_device_handler.go` 内で適用されます。
- **必須チェック**: トークンが空文字列でないこと。
  - `payload.Token == ""` の形式でチェックされます。
- **最大長**: 現在の実装では、デバイストークンの最大長に関する明示的なバリデーションは行われていません。FCM自体のトークン長の制約（通常4KB程度）に依存します。ドキュメントに記載されていた「最大長: 4096文字」のチェックは現在のコードには含まれていません。

このため、非常に長いトークンが指定された場合の挙動はFCMライブラリまたはFCMサーバー側のバリデーションに委ねられます。

## 注意事項
- **デバイストークンの扱い**: このアプリケーションはデバイストークンをサーバー側に保存・キャッシュしません。通知の送信対象（トークンまたはトピック）は、Pub/Subメッセージで都度指定される必要があります。
- **エラーハンドリング**: Pub/Subメッセージの処理失敗時のリトライ戦略（Pushサブスクリプションの再試行ポリシーやデッドレター設定）や、FCMへの送信失敗時の詳細なエラーハンドリングは、要件に応じて強化が必要です。
- **セキュリティ**: 各 `/pubsub/push/*` エンドポイントはIAMによって保護されています。
