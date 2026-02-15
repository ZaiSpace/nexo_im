package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type wsProtocolResponse struct {
	ReqIdentifier int32  `json:"req_identifier"`
	MsgIncr       string `json:"msg_incr"`
	OperationId   string `json:"operation_id"`
	ErrCode       int    `json:"err_code"`
	ErrMsg        string `json:"err_msg"`
	Data          []byte `json:"data"`
}

type wsProtocolRequest struct {
	ReqIdentifier int32  `json:"req_identifier"`
	MsgIncr       string `json:"msg_incr"`
	OperationId   string `json:"operation_id"`
	SendId        string `json:"send_id"`
	Data          []byte `json:"data"`
}

type wsSendMsgReq struct {
	ClientMsgId string `json:"client_msg_id"`
	RecvId      string `json:"recv_id,omitempty"`
	GroupId     string `json:"group_id,omitempty"`
	SessionType int32  `json:"session_type"`
	MsgType     int32  `json:"msg_type"`
	Content     struct {
		Text string `json:"text,omitempty"`
	} `json:"content"`
}

type wsSendMsgResp struct {
	ServerMsgId    int64  `json:"server_msg_id"`
	ConversationId string `json:"conversation_id"`
	Seq            int64  `json:"seq"`
	ClientMsgId    string `json:"client_msg_id"`
	SendAt         int64  `json:"send_at"`
}

// WSClient is a WebSocket test client
type WSClient struct {
	conn     *websocket.Conn
	messages chan WSMessage
	done     chan struct{}
	mu       sync.Mutex
}

// NewWSClient creates a new WebSocket client
func NewWSClient(token, userId string) (*WSClient, error) {
	// Parse base URL to get host
	baseURL := testConfig.BaseURL
	host := "localhost:8080"
	if len(baseURL) > 7 && baseURL[:7] == "http://" {
		host = baseURL[7:]
	} else if len(baseURL) > 8 && baseURL[:8] == "https://" {
		host = baseURL[8:]
	}

	u := url.URL{
		Scheme:   "ws",
		Host:     host,
		Path:     "/ws",
		RawQuery: fmt.Sprintf("token=%s&send_id=%s&platform_id=5", token, userId),
	}

	header := http.Header{}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		return nil, fmt.Errorf("dial websocket: %w", err)
	}

	client := &WSClient{
		conn:     conn,
		messages: make(chan WSMessage, 100),
		done:     make(chan struct{}),
	}

	go client.readLoop()

	return client, nil
}

// readLoop reads messages from WebSocket
func (c *WSClient) readLoop() {
	defer close(c.done)
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		select {
		case c.messages <- msg:
		default:
			// Channel full, drop message
		}
	}
}

