package auth

// GenerateTokenRequest トークン生成リクエスト
type GenerateTokenRequest struct {
	UserID string
}

// GenerateTokenResponse トークン生成レスポンス
type GenerateTokenResponse struct {
	Token     string
	ExpiresIn int64  // 秒単位
	TokenType string // "Bearer"
}
