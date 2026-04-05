package gateway

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ZaiSpace/nexo_im/internal/config"
	"github.com/ZaiSpace/nexo_im/internal/entity"
	"github.com/ZaiSpace/nexo_im/pkg/constant"
)

type mockClientConn struct {
	writeCount int
}

func (m *mockClientConn) ReadMessage() ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (m *mockClientConn) WriteMessage(_ []byte) error {
	m.writeCount++
	return nil
}

func (m *mockClientConn) Close() error {
	return nil
}

func (m *mockClientConn) SetReadDeadline(_ time.Time) error {
	return nil
}

func (m *mockClientConn) SetWriteDeadline(_ time.Time) error {
	return nil
}

type mockAppPushSender struct {
	calls         []*AppPushRequest
	userNameByID  map[int64]string
	lookupUserIDs []int64
}

func (m *mockAppPushSender) SendPush(_ context.Context, req *AppPushRequest) error {
	m.calls = append(m.calls, req)
	return nil
}

func (m *mockAppPushSender) GetUserDisplayName(_ context.Context, userID int64) (string, error) {
	m.lookupUserIDs = append(m.lookupUserIDs, userID)
	if m.userNameByID == nil {
		return "", nil
	}
	return m.userNameByID[userID], nil
}

func newTestWsServer() *WsServer {
	cfg := &config.Config{
		WebSocket: config.WebSocketConfig{
			PushChannelSize: 16,
		},
	}
	return NewWsServer(cfg, nil, nil, nil)
}

func newMessage(senderId, recvId string) *entity.Message {
	return &entity.Message{
		Id:             1,
		ConversationId: "si_100_200",
		Seq:            10,
		ClientMsgId:    "client-msg-id",
		SenderId:       senderId,
		RecvId:         recvId,
		SessionType:    constant.SessionTypeSingle,
		MsgType:        constant.MsgTypeText,
		Content: entity.MessageContent{
			Text: &entity.TextContent{Text: "hello"},
		},
	}
}

func TestProcessPushTask_OfflineUserTriggersAppPush(t *testing.T) {
	s := newTestWsServer()
	mockPush := &mockAppPushSender{}
	s.SetAppPushSender(mockPush)

	msg := newMessage("100", "200")
	task := &PushTask{
		Msg:       msg,
		TargetIds: []string{"100", "200"},
	}

	s.processPushTask(context.Background(), task)

	if len(mockPush.calls) != 1 {
		t.Fatalf("expected 1 app push call, got %d", len(mockPush.calls))
	}
	if mockPush.calls[0].UserId != 200 {
		t.Fatalf("expected app push target user 200, got %d", mockPush.calls[0].UserId)
	}
}

func TestProcessPushTask_OnlineUserSkipsAppPush(t *testing.T) {
	s := newTestWsServer()
	mockPush := &mockAppPushSender{}
	s.SetAppPushSender(mockPush)

	conn := &mockClientConn{}
	client := NewClient(conn, "200", constant.PlatformIdIOS, "go", "token", "conn-1", s)
	s.userMap.Register(context.Background(), client)

	msg := newMessage("100", "200")
	task := &PushTask{
		Msg:       msg,
		TargetIds: []string{"200"},
	}

	s.processPushTask(context.Background(), task)

	if conn.writeCount == 0 {
		t.Fatalf("expected websocket push to online user")
	}
	if len(mockPush.calls) != 0 {
		t.Fatalf("expected 0 app push calls for online user, got %d", len(mockPush.calls))
	}
}

func TestProcessPushTask_SenderNeverTriggersAppPush(t *testing.T) {
	s := newTestWsServer()
	mockPush := &mockAppPushSender{}
	s.SetAppPushSender(mockPush)

	msg := newMessage("100", "200")
	task := &PushTask{
		Msg:       msg,
		TargetIds: []string{"100"},
	}

	s.processPushTask(context.Background(), task)

	if len(mockPush.calls) != 0 {
		t.Fatalf("expected sender to be skipped for app push, got %d calls", len(mockPush.calls))
	}
}

func TestProcessPushTask_OfflineSingleUserUsesSenderDisplayNameInTitle(t *testing.T) {
	s := newTestWsServer()
	mockPush := &mockAppPushSender{
		userNameByID: map[int64]string{
			100: "Alice",
		},
	}
	s.SetAppPushSender(mockPush)

	msg := newMessage("100", "200")
	task := &PushTask{
		Msg:       msg,
		TargetIds: []string{"200"},
	}

	s.processPushTask(context.Background(), task)

	if len(mockPush.calls) != 1 {
		t.Fatalf("expected 1 app push call, got %d", len(mockPush.calls))
	}
	if len(mockPush.lookupUserIDs) != 1 || mockPush.lookupUserIDs[0] != 100 {
		t.Fatalf("expected lookup sender id 100 once, got %+v", mockPush.lookupUserIDs)
	}
	if got := mockPush.calls[0].Title; got != "Alice sent you a message" {
		t.Fatalf("expected title from sender display name, got %q", got)
	}
}