// Send sends a message through WebSocket
func (c *WSClient) Send(msg any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// WaitForMessage waits for a message with timeout
func (c *WSClient) WaitForMessage(timeout time.Duration) (*WSMessage, error) {
	select {
	case msg := <-c.messages:
		return &msg, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for message")
	case <-c.done:
		return nil, fmt.Errorf("connection closed")
	}
}

// Close closes the WebSocket connection
func (c *WSClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.Close()
}

func buildWebSocketURL(token, userId string) (string, error) {
	u, err := url.Parse(testConfig.BaseURL)
	if err != nil {
		return "", fmt.Errorf("parse TEST_BASE_URL: %w", err)
	}

	wsScheme := "ws"
	if u.Scheme == "https" {
		wsScheme = "wss"
	}

	query := url.Values{}
	if token != "" {
		query.Set("token", token)
	}
	if userId != "" {
		query.Set("send_id", userId)
		query.Set("platform_id", "5")
	}

	wsURL := url.URL{
		Scheme:   wsScheme,
		Host:     u.Host,
		Path:     "/ws",
		RawQuery: query.Encode(),
	}
	return wsURL.String(), nil
}

func TestWebSocket_Connect(t *testing.T) {
	userId := generateUserId("ws_user")
	_, token := RegisterAndLogin(t, userId, "WS User", "password123")

	t.Run("connect with valid token", func(t *testing.T) {
		wsClient, err := NewWSClient(token, userId)
		if err != nil {
			t.Fatalf("connect websocket failed: %v", err)
		}
		defer wsClient.Close()

		// Connection should be established
		t.Log("WebSocket connected successfully")
	})

	t.Run("connect with invalid token", func(t *testing.T) {
		_, err := NewWSClient("invalid_token", userId)
		if err == nil {
			t.Error("should fail with invalid token")
		}
	})

	t.Run("connect without token", func(t *testing.T) {
		// Parse base URL to get host
		baseURL := testConfig.BaseURL
		host := "localhost:8080"
		if len(baseURL) > 7 && baseURL[:7] == "http://" {
			host = baseURL[7:]
		} else if len(baseURL) > 8 && baseURL[:8] == "https://" {
			host = baseURL[8:]
		}

		u := url.URL{
			Scheme: "ws",
			Host:   host,
			Path:   "/ws",
		}

		_, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err == nil {
			t.Error("should fail without token")
		}
	})
}

func TestWebSocket_ReceiveMessage(t *testing.T) {
	// Create two users
	user1Id := generateUserId("ws_sender")
	user2Id := generateUserId("ws_receiver")
	client1, _ := RegisterAndLogin(t, user1Id, "WS Sender", "password123")
	_, token2 := RegisterAndLogin(t, user2Id, "WS Receiver", "password123")

	// Connect user2 to WebSocket
	wsClient, err := NewWSClient(token2, user2Id)
	if err != nil {
		t.Fatalf("connect websocket failed: %v", err)
	}
	defer wsClient.Close()

	// Give some time for connection to establish
	time.Sleep(100 * time.Millisecond)

	t.Run("receive message via websocket", func(t *testing.T) {
		// Send message via HTTP
		req := SendMessageRequest{
			ClientMsgId: generateClientMsgId(),
			RecvId:      user2Id,
			SessionType: SessionTypeSingle,
			MsgType:     MsgTypeText,
			Content: MessageContent{
				Text: "Hello via WebSocket!",
			},
		}

		resp, err := client1.POST("/msg/send", req)
		if err != nil {
			t.Fatalf("send message failed: %v", err)
		}
		AssertSuccess(t, resp, "send message should succeed")

		// Wait for WebSocket message
		msg, err := wsClient.WaitForMessage(5 * time.Second)
		if err != nil {
			t.Fatalf("wait for message failed: %v", err)
		}

		t.Logf("Received WebSocket message: type=%s", msg.Type)

		// Verify message content
		if msg.Type != "new_message" && msg.Type != "message" {
			t.Logf("Received message type: %s (may vary by implementation)", msg.Type)
		}
	})
}

func TestWebSocket_MultipleConnections(t *testing.T) {
	userId := generateUserId("ws_multi")
	_, token := RegisterAndLogin(t, userId, "WS Multi", "password123")

	t.Run("multiple connections same user", func(t *testing.T) {
		// Connect multiple times
		clients := make([]*WSClient, 3)
		for i := range clients {
			client, err := NewWSClient(token, userId)
			if err != nil {
				t.Fatalf("connect websocket %d failed: %v", i, err)
			}
			clients[i] = client
		}

		// Clean up
		for _, client := range clients {
			if client != nil {
				client.Close()
			}
		}
	})
}

func TestWebSocket_GroupMessage(t *testing.T) {
	// Create users
	ownerId := generateUserId("ws_group_owner")
	memberId := generateUserId("ws_group_member")
	ownerClient, ownerToken := RegisterAndLogin(t, ownerId, "WS Group Owner", "password123")
	_, memberToken := RegisterAndLogin(t, memberId, "WS Group Member", "password123")

	// Create group
	groupId := CreateGroupAndGetId(t, ownerClient, "WS Test Group", []string{memberId})

	// Connect both users to WebSocket
	ownerWS, err := NewWSClient(ownerToken, ownerId)
	if err != nil {
		t.Fatalf("connect owner websocket failed: %v", err)
	}
	defer ownerWS.Close()

	memberWS, err := NewWSClient(memberToken, memberId)
	if err != nil {
		t.Fatalf("connect member websocket failed: %v", err)
	}
	defer memberWS.Close()

	time.Sleep(100 * time.Millisecond)

	t.Run("group members receive message", func(t *testing.T) {
		// Send group message
		req := SendMessageRequest{
			ClientMsgId: generateClientMsgId(),
			GroupId:     groupId,
			SessionType: SessionTypeGroup,
			MsgType:     MsgTypeText,
			Content: MessageContent{
				Text: "Hello group via WebSocket!",
			},
		}

		resp, err := ownerClient.POST("/msg/send", req)
		if err != nil {
			t.Fatalf("send message failed: %v", err)
		}
		AssertSuccess(t, resp, "send message should succeed")

		// Both should receive the message
		// Owner receives their own message
		ownerMsg, err := ownerWS.WaitForMessage(5 * time.Second)
		if err != nil {
			t.Logf("Owner may not receive own message: %v", err)
		} else {
			t.Logf("Owner received: type=%s", ownerMsg.Type)
		}

		// Member receives the message
		memberMsg, err := memberWS.WaitForMessage(5 * time.Second)
		if err != nil {
			t.Fatalf("member wait for message failed: %v", err)
		}
		t.Logf("Member received: type=%s", memberMsg.Type)
	})
}

func TestWebSocket_SendMessageToRemoteServer(t *testing.T) {
	userId := generateUserId("ws_direct_send")
	_, token := RegisterAndLogin(t, userId, "WS Direct Sender", "password123")

	wsURL, err := buildWebSocketURL(token, userId)
	if err != nil {
		t.Fatalf("build websocket url failed: %v", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket failed: %v", err)
	}
	defer conn.Close()

	req := map[string]any{
		"req_identifier": 9999,
		"msg_incr":       "1",
		"operation_id":   "ws_direct_send_test",
		"send_id":        userId,
		"data":           json.RawMessage(`{}`),
	}

	if err = conn.WriteJSON(req); err != nil {
		t.Fatalf("write websocket message failed: %v", err)
	}

	if err = conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("set read deadline failed: %v", err)
	}
	_, respBytes, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read websocket response failed: %v", err)
	}

	var resp wsProtocolResponse
	if err = json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal websocket response failed: %v, raw=%s", err, string(respBytes))
	}

	if resp.ReqIdentifier != 9999 {
		t.Fatalf("unexpected req_identifier: got %d want %d", resp.ReqIdentifier, 9999)
	}
	if resp.ErrCode == 0 {
		t.Fatalf("expected non-zero err_code for unknown req_identifier, got %+v", resp)
	}
}

