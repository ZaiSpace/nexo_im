package gateway

import (
	"testing"

	"github.com/mbeoliero/nexo/internal/entity"
)

func TestWsServerMessageToMsgDataKeepsWireShape(t *testing.T) {
	server := &WsServer{}
	msg := &entity.Message{
		Id:             1,
		ConversationId: "conv_1",
		Seq:            3,
		ClientMsgId:    "client_1",
		SenderId:       "user_1",
		SessionType:    1,
		MsgType:        1,
		Content: entity.MessageContent{
			Text: &entity.TextContent{Text: "hello"},
		},
		SendAt: 100,
	}

	data := server.messageToMsgData(msg)
	if data.Content.Text != "hello" {
		t.Fatalf("expected text content on wire, got %q", data.Content.Text)
	}
}
