package gateway

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ZaiSpace/nexo_im/common"
	"github.com/ZaiSpace/nexo_im/internal/middleware"
	"github.com/bytedance/sonic"
	hzclient "github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

const (
	// TODO: move to config after integration is stable.
	appGatewayPushBaseURL = "http://localhost:8000"
	appGatewayPushPath    = "/api/push/send_push"
	appPushBizTypeIM      = "im"
)

type AppPushRequest struct {
	UserId string
	Title  string
	Body   string
	Data   map[string]any
}

type AppPushSender interface {
	SendPush(ctx context.Context, req *AppPushRequest) error
}

type appGatewayPushSender struct {
	baseURL string
	path    string
	client  *hzclient.Client
}

type appGatewaySendPushBody struct {
	UserId      int64                  `json:"user_id"`
	BizType     string                 `json:"biz_type"`
	Title       string                 `json:"title"`
	Body        string                 `json:"body"`
	DataJSON    string                 `json:"data_json,omitempty"`
	CommonParam *appGatewayCommonParam `json:"common_param,omitempty"`
}

type appGatewayCommonParam struct {
	UserId int64 `json:"user_id"`
}

type appGatewayResp struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

func NewDefaultAppPushSender() AppPushSender {
	c, err := hzclient.NewClient(
		hzclient.WithDialTimeout(3*time.Second),
		hzclient.WithClientReadTimeout(3*time.Second),
		hzclient.WithWriteTimeout(3*time.Second),
	)
	if err != nil {
		c = nil
	}

	return &appGatewayPushSender{
		baseURL: appGatewayPushBaseURL,
		path:    appGatewayPushPath,
		client:  c,
	}
}

func (s *appGatewayPushSender) SendPush(ctx context.Context, req *AppPushRequest) error {
	if req == nil {
		return fmt.Errorf("app push request is nil")
	}
	if s.client == nil {
		return fmt.Errorf("hertz client is nil")
	}

	userId, err := parseUserId(req.UserId)
	if err != nil {
		return err
	}

	dataJSON := ""
	if len(req.Data) > 0 {
		raw, marshalErr := sonic.Marshal(req.Data)
		if marshalErr != nil {
			return fmt.Errorf("marshal data_json failed: %w", marshalErr)
		}
		dataJSON = string(raw)
	}

	body := &appGatewaySendPushBody{
		UserId:   userId,
		BizType:  appPushBizTypeIM,
		Title:    req.Title,
		Body:     req.Body,
		DataJSON: dataJSON,
		CommonParam: &appGatewayCommonParam{
			UserId: userId,
		},
	}
	payload, err := sonic.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request failed: %w", err)
	}

	reqURL := strings.TrimRight(s.baseURL, "/") + s.path
	hzReq := &protocol.Request{}
	hzResp := &protocol.Response{}
	hzReq.SetMethod(consts.MethodPost)
	hzReq.SetRequestURI(reqURL)
	hzReq.Header.Set("Content-Type", "application/json")
	hzReq.SetBody(payload)

	if traceID := middleware.GetTraceID(ctx); traceID != "" {
		hzReq.Header.Set(middleware.TraceIDHeader, traceID)
		hzReq.Header.Set(middleware.XTraceIDHeader, traceID)
	}

	err = s.client.Do(ctx, hzReq, hzResp)
	if err != nil {
		return fmt.Errorf("send push request failed: %w", err)
	}
	respBody := hzResp.Body()

	statusCode := hzResp.StatusCode()
	if statusCode < 200 || statusCode >= 300 {
		return fmt.Errorf("push request status=%d body=%s", statusCode, string(respBody))
	}

	var resp appGatewayResp
	if err = sonic.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("decode push response failed: %w", err)
	}
	if resp.Code != 0 {
		return fmt.Errorf("push response code=%d msg=%s", resp.Code, resp.Message)
	}

	return nil
}

func parseUserId(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("invalid user_id: %q", raw)
	}

	if userId, err := strconv.ParseInt(raw, 10, 64); err == nil && userId > 0 {
		return userId, nil
	}

	var actor common.Actor
	if err := actor.FromIMUserId(raw); err == nil && actor.Id > 0 && actor.Role == common.RoleUser {
		return actor.Id, nil
	}
	return 0, fmt.Errorf("invalid user_id: %q", raw)
}
