package service

import (
	"context"

	"github.com/ZaiSpace/nexo_im/internal/entity"
	"github.com/ZaiSpace/nexo_im/internal/repository"
	"github.com/ZaiSpace/nexo_im/pkg/errcode"
	"github.com/mbeoliero/kit/log"
)

// ConversationService handles conversation-related business logic
type ConversationService struct {
	convRepo *repository.ConversationRepo
	msgRepo  *repository.MessageRepo
	seqRepo  *repository.SeqRepo
	repos    *repository.Repositories
}

const (
	DefaultConversationListLimit = 20
	MaxConversationListLimit     = 100
)

// ConversationListCursor is the cursor for conversation list pagination.
type ConversationListCursor struct {
	UpdatedAt      int64  `json:"updated_at"`
	ConversationId string `json:"conversation_id"`
}

// ConversationListResult is the paginated conversation list result.
type ConversationListResult struct {
	List       []*entity.ConversationInfo `json:"list"`
	HasMore    bool                       `json:"has_more"`
	NextCursor *ConversationListCursor    `json:"next_cursor,omitempty"`
}

// NewConversationService creates a new ConversationService
func NewConversationService(repos *repository.Repositories) *ConversationService {
	return &ConversationService{
		convRepo: repos.Conversation,
		msgRepo:  repos.Message,
		seqRepo:  repos.Seq,
		repos:    repos,
	}
}

// GetAllUserConversations gets all conversations for a user.
// withLastMessage controls whether to include the latest message for each conversation.
func (s *ConversationService) GetAllUserConversations(ctx context.Context, userId string, withLastMessage bool) ([]*entity.ConversationInfo, error) {
	convWithSeqs, err := s.convRepo.GetUserConversationsWithSeq(ctx, userId)
	if err != nil {
		log.CtxError(ctx, "get user conversations failed: user_id=%s, error=%v", userId, err)
		return nil, errcode.ErrInternalServer
	}
	return s.buildConversationInfos(ctx, userId, convWithSeqs, withLastMessage)
}

