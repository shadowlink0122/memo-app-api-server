# 本番環境デプロイガイド

## 🚀 本番環境構成

このアプリケーションの本番環境では以下の構成を使用します：

### アーキテクチャ
```
[ロードバランサー] → [memo-app Container] → [AWS RDS PostgreSQL]
                                        → [AWS S3 (ログ保存)]
```

### 使用サービス
- **アプリケーション**: Docker Container（EC2/ECS/EKS等）
- **データベース**: AWS RDS PostgreSQL 15+
- **ストレージ**: AWS S3（ログファイル保存用）
- **監視**: Docker Watchtower（オプション）

## 📋 デプロイ手順

### 1. AWS リソースの準備

#### RDS PostgreSQL の作成
```bash
# AWS CLI での作成例
aws rds create-db-instance \
  --db-instance-identifier memo-app-db \
  --db-instance-class db.t3.micro \
  --engine postgres \
  --engine-version 15.4 \
  --master-username memo_user \
  --master-user-password "your-secure-password" \
  --allocated-storage 20 \
  --vpc-security-group-ids sg-xxxxxxxxx \
  --db-subnet-group-name your-subnet-group
```

#### S3 バケットの作成
```bash
# AWS CLI での作成例
aws s3 mb s3://memo-app-logs-prod
aws s3api put-bucket-versioning \
  --bucket memo-app-logs-prod \
  --versioning-configuration Status=Enabled
```

### 2. 環境変数の設定

```bash
# 本番環境用の設定をコピー
cp .env.production .env

# 必須項目を設定
export DB_HOST="your-rds-endpoint.region.rds.amazonaws.com"
export DB_PASSWORD="your-secure-database-password"
export S3_ACCESS_KEY_ID="your-aws-access-key"
export S3_SECRET_ACCESS_KEY="your-aws-secret-key"
```

### 3. アプリケーションのデプロイ

```bash
# 本番環境でアプリケーションを起動
make docker-prod-up

# ログの確認
docker compose -f docker-compose.prod.yml logs -f app

# ヘルスチェック
curl http://localhost:80/health
```

### 4. 監視設定（オプション）

```bash
# Watchtowerによる自動監視を有効化
docker compose -f docker-compose.prod.yml --profile monitoring up -d
```

## 🔒 セキュリティ考慮事項

### 必須設定
- [ ] データベースのSSL接続（`DB_SSLMODE=require`）
- [ ] 強力なデータベースパスワード
- [ ] AWS IAMロールでの最小権限設定
- [ ] セキュリティグループでのポート制限
- [ ] S3バケットポリシーの適切な設定

### 推奨設定
- [ ] AWS WAFの設定
- [ ] CloudFrontでのCDN設定
- [ ] ALBでのHTTPS終端
- [ ] VPC内でのプライベートサブネット使用
- [ ] CloudWatchでの監視設定

## 📊 監視・ログ

### ログ出力先
- **アプリケーションログ**: AWS S3バケット
- **コンテナログ**: Docker jsonファイル（max 10MB × 3ファイル）
- **ヘルスチェック**: `/health` エンドポイント

### 監視項目
- CPU使用率（制限: 0.8 CPU）
- メモリ使用量（制限: 800MB）
- ディスク使用量
- データベース接続状態
- S3アップロード成功率

## 🔄 アップデート手順

```bash
# 1. 新しいコードをプル
git pull origin main

# 2. コンテナを再ビルド・再起動
docker compose -f docker-compose.prod.yml build app
docker compose -f docker-compose.prod.yml up -d app

# 3. ヘルスチェックで確認
curl http://localhost:80/health
```

## 💡 トラブルシューティング

### よくある問題

1. **データベース接続エラー**
   ```bash
   # RDSのセキュリティグループ確認
   # DB_HOST, DB_PASSWORD の確認
   ```

2. **S3アップロードエラー**
   ```bash
   # IAM権限確認
   # S3_ACCESS_KEY_ID, S3_SECRET_ACCESS_KEY の確認
   ```

3. **メモリ不足**
   ```bash
   # リソース制限の調整
   # docker-compose.prod.yml の memory設定を変更
   ```
