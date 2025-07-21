# マルチステージビルドを使用して軽量なイメージを作成
FROM golang:1.24.5-alpine AS builder

# 作業ディレクトリを設定
WORKDIR /app

# 依存関係ファイルをコピー
COPY go.mod go.sum ./

# 依存関係をダウンロード
RUN go mod download

# ソースコードをコピー
COPY src/ ./src/

# バイナリをビルド（静的リンク、サイズ最適化）
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o main src/main.go

# 実行用の軽量イメージ
FROM alpine:3.19

# セキュリティ更新とca-certificatesをインストール
RUN apk --no-cache add ca-certificates tzdata

# 作業ディレクトリを設定
WORKDIR /root/

# ビルド済みバイナリをコピー
COPY --from=builder /app/main .

# ポート8080を公開
EXPOSE 8080

# ヘルスチェック設定
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# アプリケーションを実行
CMD ["./main"]
