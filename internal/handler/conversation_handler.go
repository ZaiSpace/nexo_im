package handler

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/ZaiSpace/nexo_im/internal/middleware"
	"github.com/ZaiSpace/nexo_im/internal/service"
	"github.com/ZaiSpace/nexo_im/pkg/errcode"
	"github.com/ZaiSpace/nexo_im/pkg/response"
)

// GetAllConversationListRequest represents conversation list request options.
type GetAllConversationListRequest struct {
	WithLastMessage *bool `json:"with_last_message" query:"with_last_message"`
}

// GetConversationListRequest represents conversation list page request options.
type GetConversationListRequest struct {
	WithLastMessage      *bool  `json:"with_last_message" query:"with_last_message"`
	Limit                int    `json:"limit" query:"limit"`
	CursorUpdatedAt      int64  `json:"cursor_updated_at" query:"cursor_updated_at"`
	CursorConversationId string `json:"cursor_conversation_id" query:"cursor_conversation_id"`
}

// ConversationHandler handles conversation-related requests
type ConversationHandler struct {
	convService *service.ConversationService
}

// NewConversationHandler creates a new ConversationHandler
func NewConversationHandler(convService *service.ConversationService) *ConversationHandler {
	return &ConversationHandler{convService: convService}
}

// GetAllConversationList handles get conversation list request
func (h *ConversationHandler) GetAllConversationList(ctx context.Context, c *app.RequestContext) {
	userId := middleware.GetUserId(c)
	if userId == "" {
		response.ErrorWithCode(ctx, c, errcode.ErrUnauthorized)
		return
	}

	// By default do not include latest message to reduce payload.
	withLastMessage := false
	var req GetAllConversationListRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.ErrorWithCode(ctx, c, errcode.ErrInvalidParam)
		return
	}
	if req.WithLastMessage != nil {
		withLastMessage = *req.WithLastMessage
	}

	convs, err := h.convService.GetAllUserConversations(ctx, userId, withLastMessage)
	if err != nil {
		response.Error(ctx, c, err)
		return
	}

	response.Success(ctx, c, convs)
}

// GetConversationList handles paginated conversation list request.
func (h *ConversationHandler) GetConversationList(ctx context.Context, c *app.RequestContext) {
	userId := middleware.GetUserId(c)
	if userId == "" {
		response.ErrorWithCode(ctx, c, errcode.ErrUnauthorized)
		return
	}

	withLastMessage := false
	var req GetConversationListRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.ErrorWithCode(ctx, c, errcode.ErrInvalidParam)
		return
	}
	if req.WithLastMessage != nil {
		withLastMessage = *req.WithLastMessage
	}

	if req.Limit < 0 || req.Limit > service.MaxConversationListLimit {
		response.ErrorWithCode(ctx, c, errcode.ErrInvalidParam)
		return
	}
	if req.CursorUpdatedAt > 0 && req.CursorConversationId == "" {
		response.ErrorWithCode(ctx, c, errcode.ErrInvalidParam)
		return
	}
	if req.CursorConversationId != "" && req.CursorUpdatedAt <= 0 {
		response.ErrorWithCode(ctx, c, errcode.ErrInvalidParam)
		return
	}

	convs, err := h.convService.GetUserConversationsPage(
		ctx,
		userId,
		withLastMessage,
		req.Limit,
		req.CursorUpdatedAt,
		req.CursorConversationId,
	)
	if err != nil {
		response.Error(ctx, c, err)
		return
	}

	response.Success(ctx, c, convs)
}

// GetConversation handles get single conversation request
func (h *ConversationHandler) GetConversation(ctx context.Context, c *app.RequestContext) {
	userId := middleware.GetUserId(c)
	if userId == "" {
		response.ErrorWithCode(ctx, c, errcode.ErrUnauthorized)
		return
	}

	conversationId := c.Query("conversation_id")
	if conversationId == "" {
		response.ErrorWithCode(ctx, c, errcode.ErrInvalidParam)
		return
	}

	conv, err := h.convService.GetConversation(ctx, userId, conversationId)
	if err != nil {
		response.Error(ctx, c, err)
		return
	}

	response.Success(ctx, c, conv)
}

