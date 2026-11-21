// Package imou_life implements the server-side Imou Life OAuth/OpenAPI client.
// Project secrets and user tokens must only be used by the Domus backend.
package imou_life

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
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

type AdminToken struct {
	AccessToken string
	ExpiresAt   time.Time
}

type Capability struct {
	CapabilityID string                `json:"capabilityId"`
	Name         string                `json:"name,omitempty"`
	Scope        string                `json:"scope,omitempty"`
	Properties   []CapabilityProperty  `json:"properties,omitempty"`
	Services     []CapabilityService   `json:"services,omitempty"`
	Events       []CapabilityEvent     `json:"events,omitempty"`
}

type CapabilityProperty struct {
	PropertyID string         `json:"propertyId"`
	Name       string         `json:"name,omitempty"`
	AccessMode string         `json:"accessMode,omitempty"`
	DataType   map[string]any `json:"dataType,omitempty"`
	Default    any            `json:"defaultData,omitempty"`
}

type CapabilityService struct {
	ServiceID string         `json:"serviceId"`
	Name      string         `json:"name,omitempty"`
	InputData []map[string]any `json:"inputData,omitempty"`
	OutputData []map[string]any `json:"outputData,omitempty"`
	TimeoutMS int            `json:"timeoutMs,omitempty"`
}

type CapabilityEvent struct {
	EventID    string           `json:"eventId"`
	Name       string           `json:"name,omitempty"`
	OutputData []map[string]any `json:"outputData,omitempty"`
}

type DeviceMessage struct {
	ID                string          `json:"id,omitempty"`
	AppID             string          `json:"appId,omitempty"`
	MessageType       string          `json:"msgType"`
	DeviceID          string          `json:"deviceId,omitempty"`
	LegacyDeviceID    string          `json:"did,omitempty"`
	ProductID         string          `json:"productId,omitempty"`
	ChannelID         string          `json:"channelId,omitempty"`
	LegacyChannelID   string          `json:"cid,omitempty"`
	OutputData        json.RawMessage `json:"outputData,omitempty"`
	OccurredUnix      int64           `json:"time,omitempty"`
	LocalTime         string          `json:"localTime,omitempty"`
	AlarmID           string          `json:"alarmId,omitempty"`
	LabelType         string          `json:"labelType,omitempty"`
	DeviceStatus      string          `json:"deviceStatus,omitempty"`
	// Play tokens, temporary image URLs and recording tokens are intentionally omitted.
}

type MessagePage struct {
	Count       int             `json:"count"`
	NextAlarmID string          `json:"nextAlarmId,omitempty"`
	Messages    []DeviceMessage `json:"messages"`
}

