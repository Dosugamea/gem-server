package handler

import (
	"context"
	"errors"
	"strconv"

	redemptionapp "gem-server/internal/application/code_redemption"
	currencyapp "gem-server/internal/application/currency"
	historyapp "gem-server/internal/application/history"
	paymentapp "gem-server/internal/application/payment"
	"gem-server/internal/domain/currency"
	"gem-server/internal/domain/payment_request"
	"gem-server/internal/domain/redemption_code"
	"gem-server/internal/domain/transaction"
	"gem-server/internal/presentation/grpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CurrencyHandler gRPC通貨サービスハンドラー
type CurrencyHandler struct {
	pb.UnimplementedCurrencyServiceServer
	currencyService   *currencyapp.CurrencyApplicationService
	paymentService    *paymentapp.PaymentApplicationService
	redemptionService *redemptionapp.CodeRedemptionApplicationService
	historyService    *historyapp.HistoryApplicationService
}

// NewCurrencyHandler 新しいCurrencyHandlerを作成
func NewCurrencyHandler(
	currencyService *currencyapp.CurrencyApplicationService,
	paymentService *paymentapp.PaymentApplicationService,
	redemptionService *redemptionapp.CodeRedemptionApplicationService,
	historyService *historyapp.HistoryApplicationService,
) *CurrencyHandler {
	return &CurrencyHandler{
		currencyService:   currencyService,
		paymentService:    paymentService,
		redemptionService: redemptionService,
		historyService:    historyService,
	}
}

// GetBalance 残高取得
func (h *CurrencyHandler) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	appReq := &currencyapp.GetBalanceRequest{
		UserID: req.UserId,
	}

	appResp, err := h.currencyService.GetBalance(ctx, appReq)
	if err != nil {
		return nil, h.handleError(err)
	}

	// レスポンスを構築
	balances := make(map[string]string)
	balances["paid"] = strconv.FormatInt(appResp.Balances["paid"], 10)
	balances["free"] = strconv.FormatInt(appResp.Balances["free"], 10)

	return &pb.GetBalanceResponse{
		UserId:   appResp.UserID,
		Balances: balances,
	}, nil
}

// Grant 通貨付与
func (h *CurrencyHandler) Grant(ctx context.Context, req *pb.GrantRequest) (*pb.GrantResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.CurrencyType == "" {
		return nil, status.Error(codes.InvalidArgument, "currency_type is required")
	}
	if req.Amount == "" {
		return nil, status.Error(codes.InvalidArgument, "amount is required")
	}

	// 金額をint64に変換
	amount, err := strconv.ParseInt(req.Amount, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid amount format")
	}

	// metadataをmap[string]interface{}に変換
	metadata := make(map[string]interface{})
	for k, v := range req.Metadata {
		metadata[k] = v
	}

	appReq := &currencyapp.GrantRequest{
		UserID:       req.UserId,
		CurrencyType: req.CurrencyType,
		Amount:       amount,
		Reason:       req.Reason,
		Metadata:     metadata,
	}

	appResp, err := h.currencyService.Grant(ctx, appReq)
	if err != nil {
		return nil, h.handleError(err)
	}

	return &pb.GrantResponse{
		TransactionId: appResp.TransactionID,
		BalanceAfter:  strconv.FormatInt(appResp.BalanceAfter, 10),
		Status:        appResp.Status,
	}, nil
}