// UpdateConversation handles update conversation settings request
func (h *ConversationHandler) UpdateConversation(ctx context.Context, c *app.RequestContext) {
	userId := middleware.GetUserId(c)
	if userId == "" {
		response.ErrorWithCode(ctx, c, errcode.ErrUnauthorized)
		return
	}

	conversationId := c.Query("conversation_id")
	if conversationId == "" {
		response.ErrorWithCode(ctx, c, errcode.ErrInvalidParam)
		return
	}

	var req service.UpdateConversationRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.ErrorWithCode(ctx, c, errcode.ErrInvalidParam)
		return
	}

	if err := h.convService.UpdateConversation(ctx, userId, conversationId, &req); err != nil {
		response.Error(ctx, c, err)
		return
	}

	response.Success(ctx, c, nil)
}

// MarkReadRequest represents mark read request
type MarkReadRequest struct {
	ConversationId string `json:"conversation_id"`
	ReadSeq        int64  `json:"read_seq"`
}

// MarkRead handles mark conversation as read request
func (h *ConversationHandler) MarkRead(ctx context.Context, c *app.RequestContext) {
	userId := middleware.GetUserId(c)
	if userId == "" {
		response.ErrorWithCode(ctx, c, errcode.ErrUnauthorized)
		return
	}

	var req MarkReadRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.ErrorWithCode(ctx, c, errcode.ErrInvalidParam)
		return
	}

	if err := h.convService.MarkRead(ctx, userId, req.ConversationId, req.ReadSeq); err != nil {
		response.Error(ctx, c, err)
		return
	}

	response.Success(ctx, c, nil)
}

// GetMaxReadSeq handles get max and read seq for a conversation
func (h *ConversationHandler) GetMaxReadSeq(ctx context.Context, c *app.RequestContext) {
	userId := middleware.GetUserId(c)
	if userId == "" {
		response.ErrorWithCode(ctx, c, errcode.ErrUnauthorized)
		return
	}

	conversationId := c.Query("conversation_id")
	if conversationId == "" {
		response.ErrorWithCode(ctx, c, errcode.ErrInvalidParam)
		return
	}

	maxSeq, readSeq, err := h.convService.GetMaxReadSeq(ctx, userId, conversationId)
	if err != nil {
		response.Error(ctx, c, err)
		return
	}

	unreadCount := maxSeq - readSeq
	if unreadCount < 0 {
		unreadCount = 0
	}

	response.Success(ctx, c, map[string]interface{}{
		"max_seq":      maxSeq,
		"read_seq":     readSeq,
		"unread_count": unreadCount,
	})
}

// GetUnreadCount handles get unread count request
func (h *ConversationHandler) GetUnreadCount(ctx context.Context, c *app.RequestContext) {
	userId := middleware.GetUserId(c)
	if userId == "" {
		response.ErrorWithCode(ctx, c, errcode.ErrUnauthorized)
		return
	}

	conversationId := c.Query("conversation_id")
	if conversationId == "" {
		response.ErrorWithCode(ctx, c, errcode.ErrInvalidParam)
		return
	}

	readSeqStr := c.Query("read_seq")
	readSeq, _ := strconv.ParseInt(readSeqStr, 10, 64)

	maxSeq, currentReadSeq, err := h.convService.GetMaxReadSeq(ctx, userId, conversationId)
	if err != nil {
		response.Error(ctx, c, err)
		return
	}

	// Use provided read_seq or current read_seq
	if readSeq == 0 {
		readSeq = currentReadSeq
	}

	unreadCount := maxSeq - readSeq
	if unreadCount < 0 {
		unreadCount = 0
	}

	response.Success(ctx, c, map[string]any{
		"unread_count": unreadCount,
	})
}
