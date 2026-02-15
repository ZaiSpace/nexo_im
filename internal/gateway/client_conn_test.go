package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ZaiSpace/nexo_im/internal/config"
	"github.com/ZaiSpace/nexo_im/pkg/jwt"
)

func makeWebSocketPair(t *testing.T) (*websocket.Conn, *websocket.Conn, func()) {
	t.Helper()

	serverConnCh := make(chan *websocket.Conn, 1)
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}
		serverConnCh <- conn
	}))

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		srv.Close()
		t.Fatalf("dial failed: %v", err)
	}

	var serverConn *websocket.Conn
	select {
	case serverConn = <-serverConnCh:
	case <-time.After(2 * time.Second):
		clientConn.Close()
		srv.Close()
		t.Fatal("timeout waiting for server websocket connection")
	}

	cleanup := func() {
		_ = clientConn.Close()
		_ = serverConn.Close()
		srv.Close()
	}

	return serverConn, clientConn, cleanup
}

func TestWebsocketClientConn_BasicReadWrite(t *testing.T) {
	serverRawConn, clientConn, cleanup := makeWebSocketPair(t)
	defer cleanup()

	conn := NewWebSocketClientConn(serverRawConn, MaxMessageSize, PongWait, PingPeriod)
	defer conn.Close()

	serverToClient := []byte("hello-client")
	if err := conn.WriteMessage(serverToClient); err != nil {
		t.Fatalf("server write failed: %v", err)
	}

	if err := clientConn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set client read deadline failed: %v", err)
	}
	msgType, got, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("client read failed: %v", err)
	}
	if msgType != websocket.BinaryMessage {
		t.Fatalf("unexpected message type: got %d want %d", msgType, websocket.BinaryMessage)
	}
	if string(got) != string(serverToClient) {
		t.Fatalf("unexpected payload: got %q want %q", string(got), string(serverToClient))
	}

	clientToServer := []byte("hello-server")
	if err := clientConn.WriteMessage(websocket.BinaryMessage, clientToServer); err != nil {
		t.Fatalf("client write failed: %v", err)
	}

	got, err = conn.ReadMessage()
	if err != nil {
		t.Fatalf("server read failed: %v", err)
	}
	if string(got) != string(clientToServer) {
		t.Fatalf("unexpected payload: got %q want %q", string(got), string(clientToServer))
	}
}

func TestWebsocketClientConn_WriteAfterClose(t *testing.T) {
	serverRawConn, _, cleanup := makeWebSocketPair(t)
	defer cleanup()

	conn := NewWebSocketClientConn(serverRawConn, MaxMessageSize, PongWait, PingPeriod)
	if err := conn.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	if err := conn.WriteMessage([]byte("should-fail")); !errors.Is(err, ErrConnClosed) {
		t.Fatalf("unexpected error: got %v want %v", err, ErrConnClosed)
	}
}

func TestWsServer_HandleConnection_DirectWebSocketMessage(t *testing.T) {
	const (
		userID    = "u_ws_direct_test"
		jwtSecret = "unit-test-secret"
	)

	cfg := &config.Config{
		Server: config.ServerConfig{
			AllowedOrigins: []string{"*"},
		},
		JWT: config.JWTConfig{
			Secret:      jwtSecret,
			ExpireHours: 1,
		},
		WebSocket: config.WebSocketConfig{
			MaxConnNum:      100,
			MaxMessageSize:  MaxMessageSize,
			PushChannelSize: 8,
		},
	}
	wsServer := NewWsServer(cfg, nil, nil, nil)

	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wsServer.HandleConnection(context.Background(), w, r)
	}))
	defer httpServer.Close()

	token, err := jwt.GenerateToken(userID, 5, jwtSecret, 1)
	if err != nil {
		t.Fatalf("generate token failed: %v", err)
	}

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") +
		"/ws?token=" + token + "&send_id=" + userID + "&platform_id=5&sdk_type=go"

	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket failed: %v", err)
	}
	defer clientConn.Close()

	req := WSRequest{
		ReqIdentifier: 9999, // unknown req id should return protocol error
		MsgIncr:       "1",
		OperationId:   "op_direct_ws",
		SendId:        userID,
	}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request failed: %v", err)
	}

	if err = clientConn.WriteMessage(websocket.TextMessage, reqBytes); err != nil {
		t.Fatalf("write websocket message failed: %v", err)
	}

	if err = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set read deadline failed: %v", err)
	}
	_, respBytes, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("read websocket response failed: %v", err)
	}

	var resp WSResponse
	if err = json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}

	if resp.ReqIdentifier != req.ReqIdentifier {
		t.Fatalf("unexpected req_identifier: got %d want %d", resp.ReqIdentifier, req.ReqIdentifier)
	}
	if resp.ErrCode == 0 {
		t.Fatalf("expected error response, got success: %+v", resp)
	}
	if !strings.Contains(resp.ErrMsg, ErrInvalidProtocol.Error()) {
		t.Fatalf("unexpected err_msg: got %q want contains %q", resp.ErrMsg, ErrInvalidProtocol.Error())
	}
}
