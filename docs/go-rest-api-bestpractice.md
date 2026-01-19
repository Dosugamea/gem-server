# Go REST API ベストプラクティス

このドキュメントでは、GoでREST APIを実装する際のベストプラクティスをまとめています。2024-2025年の最新の情報とコミュニティの推奨事項を反映しています。

## 目次

1. [API設計とルーティング](#api設計とルーティング)
2. [プロジェクト構造](#プロジェクト構造)
3. [入力検証とモデリング](#入力検証とモデリング)
4. [エラーハンドリング](#エラーハンドリング)
5. [コンテキストの使用](#コンテキストの使用)
6. [ページネーション、フィルタリング、ソート](#ページネーションフィルタリングソート)
7. [セキュリティ](#セキュリティ)
8. [パフォーマンス最適化](#パフォーマンス最適化)
9. [テストとドキュメント](#テストとドキュメント)
10. [ミドルウェア](#ミドルウェア)
11. [チェックリスト](#チェックリスト)

---

## API設計とルーティング

### リソースベースのURL設計

- **名詞を使用し、動詞は避ける**: HTTPメソッド（GET、POSTなど）がアクションを定義する。例: `/articles` を使用し、`/getArticles` は避ける
  - 出典: [goa.design - HTTP Routing](https://www.goa.design/docs/4-concepts/3-http/2-routing/)

- **リソースは複数形で統一**: 一貫性を保つため、リソース名は複数形、小文字、複数単語はハイフン（kebab-case）を使用
  - 出典: [dev.to - REST API Best Practices](https://dev.to/george_pollock/rest-api-best-practices-why-s-and-how-s-304n)

- **浅いURIネスト**: 深すぎるネストは複雑さと結合を増やす。ネストする場合は意味のある場合のみ（例: `/users/{id}/orders`）
  - 出典: [dev.to - REST API Best Practices](https://dev.to/george_pollock/rest-api-best-practices-why-s-and-how-s-304n)

### APIバージョニング

- **最初からバージョニング**: `/api/v1/...` のようにAPIをバージョン管理し、クライアントを壊すことなく進化を可能にする。セマンティックバージョニングも有用
  - 出典: [infinitejs.com - Common Pitfalls Go REST APIs](https://infinitejs.com/posts/common-pitfalls-go-rest-apis/)

### ルーターの選択

- **標準ライブラリ vs サードパーティ**: 小規模プロジェクトでは `net/http` と `http.ServeMux` で十分。大規模では Chi、Gorilla Mux、Gin などのルーターライブラリを検討
  - 出典: [Medium - Building REST APIs in Go](https://medium.com/@raihanur.rahman.2022/building-rest-apis-in-go-a-comprehensive-guide-from-basics-to-frameworks-0084e481937b)

---

## プロジェクト構造

### 標準的なディレクトリ構造

Goコミュニティで広く認識されている `golang-standards/project-layout` に基づく構造:

```
myproject/
├── cmd/
│   └── api/
│       └── main.go          # エントリーポイント
├── internal/                # プライベートなビジネスロジック
│   └── user/               # 機能ベースのパッケージ
│       ├── handler.go
│       ├── service.go
│       └── repository.go
├── pkg/                     # 外部公開可能な再利用コンポーネント
├── api/                     # OpenAPI仕様ファイル
├── config/                  # 設定ファイル
├── scripts/                 # デプロイやツールスクリプト
├── docs/                    # ドキュメント
├── go.mod
├── go.sum
└── README.md
```

- 出典: [golang-standards/project-layout](https://github.com/golang-standards/project-layout)
- 出典: [glukhov.org - Go Project Structure](https://www.glukhov.org/post/2025/12/go-project-structure/)

### ディレクトリの役割

- **`cmd/`**: 実行可能ファイルの `main.go` を配置。各サブディレクトリが1つの実行可能ファイルに対応
- **`internal/`**: プライベートなビジネスロジック、ドメインモデル、インターフェースなど。外部からインポートできない
- **`pkg/`**: 外部プロジェクトで再利用可能なパブリックコンポーネント
- **`api/`**: OpenAPI/Swagger仕様ファイルを配置

### 機能ベース vs レイヤーベース

- **機能ベース**: `internal/user/` のように機能ごとにまとめる。関連するコードが近くにあり、結合度が低い
- **レイヤーベース**: `handlers/`, `services/`, `repositories/` のようにレイヤーで分ける。小規模アプリには適するが、大規模では関連ロジックが散在しやすい

- 出典: [glukhov.org - Go Project Structure](https://www.glukhov.org/post/2025/12/go-project-structure/)
- 出典: [codingexplorations.com - Managing Files in Go API](https://www.codingexplorations.com/blog/managing-files-in-a-go-api-folder-structure-best-practices-for-organizing-your-project)

---

## 入力検証とモデリング

### 構造体の使用

- **リクエスト/レスポンスモデルに構造体を使用**: Goの静的型付けを活用し、データ構造を表現。JSONタグ、空フィールドの省略などを適切に設定
  - 出典: [moldstud.com - Implementing RESTful APIs in Go](https://moldstud.com/articles/p-implementing-restful-apis-in-go-best-practices-and-performance-optimization)

### 入力検証

- **すべての入力を検証**: パスパラメータ、クエリパラメータ、ボディを検証。`github.com/go-playground/validator/v10` などのライブラリを使用
  - 出典: [infinitejs.com - Common Pitfalls Go REST APIs](https://infinitejs.com/posts/common-pitfalls-go-rest-apis/)

- **検証ライブラリの活用**: Ginの `binding` タグや `go-playground/validator` を使用して構造体レベルで検証
  - 出典: [djamware.com - Input Validation in Golang APIs](https://www.djamware.com/post/6892fa19312f402733c84d61/input-validation-in-golang-apis-using-go-validator-or-gin-binding)

---

## エラーハンドリング

### エラーは値として扱う

- **`(result, error)` を返す**: 通常のフローでは `panic()` を使わず、エラーを値として返す。`panic()` は本当に例外的で回復不可能な場合のみ
  - 出典: [blog.marcnuri.com - Error Handling Best Practices in Go](https://blog.marcnuri.com/error-handling-best-practices-in-go)

### エラーのラッピング

- **`fmt.Errorf` と `%w` でラッピング**: `errors.Is` や `errors.As` で検査できるように、コンテキストを追加しつつ元のエラーを保持
  - 出典: [golang.ntxm.org - Error Handling in Go](https://golang.ntxm.org/docs/error-handling-in-go/best-practices-for-error-handling/)

### 早期リターン

- **Fail Fast**: エラーに遭遇したら早期にリターン。深くネストした `if` ブロックを避け、コードを読みやすく保守しやすくする
  - 出典: [Medium - Error Handling in Go](https://medium.com/@didinjamaludin/error-handling-in-go-idiomatic-patterns-for-clean-code-f8377d420aa3)

### エラーレスポンス

- **内部エラーを漏らさない**: サーバー側で詳細なエラーをログに記録し、クライアントには安全でユーザーフレンドリーなメッセージを返す
  - 出典: [Medium - Error Handling in Go](https://medium.com/@didinjamaludin/error-handling-in-go-idiomatic-patterns-for-clean-code-f8377d420aa3)

- **標準化されたエラーレスポンス形式**: ミドルウェアや集中化されたハンドラーで、ドメインエラーをHTTPステータスコードと標準化されたエラーレスポンス形式にマッピング
  - 出典: [Medium - Error Handling in Go](https://medium.com/@didinjamaludin/error-handling-in-go-idiomatic-patterns-for-clean-code-f8377d420aa3)
  - 出典: [alesr.github.io - Effective RESTful Error Handling in Go](https://alesr.github.io/posts/rest-errors/)

### HTTPステータスコード

適切なHTTPステータスコードを使用:
- `200 OK`: 成功
- `201 Created`: リソース作成成功
- `400 Bad Request`: 不正なリクエスト
- `401 Unauthorized`: 認証が必要
- `404 Not Found`: リソースが見つからない
- `500 Internal Server Error`: サーバー内部エラー

- 出典: [calmops.com - Go Building REST APIs](https://calmops.com/programming/golang/go-building-rest-apis/)

---

## コンテキストの使用

### コンテキストの基本原則

- **常に最初のパラメータとして受け取る**: I/O、外部呼び出し、DB操作を行う関数/メソッドでは、`ctx context.Context` を最初のパラメータとして受け取る
  - 出典: [leapcell.io - The Power of Context.Context in Go Microservices](https://leapcell.io/blog/the-power-of-context-context-in-go-microservices)

- **構造体フィールドやグローバル変数に保存しない**: 古いコンテキストが再利用され、キャンセル動作が予測不能になるリスクがある
  - 出典: [gist.github.com - Context Best Practices](https://gist.github.com/ashokallu/47a70a70c7f6857ff29e1cd3cb97bbd3)

### コンテキストの作成

- **`context.Background()`**: アプリケーションのトップレベル（`main()`、サーバー設定など）で、他のコンテキストが存在しない場合に使用
  - 出典: [leapcell.io - The Power of Context.Context in Go Microservices](https://leapcell.io/blog/the-power-of-context-context-in-go-microservices)

- **`context.TODO()`**: APIシグネチャを満たす必要があるが、適切なコンテキストをまだ決定していない場合に使用
  - 出典: [leapcell.io - The Power of Context.Context in Go Microservices](https://leapcell.io/blog/the-power-of-context-context-in-go-microservices)

### タイムアウトとキャンセル

- **`context.WithCancel`, `context.WithTimeout`, `context.WithDeadline` を使用**: 派生コンテキストを作成
  - 出典: [goperf.dev - Context Patterns](https://goperf.dev/01-common-patterns/context/)

- **`cancel()` を必ず呼び出す**: 返された `cancel()` 関数をできるだけ早く呼び出す（通常は `defer cancel()`）。そうしないとタイマー/リソースがリークする可能性がある
  - 出典: [betterstack.com - Golang Timeouts](https://betterstack.com/community/guides/scaling-go/golang-timeouts/)

- **サービスエッジでタイムアウトを設定**: HTTPハンドラーやAPIゲートウェイでタイムアウトを設定し、ダウンストリームサービスが意味のある作業を超えないようにする
  - 出典: [leapcell.medium.com - Robust Context in Go Microservices](https://leapcell.medium.com/robust-context-in-go-microservices-e7df518b8998)

### キャンセルシグナルの尊重

- **`ctx.Done()` を定期的にチェック**: 長時間実行される関数やループでは、`<-ctx.Done()` を定期的にチェックして、迅速に中止できるようにする
  - 出典: [goperf.dev - Context Patterns](https://goperf.dev/01-common-patterns/context/)

- **`ctx.Err()` でエラーを区別**: タイムアウト（`context.DeadlineExceeded`）か手動キャンセル（`context.Canceled`）かを区別し、正確なエラーメッセージを返す
  - 出典: [calmops.com - Go Context Cancellation Timeouts](https://calmops.com/programming/golang/go-context-cancellation-timeouts)

### コンテキスト値の使用

- **控えめに使用**: リクエストメタデータ（トレース/リクエストID、認証からのユーザーID、ロケール/リージョンなど）のみを含める
  - 出典: [leapcell.io - The Power of Context.Context in Go Microservices](https://leapcell.io/blog/the-power-of-context-context-in-go-microservices)

- **非公開のカスタム型をキーとして使用**: 衝突を避けるため、非公開（プライベート）のカスタム型をキーとして使用。大きなデータ（大きな構造体、DBハンドル、ロガー）をコンテキスト値に入れない
  - 出典: [leapcell.medium.com - Robust Context in Go Microservices](https://leapcell.medium.com/robust-context-in-go-microservices-e7df518b8998)

### グレースフルシャットダウン

- **シグナルベースのコンテキストを使用**: `signal.NotifyContext(os.Interrupt, os.Kill, etc.)` のようなシグナルベースのコンテキストを使用して、新しいリクエストの受け入れを停止。進行中のリクエストを完了させるか、適切にタイムアウトさせる
  - 出典: [gist.github.com - Context Best Practices](https://gist.github.com/ashokallu/47a70a70c7f6857ff29e1cd3cb97bbd3)

### よくある間違い

| 間違い | なぜ悪いか |
|--------|-----------|
| `cancel()` を呼ばない | タイマー、子コンテキスト、リソースがリークする |
| 呼び出しチェーンの深いところで `context.Background()` を使用 | 親のタイムアウト/キャンセルが失われる |
| ブロッキング操作で `ctx.Done()` をチェックしない | キャンセルシグナルが効果を発揮しない |
| `nil` のコンテキストを渡す | パニック、未定義の動作 |

- 出典: [betterstack.com - Golang Timeouts](https://betterstack.com/community/guides/scaling-go/golang-timeouts/)
- 出典: [goperf.dev - Context Patterns](https://goperf.dev/01-common-patterns/context/)
- 出典: [leapcell.io - The Power of Context.Context in Go Microservices](https://leapcell.io/blog/the-power-of-context-context-in-go-microservices)

---

## ページネーション、フィルタリング、ソート

### ページネーション

- **リストを返すエンドポイントにページネーションを実装**: 大きなデータセットでは、オフセットベースよりも**カーソルベース**のページネーションを推奨（安定性とパフォーマンスのため）
  - 出典: [ory.com - API REST Pagination](https://www.ory.com/docs/guides/api-rest-pagination)

- **標準的なパラメータ名を使用**: `limit`, `offset`, `page`, `cursor` などの標準的な名前を使用し、デフォルトと最大制限を設定
  - 出典: [blog.treblle.com - API Pagination Guide](https://blog.treblle.com/api-pagination-guide-techniques-benefits-implementation/)

- **ページネーションメタデータを含める**: レスポンスに `total`, `current_page`, `next`, `previous` などのメタデータを含める。オプションでハイパーメディアや `Link` ヘッダーを含める
  - 出典: [speakeasy.com - API Design Pagination](https://www.speakeasy.com/api-design/pagination)

### フィルタリングとソート

- **クエリパラメータでフィルタリングとソートをサポート**: そのような操作がデータベースでインデックスされていることを確認。許可されたフィルターとソートフィールドをドキュメント化
  - 出典: [moldstud.com - Implementing RESTful APIs in Go](https://moldstud.com/articles/p-implementing-restful-apis-in-go-best-practices-and-performance-optimization)

---

## セキュリティ

### HTTPS/TLS

- **常にHTTPS（TLS）を使用**: すべての通信でHTTPSを使用。本番環境ではプレーンHTTPをサポートしない
  - 出典: [dev.to - Best Practices for Securing REST APIs](https://dev.to/adityabhuyan/best-practices-for-securing-rest-apis-balancing-performance-usability-and-security-106h)

### レート制限

- **レート制限とスロットリングを実装**: 悪用を防ぐため、IP、ユーザー、エンドポイントごとにレート制限を実装。APIゲートウェイやミドルウェアの使用を検討
  - 出典: [stackhawk.com - REST API Security Best Practices](https://www.stackhawk.com/blog/rest-api-security-best-practices/)

### APIゲートウェイ

- **APIゲートウェイ/リバースプロキシを使用**: 認証、ロギング、メトリクス、レート制限などの横断的関心事を一元化。バックエンドサービスがビジネスロジックに集中できるようにする
  - 出典: [dev.to - Best Practices for Securing REST APIs](https://dev.to/adityabhuyan/best-practices-for-securing-rest-apis-balancing-performance-usability-and-security-106h)

### データ暗号化

- **保存時の暗号化**: 機密フィールドを保存する場合は暗号化。ペイロードの移動時は、転送暗号化を確保
  - 出典: [dev.to - Best Practices for Securing REST APIs](https://dev.to/adityabhuyan/best-practices-for-securing-rest-apis-balancing-performance-usability-and-security-106h)

---

## パフォーマンス最適化

### データベース接続プール

- **`sql.DB` は接続プールマネージャー**: `sql.DB` は直接のデータベース接続ではなく、複数の基盤接続を安全に共有するプールマネージャー
  - 出典: [go.dev - Database Connection Management](https://go.dev/doc/database/manage-connections)

### プール設定

主要なパラメータ:

| 関数 | 目的 |
|------|------|
| `SetMaxOpenConns(n)` | 最大接続数（開いている + アイドル）。DBの過負荷を防ぐ |
| `SetMaxIdleConns(n)` | キューに保持できるアイドル接続数。トラフィックバーストを低レイテンシで処理 |
| `SetConnMaxIdleTime(d)` | 接続がアイドル状態でいられる時間。プールサイズを超えていなくても閉じられる |
| `SetConnMaxLifetime(d)` | 接続の最大総寿命。ロードバランサー/ファイアウォールの背後で古い接続を防ぐ |

推奨されるデフォルト値（開始点）:
- `MaxOpenConns`: 25-100（DB容量、Goインスタンス数に依存）
- `MaxIdleConns`: `MaxOpenConns` の約20-50%、または高トラフィックサービスでは等しい
- `ConnMaxIdleTime`: 1-5分（低調時にアイドル接続を削除）
- `ConnMaxLifetime`: ネットワークアイドルタイムアウト/プロキシタイムアウトより短く設定（例: インフラが10分でアイドルをキルする場合、5分に設定）

- 出典: [go.dev - Database Connection Management](https://go.dev/doc/database/manage-connections)
- 出典: [akemara.medium.com - Production Grade Guide to Golang Database Connection Management](https://akemara.medium.com/a-production-grade-guide-to-golang-database-connection-management-with-mysql-mariadb-6b00189ec25a)
- 出典: [go-database-sql.org - Connection Pool](https://go-database-sql.org/connection-pool.html)

### プロファイリングとモニタリング

- **`db.Stats()` を定期的に使用**: `OpenConnections`, `InUse`, `Idle`, `WaitCount`, `WaitDuration`, `MaxIdleClosed`, `MaxLifetimeClosed` などの値を検査。`WaitCount` の急増や `InUse == MaxOpenConns` の継続は、容量不足やリークを示唆
  - 出典: [dev.to - Mastering Database Connection Pooling in Go](https://dev.to/aaravjoshi/mastering-database-connection-pooling-in-go-performance-best-practices-4mic)

### アプリケーションレベルのパターン

- **接続ウォーミング**: トラフィックを提供する前に（特にデプロイ後やスケールアップ後）、各ワーカーでpingまたは簡単なクエリを実行してアイドルプールをウォームアップ
  - 出典: [dev.to - Mastering Database Connection Pooling in Go](https://dev.to/aaravjoshi/mastering-database-connection-pooling-in-go-performance-best-practices-4mic)

- **プリペアドステートメント**: 頻繁に使用されるクエリでは、一度準備（`db.Prepare`）して `*sql.Stmt` を再利用し、繰り返しのパース/プランオーバーヘッドを回避
  - 出典: [akemara.medium.com - Production Grade Guide to Golang Database Connection Management](https://akemara.medium.com/a-production-grade-guide-to-golang-database-connection-management-with-mysql-mariadb-6b00189ec25a)

- **トランザクション処理**: 明示的に `Begin`/`Commit` または `Rollback`。トランザクションは閉じられるまで接続を予約するため、長時間実行されるトランザクションはプールの使用をブロック
  - 出典: [dev.to - Mastering Database Connection Pooling in Go](https://dev.to/aaravjoshi/mastering-database-connection-pooling-in-go-performance-best-practices-4mic)

### よくある問題と回避方法

| 問題 | 根本原因 | 緩和策 |
|------|---------|--------|
| **接続リーク** | `rows.Close()` を失敗、またはトランザクションが `Commit/Rollback` されない: 接続が解放されない | `defer rows.Close()` をエラーチェック後すぐに使用。トランザクションでは常に `defer tx.Rollback()` |
| **プールが大きすぎる** | データベースの最大接続数を超える; 多くのDBプロセス; システム全体のリソース過負荷 | サービスインスタンス数 + データベースサーバー制限を考慮して `MaxOpenConns` を調整 |
| **古い接続** | プロキシ/ファイアウォールがアイドルソケットをキル; 閉じられたまたはリセットされた接続の再利用がエラーにつながる | `SetConnMaxLifetime` を使用して外部タイムアウトの前に閉じる/リサイクル |
| **プールが不足** | プールがワークロードに対して小さすぎる → 高レイテンシと待機時間 | `WaitCount`, `WaitDuration` を観察。現実的な並行性でロードテスト |

- 出典: [akemara.medium.com - Production Grade Guide to Golang Database Connection Management](https://akemara.medium.com/a-production-grade-guide-to-golang-database-connection-management-with-mysql-mariadb-6b00189ec25a)
- 出典: [dev.to - Mastering Database Connection Pooling in Go](https://dev.to/aaravjoshi/mastering-database-connection-pooling-in-go-performance-best-practices-4mic)

---

## テストとドキュメント

### テスト

- **様々なレベルでテストを記述**:
  - ビジネスロジックのユニットテスト
  - ハンドラーテスト（Goの `net/http/httptest` パッケージを使用）
  - 統合/エンドツーエンドテストで実際のリクエストをシミュレート
  - 出典: [Reddit - r/golang](https://www.reddit.com/r/golang/comments/trpu28)

### ドキュメント

- **OpenAPI / Swagger を使用**: APIをドキュメント化するためのツールを使用
  - リクエスト/レスポンススキーマを定義
  - すべてのエンドポイント、パラメータ、エラーレスポンスをドキュメント化
  - APIクライアント/仕様を生成可能にする
  - 出典: [infinitejs.com - Common Pitfalls Go REST APIs](https://infinitejs.com/posts/common-pitfalls-go-rest-apis/)

- **Swagger/OpenAPI ツール**:
  - **go-swagger**: Swagger 2.0 / OpenAPI 2.0 を実装。サーバー、クライアント、仕様を生成、検証
  - **swaggo / swag**: コード内の特別にフォーマットされたコメント（「アノテーション」）からSwaggerドキュメントを生成
  - 出典: [goswagger.io](https://goswagger.io/go-swagger/)
  - 出典: [github.com/swaggo/swag](https://github.com/swaggo/swag)
  - 出典: [blog.logrocket.com - Documenting Go Web APIs with Swag](https://blog.logrocket.com/documenting-go-web-apis-with-swag/)

- **ドキュメントに例を含める**: サンプルリクエストとレスポンス、エッジケース、ページネーション例、エラーケースを含める

---

## ミドルウェア

### ハンドラー構造

- **ハンドラー構造体を使用**: スタンドアロン関数の代わりに構造体を使用

```go
type UserHandler struct {
    userService UserService
}

func (h *UserHandler) RegisterRoutes(router Router) {
    router.GET("/users", h.GetUsers)
    router.POST("/users", h.CreateUser)
}
```

これにより、依存性注入（DB、サービスなど）、テスト、論理的なグループ化が容易になる
- 出典: [Medium - Stop Creating Hundreds of Handlers](https://elsyarifx.medium.com/stop-creating-hundreds-of-handlers-simplifying-go-api-routing-with-efficient-patterns-a42540224925)

- **シンなハンドラー**: ハンドラーは主に入力のパース、サービス/ビジネスロジックへの呼び出し、レスポンスのマーシャリングを行う。HTTPハンドラーにビジネスロジックを持たせない
- 出典: [Reddit - r/golang](https://www.reddit.com//r/golang/comments/1nrw741)

### ミドルウェアの使用

横断的関心事にミドルウェアを使用:
- ロギング、トレーシング
- 認証/認可
- リクエストID / 相関
- CORS、レート制限、リクエスト検証

グループ/サブルーターレベルでミドルウェアのチェーンまたはネストを許可するルーターを使用
- 出典: [codezup.com - Building Scalable REST API with Go](https://codezup.com/building-scalable-rest-api-with-go/)

### 構造化ロギング

- **構造化ロギングを使用**: プレーンテキストではなく、構造化ロギング（"key=value" または JSON）を使用。リクエストID、ユーザーIDなどのコンテキストを含める

### モニタリングとオブザーバビリティ

- **メトリクスを公開**: Prometheusなどを使用してメトリクスを公開し、レイテンシ、エラー率を追跡

---

## チェックリスト

Go REST APIを実装する際のチェックリスト:

| 領域 | 項目 |
|------|------|
| **ルーティング & URL** | ✓ リソース名詞、複数形、kebab-case、バージョン管理 |
| **入力 & モデル** | ✓ すべての入力を検証、構造体を使用、JSONタグ |
| **エラーハンドリング** | ✓ エラーを返す、ラップ/`errors.Is` & `As` を使用、集中エラーレイヤー、リークなし |
| **リストエンドポイント** | ✓ ページネーション、フィルタリング、ソート、メタデータ |
| **セキュリティ** | ✓ HTTPS、認証 & 認可、レート制限、ゲートウェイ、データ暗号化 |
| **パフォーマンス** | ✓ 効率的なクエリ、ページサイズを制限、必要に応じてキャッシュを使用 |
| **テスト & ドキュメント** | ✓ ユニット + 統合テスト、OpenAPI仕様、例、バージョニングドキュメント |
| **オブザーバビリティ** | ✓ ロギング、メトリクス、トレーシング |
| **コンテキスト** | ✓ すべてのI/O操作でコンテキストを使用、タイムアウト設定、キャンセルを尊重 |
| **データベース** | ✓ 接続プール設定、プリペアドステートメント、トランザクション管理 |

---

## 参考資料

### 公式ドキュメント

- [Go Official Blog - Context](https://go.dev/blog/context)
- [Go Official Documentation - Database Connection Management](https://go.dev/doc/database/manage-connections)

### コミュニティリソース

- [golang-standards/project-layout](https://github.com/golang-standards/project-layout)
- [goa.design - HTTP Routing](https://www.goa.design/docs/4-concepts/3-http/2-routing/)
- [goswagger.io](https://goswagger.io/go-swagger/)

### 記事とガイド

- [dev.to - REST API Best Practices](https://dev.to/george_pollock/rest-api-best-practices-why-s-and-how-s-304n)
- [infinitejs.com - Common Pitfalls Go REST APIs](https://infinitejs.com/posts/common-pitfalls-go-rest-apis/)
- [blog.marcnuri.com - Error Handling Best Practices in Go](https://blog.marcnuri.com/error-handling-best-practices-in-go)
- [leapcell.io - The Power of Context.Context in Go Microservices](https://leapcell.io/blog/the-power-of-context-context-in-go-microservices)
- [dev.to - Mastering Database Connection Pooling in Go](https://dev.to/aaravjoshi/mastering-database-connection-pooling-in-go-performance-best-practices-4mic)
- [blog.logrocket.com - Documenting Go Web APIs with Swag](https://blog.logrocket.com/documenting-go-web-apis-with-swag/)

---

**最終更新**: 2025年1月

このドキュメントは、Goコミュニティのベストプラクティスと2024-2025年の最新情報に基づいて作成されています。
