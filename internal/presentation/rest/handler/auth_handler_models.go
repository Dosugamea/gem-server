package handler

// GenerateTokenRequest トークン生成リクエスト
// @Description トークン生成リクエスト
type GenerateTokenRequest struct {
	UserID string `json:"user_id" example:"user123"`
}

// GenerateTokenResponse トークン生成レスポンス
// @Description トークン生成レスポンス
type GenerateTokenResponse struct {
	Token     string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidXNlcjEyMyIsImV4cCI6MTcwMDAwMDAwMH0.signature"`
	ExpiresIn int    `json:"expires_in" example:"3600"`
	TokenType string `json:"token_type" example:"Bearer"`
}

// ErrorResponse エラーレスポンス
// @Description エラーレスポンス
type ErrorResponse struct {
	Error string `json:"error" example:"invalid request body"`
}
