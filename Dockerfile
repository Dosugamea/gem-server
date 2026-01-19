# マルチステージビルド

# ビルドステージ
FROM golang:1.24-alpine AS builder

# ビルドに必要なパッケージをインストール
RUN apk add --no-cache git make

# golang-migrateをインストール
RUN go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# swagをインストール（Swaggerドキュメント生成用）
RUN go install github.com/swaggo/swag/cmd/swag@latest

# 作業ディレクトリを設定
WORKDIR /app

# go.modとgo.sumをコピーして依存関係をダウンロード
COPY go.mod go.sum ./
RUN go mod download

# ソースコードをコピー
COPY . .

# Swaggerドキュメントを生成
RUN swag init -g cmd/server/main.go -o docs

# アプリケーションをビルド
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o /app/bin/gem-server ./cmd/server

# 実行ステージ
FROM alpine:latest

# セキュリティアップデートと必要なパッケージをインストール
RUN apk --no-cache add ca-certificates tzdata wget netcat-openbsd

# タイムゾーンを設定（オプション）
ENV TZ=Asia/Tokyo

# 作業ディレクトリを設定
WORKDIR /app

# ビルドステージからバイナリをコピー
COPY --from=builder /app/bin/gem-server .

# golang-migrateをコピー
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate

# マイグレーションファイルをコピー
COPY --from=builder /app/migrations ./migrations

# 静的ファイルをコピー
COPY --from=builder /app/public ./public

# 非rootユーザーを作成して使用
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

# ポートを公開
EXPOSE 8080

# ヘルスチェック
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# アプリケーションを実行
CMD ["./gem-server"]
