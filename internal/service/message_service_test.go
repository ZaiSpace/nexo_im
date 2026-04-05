package service

import (
	"encoding/json"
	"testing"

	"github.com/mbeoliero/nexo/internal/entity"
	"github.com/mbeoliero/nexo/pkg/constant"
)

func TestValidateMessageContentRejectsMismatchedPayload(t *testing.T) {
	err := validateMessageContent(constant.MsgTypeText, entity.MessageContent{
		Image: &entity.ImageContent{Url: "https://example.com/a.png"},
	})
	if err == nil {
		t.Fatal("expected validation error for mismatched payload")
	}
}

func TestValidateMessageContentAcceptsCustomPayload(t *testing.T) {
	err := validateMessageContent(constant.MsgTypeCustom, entity.MessageContent{
		Custom: json.RawMessage(`{"biz":"ok"}`),
	})
	if err != nil {
		t.Fatalf("expected custom payload to be valid, got %v", err)
	}
}