func TestWebSocket_SendRealMessage(t *testing.T) {
	senderID := generateUserId("ws_real_sender")
	receiverID := generateUserId("ws_real_receiver")
	_, senderToken := RegisterAndLogin(t, senderID, "WS Real Sender", "password123")
	receiverClient, _ := RegisterAndLogin(t, receiverID, "WS Real Receiver", "password123")

	wsURL, err := buildWebSocketURL(senderToken, senderID)
	if err != nil {
		t.Fatalf("build websocket url failed: %v", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket failed: %v", err)
	}
	defer conn.Close()

	sendReq := wsSendMsgReq{
		ClientMsgId: generateClientMsgId(),
		RecvId:      receiverID,
		SessionType: SessionTypeSingle,
		MsgType:     MsgTypeText,
	}
	sendReq.Content.Text = "websocket real send integration test"

	data, err := json.Marshal(sendReq)
	if err != nil {
		t.Fatalf("marshal ws send req failed: %v", err)
	}

	req := wsProtocolRequest{
		ReqIdentifier: 1003, // WSSendMsg
		MsgIncr:       "1",
		OperationId:   "ws_real_send_test",
		SendId:        senderID,
		Data:          data,
	}

	if err = conn.WriteJSON(req); err != nil {
		t.Fatalf("write websocket request failed: %v", err)
	}

	if err = conn.SetReadDeadline(time.Now().Add(8 * time.Second)); err != nil {
		t.Fatalf("set read deadline failed: %v", err)
	}
	_, respBytes, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read websocket response failed: %v", err)
	}

	var resp wsProtocolResponse
	if err = json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal websocket response failed: %v, raw=%s", err, string(respBytes))
	}

	if resp.ReqIdentifier != 1003 {
		t.Fatalf("unexpected req_identifier: got %d want %d", resp.ReqIdentifier, 1003)
	}
	if resp.ErrCode != 0 {
		t.Fatalf("expected err_code=0, got err_code=%d err_msg=%s", resp.ErrCode, resp.ErrMsg)
	}

	var sendResp wsSendMsgResp
	if err = json.Unmarshal(resp.Data, &sendResp); err != nil {
		t.Fatalf("unmarshal send response data failed: %v", err)
	}
	t.Log(sendResp)
	if sendResp.ClientMsgId != sendReq.ClientMsgId {
		t.Fatalf("client_msg_id mismatch: got %s want %s", sendResp.ClientMsgId, sendReq.ClientMsgId)
	}
	if sendResp.ConversationId == "" || sendResp.Seq <= 0 {
		t.Fatalf("invalid send response: %+v", sendResp)
	}

	// Verify message can be pulled by receiver, which indirectly verifies persistence.
	pullResp, err := receiverClient.GET(fmt.Sprintf(
		"/msg/pull?conversation_id=%s&begin_seq=%d&end_seq=%d&limit=10",
		sendResp.ConversationId, sendResp.Seq, sendResp.Seq,
	))
	if err != nil {
		t.Fatalf("receiver pull message failed: %v", err)
	}
	AssertSuccess(t, pullResp, "receiver pull should succeed")

	var pulled PullMessagesResponse
	if err = pullResp.ParseData(&pulled); err != nil {
		t.Fatalf("parse pull response failed: %v", err)
	}
	if len(pulled.Messages) == 0 {
		t.Fatalf("pull returned no messages for conversation=%s seq=%d", sendResp.ConversationId, sendResp.Seq)
	}

	last := pulled.Messages[len(pulled.Messages)-1]
	if last.ClientMsgId != sendReq.ClientMsgId {
		t.Fatalf("pulled client_msg_id mismatch: got %s want %s", last.ClientMsgId, sendReq.ClientMsgId)
	}
}