type Recording struct {
	FileName      string `json:"fileName,omitempty"`
	FileLength    int64  `json:"fileLength,omitempty"`
	Type          int    `json:"type,omitempty"`
	StreamType    int    `json:"streamType,omitempty"`
	BeginTime     string `json:"beginTime,omitempty"`
	EndTime       string `json:"endTime,omitempty"`
	LocalBeginTime string `json:"localBeginTime,omitempty"`
	LocalEndTime  string `json:"localEndTime,omitempty"`
	ObjectType    string `json:"objectType,omitempty"`
	XSeq          int64  `json:"xseq,omitempty"`
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
	if !strings.HasPrefix(raw.AccessToken, "Ot_") || len(raw.AccessToken) > 4096 ||
		raw.RefreshToken == "" || len(raw.RefreshToken) > 4096 || raw.ExpiresIn < 60 || raw.ExpiresIn > 86400 {
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
	return validateImouRTSPURL(ctx, result.URL)
}


func (c *Client) CreateVideoDownloadStreamURL(ctx context.Context, adminToken, deviceID, channelID, productID, businessType, streamType string, begin, end time.Time) (string, error) {
	if !strings.HasPrefix(adminToken, "At_") || len(adminToken) > 4096 {
		return "", errors.New("imou: administrator token is required for video download")
	}
	if !validDeviceID(deviceID) || !validRequiredChannel(channelID) {
		return "", errors.New("imou: invalid device or channel for video download")
	}
	if streamType == "" { streamType = "0" }
	if streamType != "0" && streamType != "1" {
		return "", errors.New("imou: stream type must be 0 (main) or 1 (sub)")
	}
	params := map[string]any{"token": adminToken, "deviceId": deviceID, "channelId": channelID, "businessType": businessType, "streamType": streamType}
	if productID != "" {
		if len(productID) > 128 || strings.ContainsAny(productID, "/\\?#\r\n\t ") {
			return "", errors.New("imou: invalid IoT product ID")
		}
		params["productId"] = productID
	}
	switch businessType {
	case "real":
	case "localRecord", "cloudRecord":
		if begin.IsZero() || end.IsZero() || !end.After(begin) || end.Sub(begin) > 24*time.Hour {
			return "", errors.New("imou: recording playback range must be between 0 and 24 hours")
		}
		params["beginTime"] = begin.Format("2006-01-02 15:04:05")
		params["endTime"] = end.Format("2006-01-02 15:04:05")
	default:
		return "", errors.New("imou: unsupported video business type")
	}
	data, err := c.call(ctx, "createVideoDownloadStreamUrl", params)
	if err != nil { return "", err }
	var result struct { URL string `json:"url"` }
	if err := json.Unmarshal(data, &result); err != nil || result.URL == "" {
		return "", errors.New("imou: provider returned an invalid encrypted video URL")
	}
	plainURL, err := decryptVideoDownloadURL(result.URL, adminToken)
	if err != nil { return "", err }
	return validateImouRTSPURL(ctx, plainURL)
}

func decryptVideoDownloadURL(encoded, token string) (string, error) {
	payload, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(payload) < 12+16 {
		return "", errors.New("imou: invalid encrypted video URL payload")
	}
	key := sha256.Sum256([]byte(token + "ENCRYPTKEY"))
	block, err := aes.NewCipher(key[:])
	if err != nil { return "", errors.New("imou: could not initialize video URL decryption") }
	gcm, err := cipher.NewGCM(block)
	if err != nil { return "", errors.New("imou: could not initialize video URL authentication") }
	plain, err := gcm.Open(nil, payload[:gcm.NonceSize()], payload[gcm.NonceSize():], nil)
	if err != nil { return "", errors.New("imou: encrypted video URL authentication failed") }
	return string(plain), nil
}

func validateImouRTSPURL(ctx context.Context, raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Scheme, "rtsp") || u.Host == "" || u.Fragment != "" ||
		strings.ContainsAny(raw, "\\r\\n\\\\ ") || u.Opaque != "" {
		return "", errors.New("imou: provider returned an invalid RTSP URL")
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if net.ParseIP(host) != nil || !(strings.HasSuffix(host, ".lechange.com") || host == "lechange.com" ||
		strings.HasSuffix(host, ".easy4ip.com") || host == "easy4ip.com" ||
		strings.HasSuffix(host, ".imoulife.com") || host == "imoulife.com") {
		return "", errors.New("imou: stream host is not an approved Imou media domain")
	}
	if port := u.Port(); port != "" {
		if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
			return "", errors.New("imou: invalid RTSP port")
		}
	}
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil || len(addresses) == 0 {
		return "", errors.New("imou: stream host could not be resolved safely")
	}
	for _, address := range addresses {
		ip := address.IP
		if ip == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() ||
			ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() ||
			isImouCarrierGradeNAT(ip) {
			return "", errors.New("imou: stream host resolves to a non-public network address")
		}
	}
	return u.String(), nil
}

func isImouCarrierGradeNAT(ip net.IP) bool {
	v4 := ip.To4()
	return v4 != nil && v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127
}

