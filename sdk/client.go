package sdk

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// Client is the SDK client for Nexo IM API
type Client struct {
	baseURL    string
	httpClient *client.Client
	token      string
	ignoreAuth bool
	internal   *internalAuthConfig
}

type internalAuthConfig struct {
	serviceName string
	secret      string
}

type actAsUserConfig struct {
	userId     string
	platformId int
}

type requestOptions struct {
	actAsUser *actAsUserConfig
}

// RequestOption configures per-request behavior.
type RequestOption func(*requestOptions)

// ClientOption is a function to configure the client
type ClientOption func(*Client)

// WithHertzClient sets a custom Hertz client
func WithHertzClient(httpClient *client.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithToken sets the authentication token
func WithToken(token string) ClientOption {
	return func(c *Client) {
		c.token = token
	}
}

// WithIgnoreAuthHeader enables Ignore-Auth header for TEST env bypass.
func WithIgnoreAuthHeader(enabled bool) ClientOption {
	return func(c *Client) {
		c.ignoreAuth = enabled
	}
}

// WithInternalAuth enables service-to-service signature auth.
func WithInternalAuth(serviceName, secret string) ClientOption {
	return func(c *Client) {
		serviceName = strings.TrimSpace(serviceName)
		secret = strings.TrimSpace(secret)
		if serviceName == "" || secret == "" {
			c.internal = nil
			return
		}
		c.internal = &internalAuthConfig{
			serviceName: serviceName,
			secret:      secret,
		}
	}
}

// WithActAsUser sets user context headers for a single internal request.
func WithActAsUser(userId string, platformId int) RequestOption {
	return func(o *requestOptions) {
		userId = strings.TrimSpace(userId)
		if userId == "" {
			o.actAsUser = nil
			return
		}
		if platformId <= 0 {
			platformId = PlatformIdWeb
		}
		o.actAsUser = &actAsUserConfig{
			userId:     userId,
			platformId: platformId,
		}
	}
}

// NewClient creates a new SDK client
func NewClient(baseURL string, opts ...ClientOption) (*Client, error) {
	if baseURL == "" {
		baseURL = BaseUrl
	}

	httpClient, err := client.NewClient(
		client.WithDialTimeout(10*time.Second),
		client.WithClientReadTimeout(30*time.Second),
		client.WithWriteTimeout(30*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http client: %w", err)
	}

	c := &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// MustNewClient creates a new SDK client and panics on error
func MustNewClient(baseURL string, opts ...ClientOption) *Client {
	if baseURL == "" {
		baseURL = BaseUrl
	}
	c, err := NewClient(baseURL, opts...)
	if err != nil {
		panic(err)
	}
	return c
}

// NewInternalClient creates a service-to-service auth client.
func NewInternalClient(baseURL, serviceName, secret string, opts ...ClientOption) (*Client, error) {
	opts = append(opts, WithInternalAuth(serviceName, secret))
	return NewClient(baseURL, opts...)
}

// MustNewInternalClient creates a service-to-service auth client and panics on error.
func MustNewInternalClient(baseURL, serviceName, secret string, opts ...ClientOption) *Client {
	opts = append(opts, WithInternalAuth(serviceName, secret))
	return MustNewClient(baseURL, opts...)
}

// SetToken sets the authentication token
func (c *Client) SetToken(token string) {
	c.token = token
}

// GetToken returns the current token
func (c *Client) GetToken() string {
	return c.token
}

// SetIgnoreAuth controls whether Ignore-Auth header is sent.
func (c *Client) SetIgnoreAuth(enabled bool) {
	c.ignoreAuth = enabled
}

func buildRequestOptions(opts ...RequestOption) *requestOptions {
	ro := &requestOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(ro)
		}
	}
	return ro
}

func (c *Client) applyAuthHeaders(req *protocol.Request, method, path string, body []byte, reqOpts *requestOptions) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("X-Token", c.token)
	}
	if c.ignoreAuth {
		req.Header.Set("Ignore-Auth", "1")
	}
	if c.internal != nil {
		ts := fmt.Sprintf("%d", time.Now().Unix())
		signature := signInternalRequest(c.internal.secret, c.internal.serviceName, ts, method, path, body)
		req.Header.Set("X-Service-Name", c.internal.serviceName)
		req.Header.Set("X-Timestamp", ts)
		req.Header.Set("X-Signature", signature)
	}
	if reqOpts != nil && reqOpts.actAsUser != nil {
		req.Header.Set("X-User-Id", reqOpts.actAsUser.userId)
		req.Header.Set("X-Platform-Id", strconv.Itoa(reqOpts.actAsUser.platformId))
	}
}

