// Package imou_life implements the server-side Imou Life OAuth/OpenAPI client.
// Project secrets and user tokens must only be used by the Domus backend.
package imou_life

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	apiPathPrefix = "/openapi/"
	maxResponse   = 1 << 20
)

var dataCenterPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,30}[a-z0-9]$`)

type Config struct {
	AccessKey  string
	SecretKey  string
	DataCenter string
}

type Client struct {
	config Config
	baseURL string
	http    *http.Client
}

type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type Device struct {
	DeviceID      string    `json:"deviceId"`
	ProductID     string    `json:"productId,omitempty"`
	Status        string    `json:"deviceStatus,omitempty"`
	Model         string    `json:"deviceModel,omitempty"`
	Category      string    `json:"catalog,omitempty"`
	Brand         string    `json:"brand,omitempty"`
	Version       string    `json:"deviceVersion,omitempty"`
	Name          string    `json:"deviceName,omitempty"`
	Source        string    `json:"source,omitempty"`
	ChannelNum    int       `json:"channelNum,omitempty"`
	Channels      []Channel `json:"channelList,omitempty"`
	// The provider also returns play tokens and encrypted device credentials.
	// They are intentionally not represented or returned by this client.
}

type Channel struct {
	ChannelID     string   `json:"channelId"`
	Name          string   `json:"channelName,omitempty"`
	Status        string   `json:"channelStatus,omitempty"`
	CameraStatus  string   `json:"cameraStatus,omitempty"`
	Capabilities  []string `json:"capabilityIds,omitempty"`
}

type apiEnvelope struct {
	Result struct {
		Code string          `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	} `json:"result"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	Error        string `json:"error"`
}

// NewClient constructs a client for the selected Imou data centre.
// The data-centre hostname is built locally; caller-supplied API URLs are not accepted.
func NewClient(cfg Config, client *http.Client) (*Client, error) {
	cfg.AccessKey = strings.TrimSpace(cfg.AccessKey)
	cfg.SecretKey = strings.TrimSpace(cfg.SecretKey)
	cfg.DataCenter = strings.ToLower(strings.TrimSpace(cfg.DataCenter))
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, errors.New("imou: project access key and secret are required")
	}
	if !dataCenterPattern.MatchString(cfg.DataCenter) {
		return nil, errors.New("imou: invalid data centre identifier")
	}
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second}
	}
	safeClient := *client
	safeClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	if safeClient.Timeout <= 0 || safeClient.Timeout > 20*time.Second {
		safeClient.Timeout = 12 * time.Second
	}
	return &Client{
		config: cfg,
		baseURL: "https://openapi-" + cfg.DataCenter + ".easy4ip.com",
		http: &safeClient,
	}, nil
}

// ExchangeCode exchanges the Imou H5 callback code for an end-user OAuth token.
func (c *Client) ExchangeCode(ctx context.Context, code, redirectURI string) (*OAuthToken, error) {
	code = strings.TrimSpace(code)
	if code == "" || len(code) > 2048 {
		return nil, errors.New("imou: invalid authorization code")
	}
	values := url.Values{"grant_type": {"authorization_code"}, "code": {code}}
	if redirectURI != "" {
		values.Set("redirect_uri", redirectURI)
	}
	return c.oauthToken(ctx, values)
}

// RefreshToken renews an end-user token. It never accepts administrator or sub-account tokens.
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*OAuthToken, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" || len(refreshToken) > 4096 {
		return nil, errors.New("imou: invalid refresh token")
	}
	return c.oauthToken(ctx, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refreshToken}})
}

func (c *Client) oauthToken(ctx context.Context, values url.Values) (*OAuthToken, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+apiPathPrefix+"oauthTokenV2", strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("imou: create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(c.config.AccessKey, c.config.SecretKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("imou: token endpoint request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse))
	if err != nil {
		return nil, fmt.Errorf("imou: read token response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("imou: token endpoint returned HTTP %d", resp.StatusCode)
	}
	var raw tokenResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("imou: invalid token response")
	}
	if raw.Error != "" {
		return nil, fmt.Errorf("imou: OAuth provider error %q", raw.Error)
	}
	if !strings.HasPrefix(raw.AccessToken, "Ot_") || raw.RefreshToken == "" || raw.ExpiresIn < 60 || raw.ExpiresIn > 86400 {
		return nil, errors.New("imou: token response is missing required or valid fields")
	}
	if raw.TokenType != "" && !strings.EqualFold(raw.TokenType, "bearer") {
		return nil, errors.New("imou: unsupported OAuth token type")
	}
	return &OAuthToken{
		AccessToken: raw.AccessToken,
		RefreshToken: raw.RefreshToken,
		TokenType: "bearer",
		ExpiresAt: time.Now().UTC().Add(time.Duration(raw.ExpiresIn) * time.Second),
	}, nil
}

// ListDevices returns one page of devices visible to the delegated OAuth user.
// page is 1-based and pageSize is restricted to the provider's documented 1..50 range.
func (c *Client) ListDevices(ctx context.Context, accessToken string, page, pageSize int) ([]Device, int, error) {
	if !strings.HasPrefix(accessToken, "Ot_") {
		return nil, 0, errors.New("imou: an end-user OAuth token is required")
	}
	if page < 1 || page > 1000 || pageSize < 1 || pageSize > 50 {
		return nil, 0, errors.New("imou: device pagination is out of range")
	}
	params := map[string]any{
		"token": accessToken,
		"page": page,
		"pageSize": pageSize,
		"source": "bindAndShare",
	}
	data, err := c.call(ctx, "v2/device.detail.page", params)
	if err != nil {
		return nil, 0, err
	}
	var result struct {
		Count int      `json:"count"`
		Devices []Device `json:"deviceList"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, 0, errors.New("imou: invalid device-list response")
	}
	if result.Devices == nil {
		result.Devices = []Device{}
	}
	return result.Devices, result.Count, nil
}