func (c *Client) call(ctx context.Context, method string, params map[string]any) (json.RawMessage, error) {
	if !validAPIMethod(method) {
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

func validAPIMethod(method string) bool {
	if method == "" || len(method) > 128 || strings.HasPrefix(method, "/") ||
		strings.HasSuffix(method, "/") || strings.ContainsAny(method, "\\\\?#") {
		return false
	}
	for _, part := range strings.Split(method, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
		for _, ch := range part {
			if !(ch >= 'a' && ch <= 'z') && !(ch >= 'A' && ch <= 'Z') &&
				!(ch >= '0' && ch <= '9') && ch != '.' && ch != '_' && ch != '-' {
				return false
			}
		}
	}
	return true
}

func (c *Client) GetAdminToken(ctx context.Context) (*AdminToken, error) {
	data, err := c.call(ctx, "accessToken", map[string]any{})
	if err != nil {
		return nil, err
	}
	var result struct {
		AccessToken string `json:"accessToken"`
		ExpireTime  int64  `json:"expireTime"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, errors.New("imou: invalid administrator token response")
	}
	if !strings.HasPrefix(result.AccessToken, "At_") || result.ExpireTime < 60 || result.ExpireTime > 259200 {
		return nil, errors.New("imou: administrator token response is invalid")
	}
	return &AdminToken{AccessToken: result.AccessToken, ExpiresAt: time.Now().UTC().Add(time.Duration(result.ExpireTime) * time.Second)}, nil
}

func (c *Client) ListCapabilities(ctx context.Context, accessToken, deviceID, channelID string) ([]Capability, error) {
	if !validUserToken(accessToken) || !validDeviceID(deviceID) || !validOptionalChannel(channelID) {
		return nil, errors.New("imou: invalid token, device or channel")
	}
	params := map[string]any{"token": accessToken, "deviceId": deviceID}
	if channelID != "" {
		params["channelId"] = channelID
	}
	data, err := c.call(ctx, "v2/device.capability.list", params)
	if err != nil {
		return nil, err
	}
	var result struct { Capabilities []Capability `json:"capabilities"` }
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, errors.New("imou: invalid capability-list response")
	}
	if result.Capabilities == nil {
		result.Capabilities = []Capability{}
	}
	return result.Capabilities, nil
}

func (c *Client) ReadProperties(ctx context.Context, accessToken, deviceID, channelID string, properties []string) (json.RawMessage, error) {
	if !validUserToken(accessToken) || !validDeviceID(deviceID) || !validOptionalChannel(channelID) ||
		len(properties) == 0 || len(properties) > 64 {
		return nil, errors.New("imou: invalid property-read request")
	}
	for _, propertyID := range properties {
		if !validCapabilityID(propertyID) {
			return nil, errors.New("imou: invalid property identifier")
		}
	}
	params := map[string]any{"token": accessToken, "deviceId": deviceID, "properties": properties}
	if channelID != "" { params["channelId"] = channelID }
	return c.call(ctx, "v2/device.property.get", params)
}

func (c *Client) WriteProperties(ctx context.Context, accessToken, deviceID, channelID string, properties map[string]any) (json.RawMessage, error) {
	if !validUserToken(accessToken) || !validDeviceID(deviceID) || !validOptionalChannel(channelID) ||
		len(properties) == 0 || len(properties) > 32 {
		return nil, errors.New("imou: invalid property-write request")
	}
	for propertyID := range properties {
		if !validCapabilityID(propertyID) {
			return nil, errors.New("imou: invalid property identifier")
		}
	}
	encoded, err := json.Marshal(properties)
	if err != nil || len(encoded) > 16*1024 {
		return nil, errors.New("imou: property values are invalid or too large")
	}
	params := map[string]any{"token": accessToken, "deviceId": deviceID, "properties": properties}
	if channelID != "" { params["channelId"] = channelID }
	return c.call(ctx, "v2/device.property.set", params)
}

func (c *Client) InvokeService(ctx context.Context, accessToken, deviceID, channelID, serviceID string, input map[string]any) (json.RawMessage, error) {
	if !validUserToken(accessToken) || !validDeviceID(deviceID) || !validOptionalChannel(channelID) ||
		!validCapabilityID(serviceID) {
		return nil, errors.New("imou: invalid service invocation request")
	}
	if input == nil { input = map[string]any{} }
	encoded, err := json.Marshal(input)
	if err != nil || len(encoded) > 16*1024 {
		return nil, errors.New("imou: service input is invalid or too large")
	}
	params := map[string]any{"token": accessToken, "deviceId": deviceID, "serviceId": serviceID, "inputData": input}
	if channelID != "" { params["channelId"] = channelID }
	return c.call(ctx, "v2/device.service.invoke", params)
}

func (c *Client) ListMessageHistory(ctx context.Context, accessToken, deviceID, channelID string, begin, end time.Time, cursor string) (*MessagePage, error) {
	if !validUserToken(accessToken) || !validDeviceID(deviceID) || !validOptionalChannel(channelID) ||
		begin.IsZero() || end.IsZero() || !end.After(begin) || end.Sub(begin) > 31*24*time.Hour {
		return nil, errors.New("imou: invalid message-history query")
	}
	params := map[string]any{
		"token": accessToken, "deviceId": deviceID,
		"beginTime": begin.Format("20060102T150405"),
		"endTime": end.Format("20060102T150405"),
	}
	if channelID != "" { params["channelId"] = channelID }
	if cursor != "" {
		if len(cursor) > 128 || strings.ContainsAny(cursor, "\r\n\t ") { return nil, errors.New("imou: invalid message cursor") }
		params["nextAlarmId"] = cursor
	} else {
		params["nextAlarmId"] = "-1"
	}
	data, err := c.call(ctx, "v2/device.message.list", params)
	if err != nil { return nil, err }
	var page MessagePage
	if err := json.Unmarshal(data, &page); err != nil { return nil, errors.New("imou: invalid message-history response") }
	if page.Messages == nil { page.Messages = []DeviceMessage{} }
	return &page, nil
}

func (c *Client) ListLocalRecordings(ctx context.Context, accessToken, deviceID, channelID string, begin, end time.Time, recordType, limit int) ([]Recording, string, error) {
	if !validUserToken(accessToken) || !validDeviceID(deviceID) || !validRequiredChannel(channelID) ||
		begin.IsZero() || end.IsZero() || !end.After(begin) || end.Sub(begin) > 7*24*time.Hour ||
		recordType < 0 || recordType > 21 || recordType == 16 || limit < 30 || limit > 100 {
		return nil, "", errors.New("imou: invalid local-recording query")
	}
	input := map[string]any{
		"type": strconv.Itoa(recordType),
		"beginTime": begin.Format("2006-01-02 15:04:05"),
		"endTime": end.Format("2006-01-02 15:04:05"),
		"limit": limit,
	}
	data, err := c.InvokeService(ctx, accessToken, deviceID, channelID, "record.file.list", input)
	if err != nil { return nil, "", err }
	var result struct {
		OutputData struct {
			Records []Recording `json:"records"`
			SessionID string `json:"sessionId"`
		} `json:"outputData"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, "", errors.New("imou: invalid local-recording response")
	}
	if result.OutputData.Records == nil { result.OutputData.Records = []Recording{} }
	return result.OutputData.Records, result.OutputData.SessionID, nil
}

func (c *Client) ConfigureMessageCallback(ctx context.Context, adminToken, callbackURL string, enabled bool, flags []string) error {
	if !strings.HasPrefix(adminToken, "At_") || len(adminToken) > 256 {
		return errors.New("imou: administrator token is required to configure message push")
	}
	if !validHTTPSCallback(callbackURL) { return errors.New("imou: callback URL must be absolute HTTPS") }
	allowed := map[string]bool{"alarm": true, "deviceStatus": true, "numberstat": true, "faceAnalysis": true, "capabilityEvent": true, "iot": true}
	unique := make(map[string]bool)
	clean := make([]string, 0, len(flags))
	for _, flag := range flags {
		if !allowed[flag] { return fmt.Errorf("imou: unsupported callback flag %q", flag) }
		if !unique[flag] { unique[flag] = true; clean = append(clean, flag) }
	}
	status := "off"
	if enabled {
		if len(clean) == 0 { return errors.New("imou: at least one message callback flag is required") }
		status = "on"
	}
	_, err := c.call(ctx, "setMessageCallback", map[string]any{
		"token": adminToken, "status": status, "callbackUrl": callbackURL,
		"callbackFlag": strings.Join(clean, ","), "basePush": "2",
	})
	return err
}

func (c *Client) MessageCallbackStatus(ctx context.Context, adminToken string) (json.RawMessage, error) {
	if !strings.HasPrefix(adminToken, "At_") { return nil, errors.New("imou: administrator token is required") }
	return c.call(ctx, "getMessageCallback", map[string]any{"token": adminToken})
}

func validUserToken(token string) bool {
	return strings.HasPrefix(token, "Ot_") && len(token) <= 4096
}
func validDeviceID(id string) bool {
	return id != "" && len(id) <= 128 && !strings.ContainsAny(id, "/\\?#\r\n\t ")
}
func validOptionalChannel(channel string) bool {
	if channel == "" { return true }
	return validRequiredChannel(channel)
}
func validRequiredChannel(channel string) bool {
	n, err := strconv.Atoi(channel)
	return err == nil && n >= 0 && n <= 255
}
func validCapabilityID(id string) bool {
	if id == "" || len(id) > 160 { return false }
	for _, part := range strings.Split(id, ".") {
		if part == "" { return false }
		for _, ch := range part {
			if !(ch >= 'a' && ch <= 'z') && !(ch >= 'A' && ch <= 'Z') &&
				!(ch >= '0' && ch <= '9') && ch != '_' && ch != '-' { return false }
		}
	}
	return true
}
func validHTTPSCallback(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil && u.Fragment == ""
}