// request makes an HTTP request and decodes the response
func (c *Client) request(ctx context.Context, method, path string, body any, result any, opts ...RequestOption) error {
	reqURL := c.baseURL + path

	req := &protocol.Request{}
	resp := &protocol.Response{}

	req.SetMethod(method)
	req.SetRequestURI(reqURL)
	req.Header.Set("Content-Type", "application/json")

	var jsonBody []byte
	if body != nil {
		var err error
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		req.SetBody(jsonBody)
	}
	c.applyAuthHeaders(req, method, path, jsonBody, buildRequestOptions(opts...))

	err := c.httpClient.Do(ctx, req, resp)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	// Decode response
	var apiResp Response
	if err := json.Unmarshal(resp.Body(), &apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for API error
	if apiResp.Code != 0 {
		return &Error{Code: apiResp.Code, Msg: apiResp.Msg}
	}

	// Decode data if result is provided
	if result != nil && apiResp.Data != nil {
		dataBytes, err := json.Marshal(apiResp.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal response data: %w", err)
		}
		if err := json.Unmarshal(dataBytes, result); err != nil {
			return fmt.Errorf("failed to decode response data: %w", err)
		}
	}

	return nil
}

// get makes a GET request with query parameters
func (c *Client) get(ctx context.Context, path string, params map[string]string, result any, opts ...RequestOption) error {
	reqURL := c.baseURL + path
	if len(params) > 0 {
		query := url.Values{}
		for k, v := range params {
			query.Set(k, v)
		}
		reqURL += "?" + query.Encode()
	}

	req := &protocol.Request{}
	resp := &protocol.Response{}

	req.SetMethod(consts.MethodGet)
	req.SetRequestURI(reqURL)
	c.applyAuthHeaders(req, consts.MethodGet, path, nil, buildRequestOptions(opts...))

	err := c.httpClient.Do(ctx, req, resp)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	// Decode response
	var apiResp Response
	if err = json.Unmarshal(resp.Body(), &apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for API error
	if apiResp.Code != 0 {
		return &Error{Code: apiResp.Code, Msg: apiResp.Msg}
	}

	// Decode data if result is provided
	if result != nil && apiResp.Data != nil {
		dataBytes, err := json.Marshal(apiResp.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal response data: %w", err)
		}
		if err := json.Unmarshal(dataBytes, result); err != nil {
			return fmt.Errorf("failed to decode response data: %w", err)
		}
	}

	return nil
}

// post makes a POST request
func (c *Client) post(ctx context.Context, path string, body interface{}, result interface{}, opts ...RequestOption) error {
	return c.request(ctx, consts.MethodPost, path, body, result, opts...)
}

// put makes a PUT request
func (c *Client) put(ctx context.Context, path string, body interface{}, result interface{}, opts ...RequestOption) error {
	return c.request(ctx, consts.MethodPut, path, body, result, opts...)
}

func signInternalRequest(secret, serviceName, timestamp, method, path string, body []byte) string {
	bodyHashBytes := sha256.Sum256(body)
	bodyHash := hex.EncodeToString(bodyHashBytes[:])
	payload := strings.Join([]string{
		serviceName,
		timestamp,
		strings.ToUpper(method),
		path,
		bodyHash,
	}, "\n")

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
