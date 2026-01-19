# 開発ガイドライン

本ドキュメントでは、プロジェクトの開発に関するガイドラインを説明します。

## 目次

- [プロジェクト構造](#プロジェクト構造)
- [コーディング規約](#コーディング規約)
- [開発フロー](#開発フロー)
- [テスト](#テスト)
- [コードレビュー](#コードレビュー)
- [デバッグ](#デバッグ)

## プロジェクト構造

本プロジェクトは、ドメイン駆動設計（DDD）とクリーンアーキテクチャの原則に基づいて設計されています。

```
internal/
├── domain/                    # ドメイン層（ビジネスロジック）
│   ├── currency/
│   ├── transaction/
│   ├── payment_request/
│   ├── redemption_code/
│   └── service/
├── application/              # アプリケーション層（ユースケース）
│   ├── currency/
│   ├── payment/
│   ├── history/
│   └── code_redemption/
├── infrastructure/          # インフラストラクチャ層（実装詳細）
│   ├── persistence/
│   ├── observability/
│   └── config/
└── presentation/            # プレゼンテーション層（API）
    ├── rest/
    ├── grpc/
    └── openapi/
```

### レイヤーの責務

- **ドメイン層**: ビジネスロジックとエンティティの定義
- **アプリケーション層**: ユースケースの実装、トランザクション管理
- **インフラストラクチャ層**: データベース、外部サービスとの連携
- **プレゼンテーション層**: HTTP/gRPCハンドラー、ルーティング

### 依存関係の方向

- 外側のレイヤーは内側のレイヤーに依存する
- 内側のレイヤーは外側のレイヤーに依存しない
- インターフェースは内側のレイヤーで定義し、実装は外側のレイヤーで行う

## コーディング規約

### Goのコーディング規約

- [Effective Go](https://go.dev/doc/effective_go) に準拠
- `gofmt` でフォーマット
- `golint` と `golangci-lint` でリント

### 命名規則

- **パッケージ名**: 小文字、単数形、簡潔に（例: `currency`, `handler`）
- **関数名**: パスカルケース（公開）またはキャメルケース（非公開）
- **変数名**: キャメルケース
- **定数**: パスカルケースまたは大文字のスネークケース

### エラーハンドリング

- エラーは必ずチェックする
- エラーメッセージは明確に、コンテキストを含める
- カスタムエラーは `errors.New()` または `fmt.Errorf()` で作成

```go
// 良い例
if err != nil {
    return fmt.Errorf("failed to save currency: %w", err)
}

// 悪い例
if err != nil {
    return err  // コンテキストがない
}
```

### ログ

- 構造化ログを使用（OpenTelemetry Logger）
- ログレベルを適切に設定（DEBUG, INFO, WARN, ERROR）
- トレースIDを含める

```go
logger.Info(ctx, "currency granted",
    "user_id", userID,
    "currency_type", currencyType,
    "amount", amount,
)
```

### コメント

- 公開関数には必ずコメントを記述
- パッケージにはパッケージコメントを記述
- 複雑なロジックには説明コメントを追加

```go
// Grant は指定されたユーザーに通貨を付与します。
// amount は正の整数値である必要があります。
func (s *CurrencyApplicationService) Grant(ctx context.Context, req *GrantRequest) (*GrantResponse, error) {
    // ...
}
```

## 開発フロー

### ブランチ戦略

- `main`: 本番環境用ブランチ
- `develop`: 開発用ブランチ
- `feature/*`: 機能追加用ブランチ
- `fix/*`: バグ修正用ブランチ

### コミットメッセージ

コミットメッセージは明確に、変更内容を説明します:

```
feat: 通貨消費APIに優先順位制御を追加

- use_priorityフラグで無料通貨優先消費を実装
- ConsumptionDetailレスポンスを追加
- 単体テストを追加
```

### プルリクエスト

1. 機能ブランチから `develop` ブランチへPRを作成
2. CIが自動的にテストを実行
3. コードレビューを依頼
4. 承認後にマージ

### コード生成

gRPCコードの生成:

```powershell
.\scripts\generate-grpc.ps1
```

## テスト

### 単体テスト

- すべての公開関数にテストを記述
- テストファイルは `*_test.go` という命名規則
- テスト関数は `Test*` という命名規則

```go
func TestCurrency_Grant(t *testing.T) {
    tests := []struct {
        name    string
        amount  int64
        wantErr bool
    }{
        {
            name:    "正常な付与",
            amount:  100,
            wantErr: false,
        },
        {
            name:    "負の値はエラー",
            amount:  -10,
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // テスト実装
        })
    }
}
```

### テストの実行

```powershell
# すべてのテストを実行
go test ./...

# カバレッジ付きテスト
go test -cover ./...

# 特定のパッケージのテスト
go test ./internal/domain/currency/...

# ベンチマークテスト
go test -bench=. ./...
```

### 統合テスト

- データベースを使用するテストは統合テストとして実装
- テスト用のデータベースを使用
- テスト後にクリーンアップ

### モック

- インターフェースをモック化してテスト
- `gomock` や手動実装のモックを使用

## コードレビュー

### レビューの観点

- **機能性**: 要件を満たしているか
- **コード品質**: 読みやすく、保守しやすいか
- **パフォーマンス**: 適切なパフォーマンスか
- **セキュリティ**: セキュリティ上の問題はないか
- **テスト**: 適切なテストが記述されているか

### レビュー時のチェックリスト

- [ ] コードが仕様に準拠しているか
- [ ] エラーハンドリングが適切か
- [ ] ログが適切に記録されているか
- [ ] テストが実装されているか
- [ ] ドキュメントが更新されているか
- [ ] セキュリティチェックが完了しているか

## デバッグ

### ログの確認

```powershell
# Docker Composeの場合
docker-compose logs -f app

# ローカル環境の場合
# ログは標準出力に出力される
```

### トレーシング

Jaeger UIでトレースを確認:

1. http://localhost:16686 にアクセス
2. サービス名 `gem-server` を選択
3. トレースを検索・確認

### データベースの確認

```powershell
# Docker Composeの場合
docker-compose exec mysql mysql -u gem_user -p gem_db

# ローカル環境の場合
mysql -u gem_user -p gem_db
```

### デバッグポイントの設定

VS CodeやGoLandなどのIDEでデバッグポイントを設定してデバッグできます。

## パフォーマンス最適化

### データベースクエリ

- インデックスを適切に使用
- N+1問題を回避
- 必要に応じてバッチ処理を使用

### キャッシュ

- Redisを使用して頻繁にアクセスされるデータをキャッシュ
- キャッシュの有効期限を適切に設定

### 並行処理

- 適切な並行処理を使用（goroutine、チャネル）
- 競合状態を避ける（楽観的ロック、排他制御）

## セキュリティ

### 認証・認可

- JWTトークンを適切に検証
- ユーザーIDの検証を必ず実施

### SQLインジェクション対策

- プリペアドステートメントを使用
- パラメータ化クエリを使用

### 入力検証

- すべての入力を検証
- 適切なバリデーションを実装

## 参考資料

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Domain-Driven Design](https://martinfowler.com/bliki/DomainDrivenDesign.html)