// Consume 通貨消費
func (h *CurrencyHandler) Consume(ctx context.Context, req *pb.ConsumeRequest) (*pb.ConsumeResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.CurrencyType == "" {
		return nil, status.Error(codes.InvalidArgument, "currency_type is required")
	}
	if req.Amount == "" {
		return nil, status.Error(codes.InvalidArgument, "amount is required")
	}

	// 金額をint64に変換
	amount, err := strconv.ParseInt(req.Amount, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid amount format")
	}

	// metadataをmap[string]interface{}に変換
	metadata := make(map[string]interface{})
	for k, v := range req.Metadata {
		metadata[k] = v
	}

	appReq := &currencyapp.ConsumeRequest{
		UserID:       req.UserId,
		CurrencyType: req.CurrencyType,
		Amount:       amount,
		ItemID:       req.ItemId,
		UsePriority:  req.UsePriority,
		Metadata:     metadata,
	}

	var appResp *currencyapp.ConsumeResponse
	if req.UsePriority || req.CurrencyType == "auto" {
		// 優先順位制御を使用
		appResp, err = h.currencyService.ConsumeWithPriority(ctx, appReq)
	} else {
		// 単一通貨タイプで消費
		appResp, err = h.currencyService.Consume(ctx, appReq)
	}

	if err != nil {
		return nil, h.handleError(err)
	}

	// レスポンスを構築
	resp := &pb.ConsumeResponse{
		TransactionId: appResp.TransactionID,
		Status:        appResp.Status,
	}

	if len(appResp.ConsumptionDetails) > 0 {
		// 優先順位制御使用時
		details := make([]*pb.ConsumptionDetail, len(appResp.ConsumptionDetails))
		for i, detail := range appResp.ConsumptionDetails {
			details[i] = &pb.ConsumptionDetail{
				CurrencyType:  detail.CurrencyType,
				Amount:        strconv.FormatInt(detail.Amount, 10),
				BalanceBefore: strconv.FormatInt(detail.BalanceBefore, 10),
				BalanceAfter:  strconv.FormatInt(detail.BalanceAfter, 10),
			}
		}
		resp.ConsumptionDetails = details
		resp.TotalConsumed = strconv.FormatInt(appResp.TotalConsumed, 10)
	} else {
		// 単一通貨タイプ消費時
		resp.BalanceAfter = strconv.FormatInt(appResp.BalanceAfter, 10)
	}

	return resp, nil
}