// GetStreamURL obtains an RTSP URL for a device/channel. Treat the returned URL as a secret.
func (c *Client) GetStreamURL(ctx context.Context, accessToken, deviceID string, channelID, streamID int) (string, error) {
	if !strings.HasPrefix(accessToken, "Ot_") {
		return "", errors.New("imou: an end-user OAuth token is required")
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" || len(deviceID) > 128 || strings.ContainsAny(deviceID, "/\\?#\r\n\t ") {
		return "", errors.New("imou: invalid device identifier")
	}
	if channelID < 0 || channelID > 255 || (streamID != 0 && streamID != 1) {
		return "", errors.New("imou: invalid channel or stream identifier")
	}
	data, err := c.call(ctx, "getStreamUrl", map[string]any{
		"token": accessToken,
		"deviceId": deviceID,
		"channelId": channelID,
		"streamId": streamID,
	})
	if err != nil {
		return "", err
	}
	var result struct { URL string `json:"url"` }
	if err := json.Unmarshal(data, &result); err != nil {
		return "", errors.New("imou: invalid stream URL response")
	}
	u, err := url.Parse(result.URL)
	if err != nil || u.Scheme != "rtsp" || u.Host == "" {
		return "", errors.New("imou: provider returned an invalid RTSP URL")
	}
	return result.URL, nil
}

func (c *Client) call(ctx context.Context, method string, params map[string]any) (json.RawMessage, error) {
	if method == "" || strings.ContainsAny(method, "/\\?#") {
		return nil, errors.New("imou: invalid API method")
	}
	now := time.Now().UTC().Unix()
	nonce, err := randomUUID()
	if err != nil {
		return nil, fmt.Errorf("imou: create request nonce: %w", err)
	}
	signature := c.sign(now, nonce)
	payload := map[string]any{
		"id": nonce,
		"system": map[string]any{"ver": "1.0", "appId": c.config.AccessKey, "sign": signature, "time": now, "nonce": nonce},
		"params": params,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("imou: encode API request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+apiPathPrefix+method, bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("imou: create API request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("imou: API request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse))
	if err != nil {
		return nil, fmt.Errorf("imou: read API response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("imou: API returned HTTP %d", resp.StatusCode)
	}
	var envelope apiEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, errors.New("imou: invalid API response")
	}
	if envelope.Result.Code != "0" {
		code := strings.TrimSpace(envelope.Result.Code)
		if len(code) > 32 { code = code[:32] }
		return nil, fmt.Errorf("imou: provider rejected %s (code %s)", method, code)
	}
	if len(envelope.Result.Data) == 0 || string(envelope.Result.Data) == "null" {
		return json.RawMessage("{}"), nil
	}
	return envelope.Result.Data, nil
}

func (c *Client) sign(timestamp int64, nonce string) string {
	source := "time:" + strconv.FormatInt(timestamp, 10) + ",nonce:" + nonce + ",appSecret:" + c.config.SecretKey
	digest := sha256.Sum256([]byte(c.config.SecretKey))
	key := []byte(hex.EncodeToString(digest[:]))
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(source))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func randomUUID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil { return "", err }
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(value[:])
	return encoded[:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:], nil
}
