package sdk

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

var integrationClient = MustNewClient("http://localhost:8080")

func ensureLocalServerReachable(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := integrationClient.GetConversationList(ctx)
	if err == nil {
		return
	}

	var apiErr *Error
	if errors.As(err, &apiErr) {
		return
	}

	if strings.Contains(err.Error(), "failed to send request") {
		t.Skipf("skip integration test: %v", err)
	}
}

type testUsers struct {
	password string
	userA    *UserInfo
	userB    *UserInfo
	clientA  *Client
	clientB  *Client
}

func setupTwoUsers(t *testing.T) *testUsers {
	t.Helper()
	ensureLocalServerReachable(t)

	password := "pass123456"
	suffix := time.Now().UnixNano()
	userAID := fmt.Sprintf("it_u_%d_a", suffix)
	userBID := fmt.Sprintf("it_u_%d_b", suffix)

	regClient := MustNewClient("http://localhost:8080")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userA, err := regClient.Register(ctx, &RegisterRequest{
		UserId:   userAID,
		Nickname: "integration-a",
		Password: password,
	})
	if err != nil {
		t.Fatalf("register userA failed: %v", err)
	}

	userB, err := regClient.Register(ctx, &RegisterRequest{
		UserId:   userBID,
		Nickname: "integration-b",
		Password: password,
	})
	if err != nil {
		t.Fatalf("register userB failed: %v", err)
	}

	clientA := MustNewClient("http://localhost:8080")
	loginA, err := clientA.Login(ctx, &LoginRequest{
		UserId:     userAID,
		Password:   password,
		PlatformId: PlatformIdWeb,
	})
	if err != nil {
		t.Fatalf("login userA failed: %v", err)
	}
	if loginA.Token == "" {
		t.Fatal("login userA token is empty")
	}

	clientB := MustNewClient("http://localhost:8080")
	loginB, err := clientB.Login(ctx, &LoginRequest{
		UserId:     userBID,
		Password:   password,
		PlatformId: PlatformIdWeb,
	})
	if err != nil {
		t.Fatalf("login userB failed: %v", err)
	}
	if loginB.Token == "" {
		t.Fatal("login userB token is empty")
	}

	return &testUsers{
		password: password,
		userA:    userA,
		userB:    userB,
		clientA:  clientA,
		clientB:  clientB,
	}
}

func TestAuthAndUserMethods_Integration(t *testing.T) {
	users := setupTwoUsers(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	me, err := users.clientA.GetUserInfo(ctx)
	if err != nil {
		t.Fatalf("GetUserInfo failed: %v", err)
	}
	if me.Id != users.userA.Id {
		t.Fatalf("GetUserInfo id = %s, want %s", me.Id, users.userA.Id)
	}

	newNickname := "integration-a-updated"
	updated, err := users.clientA.UpdateUserInfo(ctx, &UpdateUserRequest{
		Nickname: newNickname,
	})
	if err != nil {
		t.Fatalf("UpdateUserInfo failed: %v", err)
	}
	if updated.Nickname != newNickname {
		t.Fatalf("UpdateUserInfo nickname = %s, want %s", updated.Nickname, newNickname)
	}

	peer, err := users.clientA.GetUserInfoById(ctx, users.userB.Id)
	if err != nil {
		t.Fatalf("GetUserInfoById failed: %v", err)
	}
	if peer.Id != users.userB.Id {
		t.Fatalf("GetUserInfoById id = %s, want %s", peer.Id, users.userB.Id)
	}

	batch, err := users.clientA.GetUsersInfo(ctx, []string{users.userA.Id, users.userB.Id})
	if err != nil {
		t.Fatalf("GetUsersInfo failed: %v", err)
	}
	if len(batch) != 2 {
		t.Fatalf("GetUsersInfo len = %d, want 2", len(batch))
	}

	statuses, err := users.clientA.GetUsersOnlineStatus(ctx, []string{users.userA.Id, users.userB.Id})
	if err != nil {
		t.Fatalf("GetUsersOnlineStatus failed: %v", err)
	}
	if len(statuses) == 0 {
		t.Fatal("GetUsersOnlineStatus returned empty list")
	}
}

func TestMessageAndConversationMethods_Integration(t *testing.T) {
	users := setupTwoUsers(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	clientMsgID := fmt.Sprintf("it-msg-%d", time.Now().UnixNano())
	msg, err := users.clientA.SendTextMessage(ctx, clientMsgID, users.userB.Id, "hello from integration test")
	if err != nil {
		t.Fatalf("SendTextMessage failed: %v", err)
	}
	if msg.ConversationId == "" {
		t.Fatal("SendTextMessage conversation_id is empty")
	}
	if msg.Seq <= 0 {
		t.Fatalf("SendTextMessage seq = %d, want > 0", msg.Seq)
	}

	maxSeq, err := users.clientA.GetMaxSeq(ctx, msg.ConversationId)
	if err != nil {
		t.Fatalf("GetMaxSeq failed: %v", err)
	}
	if maxSeq < msg.Seq {
		t.Fatalf("GetMaxSeq = %d, want >= %d", maxSeq, msg.Seq)
	}

	pulled, err := users.clientA.PullMessages(ctx, msg.ConversationId, 0, 0, 50)
	if err != nil {
		t.Fatalf("PullMessages failed: %v", err)
	}
	if len(pulled.Messages) == 0 {
		t.Fatal("PullMessages returned empty messages")
	}

	unread, err := users.clientB.GetUnreadCount(ctx, msg.ConversationId, 0)
	if err != nil {
		t.Fatalf("GetUnreadCount failed: %v", err)
	}
	if unread < 0 {
		t.Fatalf("GetUnreadCount = %d, want >= 0", unread)
	}

	if err := users.clientB.MarkRead(ctx, msg.ConversationId, msg.Seq); err != nil {
		t.Fatalf("MarkRead failed: %v", err)
	}

	seqInfo, err := users.clientB.GetMaxReadSeq(ctx, msg.ConversationId)
	if err != nil {
		t.Fatalf("GetMaxReadSeq failed: %v", err)
	}
	if seqInfo.ReadSeq < msg.Seq {
		t.Fatalf("GetMaxReadSeq.read_seq = %d, want >= %d", seqInfo.ReadSeq, msg.Seq)
	}

	if err := users.clientA.SetConversationPinned(ctx, msg.ConversationId, true); err != nil {
		t.Fatalf("SetConversationPinned failed: %v", err)
	}
	if err := users.clientA.SetConversationRecvMsgOpt(ctx, msg.ConversationId, RecvMsgOptNoNotify); err != nil {
		t.Fatalf("SetConversationRecvMsgOpt failed: %v", err)
	}

	conv, err := users.clientA.GetConversation(ctx, msg.ConversationId)
	if err != nil {
		t.Fatalf("GetConversation failed: %v", err)
	}
	if conv.ConversationId != msg.ConversationId {
		t.Fatalf("GetConversation conversation_id = %s, want %s", conv.ConversationId, msg.ConversationId)
	}

	conversations, err := users.clientA.GetConversationList(ctx)
	if err != nil {
		t.Fatalf("GetConversationList failed: %v", err)
	}
	if len(conversations) == 0 {
		t.Fatal("GetConversationList returned empty list")
	}
}
