# ローカルビルド対応のDockerfile
# 事前にローカルでビルドされたバイナリを使用して実行環境を構築

# テスト/開発用ステージ（Goツールチェーン含む）
FROM golang:1.24.5-alpine AS test

# 必要なツールをインストール
RUN apk --no-cache add git ca-certificates tzdata wget curl make bash

# 作業ディレクトリを設定
WORKDIR /app

# プロジェクトファイルをすべてコピー
COPY . .

# 依存関係をダウンロード
RUN go mod download

# テスト環境であることを明示
ENV DOCKER_CONTAINER=true
ENV GIN_MODE=test

# ログディレクトリを作成
RUN mkdir -p /app/logs

# ポート8000を公開
EXPOSE 8000

# デフォルトコマンド（テスト用に無限ループ）
CMD ["tail", "-f", "/dev/null"]

# 本番実行環境（事前ビルドされたバイナリを使用）
FROM alpine:3.19 AS production

# セキュリティ更新とca-certificatesをインストール
RUN apk --no-cache add ca-certificates tzdata wget

# 作業ディレクトリを設定
WORKDIR /app

# ログディレクトリを作成
RUN mkdir -p /app/logs

# ローカルでビルドされたバイナリをコピー
COPY bin/memo-app ./memo-app

# バイナリが実行可能であることを確認
RUN chmod +x ./memo-app

# Docker環境であることを明示
ENV DOCKER_CONTAINER=true

# ポート8000を公開
EXPOSE 8000

# ヘルスチェック設定
HEALTHCHECK --interval=30s --timeout=5s --start-period=40s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8000/health || exit 1

# アプリケーションを実行
CMD ["./memo-app"]
