package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	hzclient "github.com/cloudwego/hertz/pkg/app/client"
)

type capturedAppPushRequest struct {
	Header http.Header
	Body   appGatewaySendPushBody
}

func TestAppGatewayPushSender_SendPush_ActorUserIdFallbackAndNoSignHeaders(t *testing.T) {
	var captured capturedAppPushRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		captured.Header = r.Header.Clone()

		if err := json.NewDecoder(r.Body).Decode(&captured.Body); err != nil {
			t.Fatalf("decode request body failed: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"ok"}`))
	}))
	defer srv.Close()

	s := &appGatewayPushSender{
		baseURL: srv.URL,
		path:    "/api/push/send_push",
		client: func() *hzclient.Client {
			c, err := hzclient.NewClient()
			if err != nil {
				t.Fatalf("new hertz client failed: %v", err)
			}
			return c
		}(),
	}

	err := s.SendPush(context.Background(), &AppPushRequest{
		UserId: "u___42",
		Title:  "title",
		Body:   "body",
		Data: map[string]any{
			"k": "v",
		},
	})
	if err != nil {
		t.Fatalf("SendPush() error = %v", err)
	}

	if got := captured.Header.Get("x-sign"); got != "" {
		t.Fatalf("x-sign should be empty, got %q", got)
	}
	if got := captured.Header.Get("x-timestamp"); got != "" {
		t.Fatalf("x-timestamp should be empty, got %q", got)
	}

	if captured.Body.UserId != 42 {
		t.Fatalf("expected user_id=42, got %d", captured.Body.UserId)
	}
	if captured.Body.CommonParam == nil || captured.Body.CommonParam.UserId != 42 {
		t.Fatalf("expected common_param.user_id=42, got %+v", captured.Body.CommonParam)
	}
	if captured.Body.BizType != appPushBizTypeIM {
		t.Fatalf("expected biz_type=%q, got %q", appPushBizTypeIM, captured.Body.BizType)
	}
}
