# ビルドステージ
FROM golang:1.24-alpine AS builder

WORKDIR /app

# go.mod と go.sum をコピーして依存関係をダウンロード
COPY go.mod go.sum ./
RUN go mod download

# ソースコードをコピー
COPY . .

# アプリケーションをビルド
# CGO_ENABLED=0 で静的リンクバイナリを生成し、外部依存を減らす
# -ldflags="-s -w" でデバッグ情報を取り除き、バイナリサイズを削減
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-s -w" -o /app/fcm-backend ./main.go

# 実行ステージ
# distroless/static イメージを使用。nonroot ユーザーで実行されることが推奨される。
FROM gcr.io/distroless/static-debian12 AS final

WORKDIR /app

# ビルドステージから実行可能ファイルをコピー
COPY --from=builder /app/fcm-backend /app/fcm-backend

# (オプション) サービスアカウントキーファイルをコピーする場合
# COPY service-account-key.json /app/service-account-key.json
# ENV GOOGLE_APPLICATION_CREDENTIALS /app/service-account-key.json
# distrolessイメージにはシェルがないため、環境変数はビルド時やCloud Runのサービス定義で設定するのが一般的。

# ポートを開放
EXPOSE 8080

# アプリケーションを実行
# CMD ["/app/fcm-backend"]
# distroless/staticでは USER nonroot がデフォルトで設定されている場合がある。
# ユーザーを指定する場合は USER nonroot:nonroot のようにするが、
# staticの場合は実行ファイルがそのままエントリーポイントになることが多い。
ENTRYPOINT ["/app/fcm-backend"]