// ProcessPayment 決済処理
func (h *CurrencyHandler) ProcessPayment(ctx context.Context, req *pb.ProcessPaymentRequest) (*pb.ProcessPaymentResponse, error) {
	if req.PaymentRequestId == "" {
		return nil, status.Error(codes.InvalidArgument, "payment_request_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Amount == "" {
		return nil, status.Error(codes.InvalidArgument, "amount is required")
	}

	// 金額をint64に変換
	amount, err := strconv.ParseInt(req.Amount, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid amount format")
	}

	appReq := &paymentapp.ProcessPaymentRequest{
		PaymentRequestID: req.PaymentRequestId,
		UserID:           req.UserId,
		MethodName:       req.MethodName,
		Details:          req.Details,
		Amount:           amount,
		Currency:         req.Currency,
	}

	appResp, err := h.paymentService.ProcessPayment(ctx, appReq)
	if err != nil {
		return nil, h.handleError(err)
	}

	// レスポンスを構築
	details := make([]*pb.ConsumptionDetail, len(appResp.ConsumptionDetails))
	for i, detail := range appResp.ConsumptionDetails {
		details[i] = &pb.ConsumptionDetail{
			CurrencyType:  detail.CurrencyType,
			Amount:        strconv.FormatInt(detail.Amount, 10),
			BalanceBefore: strconv.FormatInt(detail.BalanceBefore, 10),
			BalanceAfter:  strconv.FormatInt(detail.BalanceAfter, 10),
		}
	}

	return &pb.ProcessPaymentResponse{
		TransactionId:      appResp.TransactionID,
		PaymentRequestId:   appResp.PaymentRequestID,
		ConsumptionDetails: details,
		TotalConsumed:      strconv.FormatInt(appResp.TotalConsumed, 10),
		Status:             appResp.Status,
	}, nil
}

// RedeemCode コード引き換え
func (h *CurrencyHandler) RedeemCode(ctx context.Context, req *pb.RedeemCodeRequest) (*pb.RedeemCodeResponse, error) {
	if req.Code == "" {
		return nil, status.Error(codes.InvalidArgument, "code is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	appReq := &redemptionapp.RedeemCodeRequest{
		Code:   req.Code,
		UserID: req.UserId,
	}

	appResp, err := h.redemptionService.Redeem(ctx, appReq)
	if err != nil {
		return nil, h.handleError(err)
	}

	return &pb.RedeemCodeResponse{
		RedemptionId:  appResp.RedemptionID,
		TransactionId: appResp.TransactionID,
		Code:          appResp.Code,
		CurrencyType:  appResp.CurrencyType,
		Amount:        strconv.FormatInt(appResp.Amount, 10),
		BalanceAfter:  strconv.FormatInt(appResp.BalanceAfter, 10),
		Status:        appResp.Status,
	}, nil
}

// GetTransactionHistory トランザクション履歴取得
func (h *CurrencyHandler) GetTransactionHistory(ctx context.Context, req *pb.GetTransactionHistoryRequest) (*pb.GetTransactionHistoryResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 50 // デフォルト値
	}
	if limit > 100 {
		limit = 100 // 最大値
	}

	offset := int(req.Offset)
	if offset < 0 {
		offset = 0
	}

	appReq := &historyapp.GetTransactionHistoryRequest{
		UserID:          req.UserId,
		Limit:           limit,
		Offset:          offset,
		CurrencyType:    req.CurrencyType,
		TransactionType: req.TransactionType,
	}

	appResp, err := h.historyService.GetTransactionHistory(ctx, appReq)
	if err != nil {
		return nil, h.handleError(err)
	}

	// トランザクションをレスポンス形式に変換
	transactions := make([]*pb.Transaction, len(appResp.Transactions))
	for i, txn := range appResp.Transactions {
		transactions[i] = &pb.Transaction{
			TransactionId:   txn.TransactionID(),
			TransactionType: txn.TransactionType().String(),
			CurrencyType:    txn.CurrencyType().String(),
			Amount:          strconv.FormatInt(txn.Amount(), 10),
			BalanceBefore:   strconv.FormatInt(txn.BalanceBefore(), 10),
			BalanceAfter:    strconv.FormatInt(txn.BalanceAfter(), 10),
			Status:          txn.Status().String(),
			CreatedAt:       txn.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return &pb.GetTransactionHistoryResponse{
		Transactions: transactions,
		Total:        int32(appResp.Total),
		Limit:        int32(appResp.Limit),
		Offset:       int32(appResp.Offset),
	}, nil
}

// handleError エラーをgRPCステータスコードに変換
func (h *CurrencyHandler) handleError(err error) error {
	// ドメインエラーの判定と処理
	if errors.Is(err, currency.ErrInsufficientBalance) {
		return status.Error(codes.FailedPrecondition, err.Error())
	}

	if errors.Is(err, currency.ErrInvalidAmount) {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	if errors.Is(err, currency.ErrCurrencyNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}

	if errors.Is(err, transaction.ErrTransactionNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}

	if errors.Is(err, payment_request.ErrPaymentRequestNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}

	if errors.Is(err, payment_request.ErrPaymentRequestAlreadyProcessed) {
		return status.Error(codes.FailedPrecondition, err.Error())
	}

	if errors.Is(err, redemption_code.ErrCodeNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}

	if errors.Is(err, redemption_code.ErrCodeNotRedeemable) {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	if errors.Is(err, redemption_code.ErrCodeAlreadyUsed) {
		return status.Error(codes.AlreadyExists, err.Error())
	}

	if errors.Is(err, redemption_code.ErrUserAlreadyRedeemed) {
		return status.Error(codes.AlreadyExists, err.Error())
	}

	// gRPCステータスエラーの場合はそのまま返す
	if _, ok := status.FromError(err); ok {
		return err
	}

	// 予期しないエラー
	return status.Error(codes.Internal, "internal server error")
}
