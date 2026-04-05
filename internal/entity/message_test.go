package entity

import (
	"encoding/json"
	"testing"
)

func TestMessageToMessageInfoFlattensTypedContent(t *testing.T) {
	custom := json.RawMessage(`{"k":"v"}`)
	msg := &Message{
		Id:             1,
		ConversationId: "conv_1",
		Seq:            2,
		ClientMsgId:    "client_1",
		SenderId:       "user_1",
		SessionType:    1,
		MsgType:        100,
		Content: MessageContent{
			Custom: custom,
		},
		SendAt: 123,
	}

	info := msg.ToMessageInfo()
	if info.Content.Custom != `{"k":"v"}` {
		t.Fatalf("expected custom content to be flattened, got %q", info.Content.Custom)
	}
}

func TestMessageToMessageInfoFlattensTextContent(t *testing.T) {
	msg := &Message{
		Content: MessageContent{
			Text: &TextContent{Text: "hello"},
		},
	}

	info := msg.ToMessageInfo()
	if info.Content.Text != "hello" {
		t.Fatalf("expected text content to be flattened, got %q", info.Content.Text)
	}
}