// GetUserConversationsPage gets conversations for a user with cursor pagination.
func (s *ConversationService) GetUserConversationsPage(ctx context.Context, userId string, withLastMessage bool, limit int, cursorUpdatedAt int64, cursorConversationId string) (*ConversationListResult, error) {
	if limit <= 0 {
		limit = DefaultConversationListLimit
	}
	if limit > MaxConversationListLimit {
		limit = MaxConversationListLimit
	}

	convWithSeqs, err := s.convRepo.GetUserConversationsWithSeqPage(ctx, userId, limit+1, cursorUpdatedAt, cursorConversationId)
	if err != nil {
		log.CtxError(ctx, "get user conversations failed: user_id=%s, error=%v", userId, err)
		return nil, errcode.ErrInternalServer
	}

	hasMore := len(convWithSeqs) > limit
	if hasMore {
		convWithSeqs = convWithSeqs[:limit]
	}

	list, err := s.buildConversationInfos(ctx, userId, convWithSeqs, withLastMessage)
	if err != nil {
		return nil, err
	}

	var nextCursor *ConversationListCursor
	if hasMore && len(convWithSeqs) > 0 {
		last := convWithSeqs[len(convWithSeqs)-1]
		nextCursor = &ConversationListCursor{
			UpdatedAt:      last.UpdatedAt,
			ConversationId: last.ConversationId,
		}
	}

	return &ConversationListResult{
		List:       list,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}

func (s *ConversationService) buildConversationInfos(ctx context.Context, userId string, convWithSeqs []*entity.ConversationWithSeq, withLastMessage bool) ([]*entity.ConversationInfo, error) {
	lastMsgMap := make(map[string]*entity.Message)
	if withLastMessage {
		convMaxSeq := make(map[string]int64, len(convWithSeqs))
		for _, conv := range convWithSeqs {
			if conv.MaxSeq > 0 {
				convMaxSeq[conv.ConversationId] = conv.MaxSeq
			}
		}

		var err error
		lastMsgMap, err = s.msgRepo.BatchGetByConvSeq(ctx, convMaxSeq)
		if err != nil {
			log.CtxError(ctx, "batch get last messages failed: user_id=%s, error=%v", userId, err)
			return nil, errcode.ErrInternalServer
		}
	}

	list := make([]*entity.ConversationInfo, 0, len(convWithSeqs))
	for _, conv := range convWithSeqs {
		var lastMsg *entity.MessageInfo
		if msg := lastMsgMap[conv.ConversationId]; msg != nil {
			lastMsg = msg.ToMessageInfo()
		}

		info := &entity.ConversationInfo{
			ConversationId:   conv.ConversationId,
			ConversationType: conv.ConversationType,
			PeerUserId:       conv.PeerUserId,
			GroupId:          conv.GroupId,
			RecvMsgOpt:       conv.RecvMsgOpt,
			IsPinned:         conv.IsPinned,
			UnreadCount:      conv.UnreadCount,
			MaxSeq:           conv.MaxSeq,
			ReadSeq:          conv.ReadSeq,
			UpdatedAt:        conv.UpdatedAt,
			LastMessage:      lastMsg,
		}
		list = append(list, info)
	}

	return list, nil
}

// GetConversation gets a specific conversation for a user
func (s *ConversationService) GetConversation(ctx context.Context, userId, conversationId string) (*entity.ConversationInfo, error) {
	conv, err := s.convRepo.GetByOwnerAndConvId(ctx, userId, conversationId)
	if err != nil {
		log.CtxError(ctx, "get conversation failed: user_id=%s, conversation_id=%s, error=%v", userId, conversationId, err)
		return nil, errcode.ErrInternalServer
	}
	if conv == nil {
		return nil, errcode.ErrConvNotFound
	}

	// Get seq info
	seqConv, _ := s.seqRepo.GetConversationSeqInfo(ctx, conversationId)
	seqUser, _ := s.seqRepo.GetSeqUser(ctx, userId, conversationId)

	maxSeq := int64(0)
	readSeq := int64(0)
	if seqConv != nil {
		maxSeq = seqConv.MaxSeq
	}
	if seqUser != nil {
		readSeq = seqUser.ReadSeq
	}

	unreadCount := maxSeq - readSeq
	if unreadCount < 0 {
		unreadCount = 0
	}

	return &entity.ConversationInfo{
		ConversationId:   conv.ConversationId,
		ConversationType: conv.ConversationType,
		PeerUserId:       conv.PeerUserId,
		GroupId:          conv.GroupId,
		RecvMsgOpt:       conv.RecvMsgOpt,
		IsPinned:         conv.IsPinned,
		UnreadCount:      unreadCount,
		MaxSeq:           maxSeq,
		ReadSeq:          readSeq,
		UpdatedAt:        conv.UpdatedAt,
	}, nil
}

// UpdateConversationRequest represents update conversation request
type UpdateConversationRequest struct {
	RecvMsgOpt *int32 `json:"recv_msg_opt,omitempty"`
	IsPinned   *bool  `json:"is_pinned,omitempty"`
}

// UpdateConversation updates conversation settings
func (s *ConversationService) UpdateConversation(ctx context.Context, userId, conversationId string, req *UpdateConversationRequest) error {
	updates := make(map[string]interface{})
	if req.RecvMsgOpt != nil {
		updates["recv_msg_opt"] = *req.RecvMsgOpt
	}
	if req.IsPinned != nil {
		updates["is_pinned"] = *req.IsPinned
	}

	if len(updates) == 0 {
		return nil
	}

	if err := s.convRepo.Update(ctx, userId, conversationId, updates); err != nil {
		log.CtxError(ctx, "update conversation failed: %v", err)
		return errcode.ErrInternalServer
	}

	return nil
}

// MarkRead marks a conversation as read up to a seq
func (s *ConversationService) MarkRead(ctx context.Context, userId, conversationId string, readSeq int64) error {
	if err := s.seqRepo.UpdateReadSeq(ctx, userId, conversationId, readSeq); err != nil {
		log.CtxError(ctx, "update read seq failed: %v", err)
		return errcode.ErrInternalServer
	}
	return nil
}

// GetMaxReadSeq gets the max seq and read seq for a conversation
func (s *ConversationService) GetMaxReadSeq(ctx context.Context, userId, conversationId string) (maxSeq, readSeq int64, err error) {
	seqConv, err := s.seqRepo.GetConversationSeqInfo(ctx, conversationId)
	if err != nil {
		return 0, 0, err
	}

	seqUser, _ := s.seqRepo.GetSeqUser(ctx, userId, conversationId)
	if seqUser != nil {
		readSeq = seqUser.ReadSeq
	}

	return seqConv.MaxSeq, readSeq, nil
}
